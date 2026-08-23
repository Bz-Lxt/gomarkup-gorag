package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/index/flat"
	"github.com/xavskye/gorag/internal/index/hnsw"
	"github.com/xavskye/gorag/internal/invert"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/provider"
	"github.com/xavskye/gorag/internal/segment"
	"github.com/xavskye/gorag/internal/store"
	"github.com/xavskye/gorag/internal/wal"
	"github.com/xavskye/gorag/pkg/logger"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type Engine struct {
	Cfg     *config.Config
	Ledger  *cost.Ledger
	Embed   provider.Embedder
	CLIP    provider.CLIP
	LLM     provider.LLM
	Assets  *store.Assets
	Man     *store.Manifest
	WAL     *wal.Log
	Seg     *segment.Manager

	mu      sync.RWMutex
	cols    map[string]*model.Collection
	ents    map[model.EntityID]*model.Entity
	hnsw    *hnsw.Index
	flat    *flat.Index
	inv     *invert.Index
	lat     []time.Duration
	queries int64
	started time.Time
	recMS   int64
	ready   bool
}

func Open(cfg *config.Config) (*Engine, error) {
	for _, d := range []string{
		cfg.DataDir,
		filepath.Join(cfg.DataDir, "wal"),
		filepath.Join(cfg.DataDir, "segments"),
		filepath.Join(cfg.DataDir, "assets"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	man, err := store.LoadManifest(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	assets, err := store.NewAssets(filepath.Join(cfg.DataDir, "assets"))
	if err != nil {
		return nil, err
	}
	wlog, err := wal.Open(filepath.Join(cfg.DataDir, "wal"))
	if err != nil {
		return nil, err
	}
	ledger := cost.New(cfg.BudgetLimitCNY)
	e := &Engine{
		Cfg: cfg, Ledger: ledger,
		Embed: provider.NewEmbedder(cfg, ledger),
		CLIP:  provider.NewCLIP(cfg, ledger),
		LLM:   provider.NewLLM(cfg, ledger),
		Assets: assets, Man: man, WAL: wlog,
		cols: make(map[string]*model.Collection),
		ents: make(map[model.EntityID]*model.Entity),
		hnsw: hnsw.New(model.DefaultDim, model.MetricCosine, hnsw.Params{
			M: cfg.HNSWM, EfConstruction: cfg.HNSWEfConstruction, EfSearch: cfg.HNSWEfSearch,
		}),
		flat:    flat.New(model.DefaultDim, model.MetricCosine),
		inv:     invert.New(),
		started: timeutil.Now(),
	}
	e.Seg = segment.NewManager(
		filepath.Join(cfg.DataDir, "segments"),
		cfg.SegmentMaxBytes, cfg.SegmentMaxRows, cfg.SegmentMaxIdleSec,
		model.IndexHNSW, man.AllocSegment,
		func(info model.SegmentInfo, ents []model.Entity) error {
			e.mu.Lock()
			e.Man.Segments = append(e.Man.Segments, info)
			e.mu.Unlock()
			return e.Man.Save()
		},
	)
	start := time.Now()
	if err := e.recover(); err != nil {
		return nil, err
	}
	e.recMS = time.Since(start).Milliseconds()
	e.ready = true
	logger.Info("engine.ready", "entities", len(e.ents), "recover_ms", e.recMS)
	return e, nil
}

func (e *Engine) recover() error {
	// persisted segments first
	segDir := filepath.Join(e.Cfg.DataDir, "segments")
	ents, _ := os.ReadDir(segDir)
	for _, de := range ents {
		if de.IsDir() {
			continue
		}
		path := filepath.Join(segDir, de.Name())
		hdr, rows, crc, err := segment.ReadFile(path)
		if err != nil {
			logger.Warn("engine.skip_bad_segment", "path", path, "err", err)
			continue
		}
		e.Seg.Remember(model.SegmentInfo{
			ID: hdr.ID, State: model.SegPersisted, RowCount: int(hdr.RowCount),
			ByteSize: int64(hdr.ByteSize), IndexType: hdr.IndexType, FilePath: path,
			CRC32: crc, MinTS: hdr.MinTS, MaxTS: hdr.MaxTS,
		})
		for i := range rows {
			e.indexLocked(&rows[i], false)
		}
	}
	recs, err := e.WAL.Replay()
	if err != nil {
		return err
	}
	for _, rec := range recs {
		switch rec.Type {
		case wal.RecCollection:
			var c model.Collection
			if err := gobDecode(rec.Payload, &c); err != nil {
				return err
			}
			e.cols[c.Name] = &c
		case wal.RecUpsert:
			var ent model.Entity
			if err := gobDecode(rec.Payload, &ent); err != nil {
				return err
			}
			e.indexLocked(&ent, false)
		case wal.RecDelete:
			var id uint64
			if err := gobDecode(rec.Payload, &id); err != nil {
				return err
			}
			e.deleteLocked(model.EntityID(id), false)
		}
	}
	if len(e.cols) == 0 {
		_ = e.CreateCollection(model.Collection{
			Name: "default", Dim: model.DefaultDim, Metric: model.MetricCosine, IndexType: model.IndexHNSW,
		})
	}
	return nil
}

func (e *Engine) Close() {
	e.ready = false
	e.Seg.Close()
	_ = e.WAL.Close()
	_ = e.Man.Save()
}

func (e *Engine) Ready() bool { return e.ready }

func (e *Engine) CreateCollection(c model.Collection) error {
	if c.Name == "" {
		return model.NewError(model.CodeValidation, "collection name required")
	}
	if c.Dim == 0 {
		c.Dim = model.DefaultDim
	}
	if c.Dim != model.DefaultDim {
		return model.NewError(model.CodeDimMismatch, "only dim=1024 is supported")
	}
	if !c.Metric.Valid() {
		c.Metric = model.MetricCosine
	}
	if !c.IndexType.Valid() {
		c.IndexType = model.IndexHNSW
	}
	c.CreatedAt = timeutil.NowNaive()
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.cols[c.Name]; ok {
		return model.NewError(model.CodeConflict, "collection exists")
	}
	e.cols[c.Name] = &c
	e.Man.Collections = append(e.Man.Collections, c)
	payload, _ := gobEncode(c)
	if err := e.WAL.Append(wal.RecCollection, payload); err != nil {
		return err
	}
	return e.Man.Save()
}

func (e *Engine) ListCollections() []model.Collection {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]model.Collection, 0, len(e.cols))
	for _, c := range e.cols {
		out = append(out, *c)
	}
	return out
}

func (e *Engine) DeleteCollection(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.cols[name]; !ok {
		return model.NewError(model.CodeNotFound, "collection not found")
	}
	delete(e.cols, name)
	for id, ent := range e.ents {
		if ent.Collection == name {
			e.deleteLocked(id, true)
		}
	}
	kept := e.Man.Collections[:0]
	for _, c := range e.Man.Collections {
		if c.Name != name {
			kept = append(kept, c)
		}
	}
	e.Man.Collections = kept
	return e.Man.Save()
}

func (e *Engine) indexLocked(ent *model.Entity, persistWAL bool) {
	cp := *ent
	if cp.Scalar == nil {
		cp.Scalar = map[string]any{}
	}
	e.ents[cp.ID] = &cp
	if len(cp.Vector) == model.DefaultDim {
		_ = e.hnsw.Insert(uint64(cp.ID), cp.Vector)
		_ = e.flat.Insert(uint64(cp.ID), cp.Vector)
	}
	text := cp.Content
	if text == "" {
		text = cp.Caption + " " + joinTags(cp.Tags)
	}
	if text != "" {
		e.inv.Add(uint64(cp.ID), text)
	}
	if persistWAL {
		payload, _ := gobEncode(cp)
		_ = e.WAL.Append(wal.RecUpsert, payload)
		_ = e.Seg.Append(cp)
	}
}

func (e *Engine) deleteLocked(id model.EntityID, persistWAL bool) {
	ent, ok := e.ents[id]
	if !ok {
		return
	}
	ent.Deleted = true
	e.hnsw.Delete(uint64(id))
	e.flat.Delete(uint64(id))
	e.inv.Delete(uint64(id))
	if persistWAL {
		payload, _ := gobEncode(uint64(id))
		_ = e.WAL.Append(wal.RecDelete, payload)
	}
}

func (e *Engine) get(id model.EntityID) *model.Entity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ents[id]
}

func (e *Engine) Flush(reason string) error {
	return e.Seg.Flush(reason)
}

func (e *Engine) Compact() error {
	e.mu.Lock()
	segs := append([]model.SegmentInfo(nil), e.Man.Segments...)
	e.mu.Unlock()
	small := segs
	if len(small) < 2 {
		return nil
	}
	info, _, err := segment.CompactSmall(filepath.Join(e.Cfg.DataDir, "segments"), small, func(ent model.Entity) bool {
		cur := e.get(ent.ID)
		return cur != nil && !cur.Deleted
	}, e.Man.AllocSegment())
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.Man.Segments = []model.SegmentInfo{info}
	e.Seg.Compact([]model.SegmentInfo{info})
	e.mu.Unlock()
	return e.Man.Save()
}

func (e *Engine) observe(d time.Duration) {
	e.mu.Lock()
	e.queries++
	e.lat = append(e.lat, d)
	if len(e.lat) > 512 {
		e.lat = e.lat[len(e.lat)-512:]
	}
	e.mu.Unlock()
}

func (e *Engine) Stats() model.Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	var p50, p99 float64
	if n := len(e.lat); n > 0 {
		cp := append([]time.Duration(nil), e.lat...)
		// insertion sort
		for i := 1; i < len(cp); i++ {
			j := i
			for j > 0 && cp[j] < cp[j-1] {
				cp[j], cp[j-1] = cp[j-1], cp[j]
				j--
			}
		}
		p50 = float64(cp[n/2].Microseconds()) / 1000
		idx := int(float64(n)*0.99) - 1
		if idx < 0 {
			idx = 0
		}
		p99 = float64(cp[idx].Microseconds()) / 1000
	}
	elapsed := timeutil.Now().Sub(e.started).Seconds()
	qps := 0.0
	if elapsed > 0 {
		qps = float64(e.queries) / elapsed
	}
	m, efc, efs := e.hnsw.Params()
	nEnt, nPatch := 0, 0
	for _, ent := range e.ents {
		if ent.Deleted {
			continue
		}
		nEnt++
		nPatch += len(ent.Patches)
	}
	return model.Stats{
		Collections:  len(e.cols),
		Entities:     nEnt,
		Vectors:      e.hnsw.Len(),
		Patches:      nPatch,
		Segments:     e.Seg.List(),
		MemBytes:     int64(mem.Alloc),
		WALBytes:     e.WAL.Size(),
		FlushHistory: e.Seg.History(),
		QPS:          qps,
		LatencyP50MS: p50,
		LatencyP99MS: p99,
		RecoverMS:    e.recMS,
		CostCNY:      e.Ledger.Spent(),
		BudgetCNY:    e.Ledger.Limit(),
		Providers: map[string]string{
			"embedding": e.Embed.Kind(),
			"vision":    e.CLIP.Name(),
			"llm":       e.LLM.Kind(),
		},
		HNSWParams: map[string]int{"M": m, "efConstruction": efc, "efSearch": efs},
	}
}

func gobEncode(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobDecode(b []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(v)
}

func joinTags(tags []string) string {
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += " "
		}
		out += t
	}
	return out
}
