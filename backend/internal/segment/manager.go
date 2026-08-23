package segment

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/logger"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type PersistFunc func(info model.SegmentInfo, ents []model.Entity) error

type Manager struct {
	dir      string
	buf      *Buffer
	mu       sync.Mutex
	sealed   []model.SegmentInfo
	history  []model.FlushEvent
	persist  PersistFunc
	nextID   func() uint64
	indexTyp model.IndexType
	stop     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

func NewManager(dir string, maxBytes int64, maxRows int, idleSec int, indexTyp model.IndexType, nextID func() uint64, persist PersistFunc) *Manager {
	if idleSec < 1 {
		idleSec = 30
	}
	m := &Manager{
		dir: dir, persist: persist, nextID: nextID, indexTyp: indexTyp,
		stop: make(chan struct{}),
	}
	id := nextID()
	m.buf = NewBuffer(id, maxBytes, maxRows, time.Duration(idleSec)*time.Second)
	go m.idleLoop(time.Duration(idleSec) * time.Second)
	return m
}

func (m *Manager) idleLoop(d time.Duration) {
	t := time.NewTicker(d / 2)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-t.C:
			if m.buf.IdleDue() {
				_ = m.Flush("max_idle")
			}
		}
	}
}

func (m *Manager) Append(e model.Entity) error {
	full, reason := m.buf.Append(e)
	if full {
		return m.Flush(reason)
	}
	return nil
}

func (m *Manager) Flush(reason string) error {
	if m.buf.Empty() {
		return nil
	}
	id, ents, bytes, minTS, maxTS := m.buf.Snapshot()
	if len(ents) == 0 {
		return nil
	}
	m.buf.Reset(m.nextID())
	start := timeutil.Now()
	go func() {
		info := model.SegmentInfo{
			ID: id, State: model.SegSealed, RowCount: len(ents), ByteSize: bytes,
			IndexType: m.indexTyp, MinTS: minTS, MaxTS: maxTS,
			FilePath: filepath.Join(m.dir, fmt.Sprintf("seg_%d.bin", id)),
		}
		if err := m.buildAndPersist(info, ents); err != nil {
			logger.Error("segment.persist_fail", "id", id, "err", err)
			return
		}
		m.wg.Add(1)
		defer m.wg.Done()
		info.State = model.SegPersisted
		m.mu.Lock()
		m.sealed = append(m.sealed, info)
		m.history = append(m.history, model.FlushEvent{
			At: timeutil.NowNaive(), Segment: id, Rows: len(ents), Bytes: bytes,
			Reason: reason, Duration: time.Since(start).String(),
		})
		m.mu.Unlock()
		logger.Info("segment.persisted", "id", id, "rows", len(ents), "reason", reason)
	}()
	return nil
}

func (m *Manager) buildAndPersist(info model.SegmentInfo, ents []model.Entity) error {
	crc, err := WriteFile(info.FilePath, FileHeader{
		ID: info.ID, RowCount: uint32(info.RowCount), ByteSize: uint32(info.ByteSize),
		IndexType: info.IndexType, MinTS: info.MinTS, MaxTS: info.MaxTS,
	}, ents)
	if err != nil {
		return err
	}
	info.CRC32 = crc
	if m.persist != nil {
		return m.persist(info, ents)
	}
	return nil
}

func (m *Manager) List() []model.SegmentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ents, bytes, minTS, maxTS := m.buf.Snapshot()
	out := append([]model.SegmentInfo(nil), m.sealed...)
	if len(ents) > 0 {
		out = append(out, model.SegmentInfo{
			ID: id, State: model.SegGrowing, RowCount: len(ents), ByteSize: bytes,
			IndexType: m.indexTyp, MinTS: minTS, MaxTS: maxTS,
		})
	}
	return out
}

func (m *Manager) History() []model.FlushEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.FlushEvent, len(m.history))
	copy(out, m.history)
	return out
}

func (m *Manager) Remember(info model.SegmentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sealed = append(m.sealed, info)
}

func (m *Manager) Compact(keep []model.SegmentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sealed = append([]model.SegmentInfo(nil), keep...)
}

func (m *Manager) Close() {
	m.once.Do(func() {
		close(m.stop)
		_ = m.Flush("shutdown")
		m.wg.Wait()
	})
}
