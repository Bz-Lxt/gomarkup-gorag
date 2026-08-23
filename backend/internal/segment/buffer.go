package segment

import (
	"sync"
	"time"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type Buffer struct {
	mu       sync.Mutex
	id       uint64
	ents     []model.Entity
	bytes    int64
	maxBytes int64
	maxRows  int
	maxIdle  time.Duration
	last     time.Time
	minTS    int64
	maxTS    int64
}

func NewBuffer(id uint64, maxBytes int64, maxRows int, maxIdle time.Duration) *Buffer {
	now := timeutil.UnixMilli()
	return &Buffer{
		id: id, maxBytes: maxBytes, maxRows: maxRows, maxIdle: maxIdle,
		last: timeutil.Now(), minTS: now, maxTS: now,
	}
}

func (b *Buffer) Append(e model.Entity) (full bool, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ents = append(b.ents, e)
	b.bytes += estimateBytes(e)
	ts := timeutil.UnixMilli()
	if b.minTS == 0 || ts < b.minTS {
		b.minTS = ts
	}
	b.maxTS = ts
	b.last = timeutil.Now()
	if b.bytes >= b.maxBytes {
		return true, "max_bytes"
	}
	if len(b.ents) >= b.maxRows {
		return true, "max_rows"
	}
	return false, ""
}

func (b *Buffer) IdleDue() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.ents) == 0 {
		return false
	}
	return timeutil.Now().Sub(b.last) >= b.maxIdle
}

func (b *Buffer) Snapshot() (id uint64, ents []model.Entity, bytes int64, minTS, maxTS int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ents = append([]model.Entity(nil), b.ents...)
	return b.id, ents, b.bytes, b.minTS, b.maxTS
}

func (b *Buffer) Reset(nextID uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.id = nextID
	b.ents = nil
	b.bytes = 0
	now := timeutil.UnixMilli()
	b.minTS, b.maxTS = now, now
	b.last = timeutil.Now()
}

func (b *Buffer) Empty() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.ents) == 0
}

func estimateBytes(e model.Entity) int64 {
	n := int64(len(e.Vector) * 4)
	n += int64(len(e.Content) + len(e.Caption) + len(e.ContentHash))
	for _, p := range e.Patches {
		n += int64(len(p.Vector) * 4)
	}
	return n + 128
}
