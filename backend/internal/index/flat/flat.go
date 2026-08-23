// Package flat 实现并发暴力检索，作为 Ground Truth 与小 Segment 快路径。
package flat

import (
	"runtime"
	"sync"

	"github.com/xavskye/gorag/internal/index"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

type record struct {
	id      uint64
	vec     []float32
	deleted bool
}

type Index struct {
	dim    int
	metric model.Metric
	mu     sync.RWMutex
	items  []record
	byID   map[uint64]int
}

func New(dim int, m model.Metric) *Index {
	return &Index{dim: dim, metric: m, byID: make(map[uint64]int)}
}

func (idx *Index) Dim() int { return idx.dim }

func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	n := 0
	for _, it := range idx.items {
		if !it.deleted {
			n++
		}
	}
	return n
}

func (idx *Index) Insert(id uint64, vec []float32) error {
	if err := metric.ValidateDim(vec, idx.dim); err != nil {
		return err
	}
	// 防御性拷贝：避免调用方复用底层缓冲区时，旧记录被新写入覆盖，
	// 导致 CompareFLAT 与正常索引结果漂移、对照排名随新请求变化。
	cp := append([]float32(nil), vec...)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if pos, ok := idx.byID[id]; ok {
		idx.items[pos].vec = cp
		idx.items[pos].deleted = false
		return nil
	}
	idx.byID[id] = len(idx.items)
	idx.items = append(idx.items, record{id: id, vec: cp})
	return nil
}

func (idx *Index) Delete(id uint64) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	pos, ok := idx.byID[id]
	if !ok {
		return false
	}
	idx.items[pos].deleted = true
	return true
}

func (idx *Index) Get(id uint64) ([]float32, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	pos, ok := idx.byID[id]
	if !ok || idx.items[pos].deleted {
		return nil, false
	}
	return idx.items[pos].vec, true
}

// Search 并发计算全体距离，返回 Top-K（距离升序）。
func (idx *Index) Search(query []float32, k int, m model.Metric) []index.Cand {
	if k <= 0 {
		k = 10
	}
	if m == "" {
		m = idx.metric
	}
	idx.mu.RLock()
	items := idx.items
	idx.mu.RUnlock()
	if len(items) == 0 {
		return nil
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}
	type localTop struct{ c []index.Cand }
	parts := make([]localTop, workers)
	var wg sync.WaitGroup
	chunk := (len(items) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo := w * chunk
		hi := lo + chunk
		if lo >= len(items) {
			break
		}
		if hi > len(items) {
			hi = len(items)
		}
		wg.Add(1)
		go func(slot, a, b int) {
			defer wg.Done()
			top := make([]index.Cand, 0, k)
			for i := a; i < b; i++ {
				it := items[i]
				if it.deleted {
					continue
				}
				d := metric.Distance(query, it.vec, m)
				top = pushTop(top, index.Cand{ID: it.id, Dist: d}, k)
			}
			parts[slot].c = top
		}(w, lo, hi)
	}
	wg.Wait()
	merged := make([]index.Cand, 0, k*workers)
	for _, p := range parts {
		merged = append(merged, p.c...)
	}
	merged = index.SortedByDist(merged)
	if len(merged) > k {
		merged = merged[:k]
	}
	return merged
}

// SearchSerial 单 goroutine 基线，用于加速比测量。
func (idx *Index) SearchSerial(query []float32, k int, m model.Metric) []index.Cand {
	if k <= 0 {
		k = 10
	}
	if m == "" {
		m = idx.metric
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	top := make([]index.Cand, 0, k)
	for _, it := range idx.items {
		if it.deleted {
			continue
		}
		d := metric.Distance(query, it.vec, m)
		top = pushTop(top, index.Cand{ID: it.id, Dist: d}, k)
	}
	return index.SortedByDist(top)
}

func pushTop(top []index.Cand, c index.Cand, k int) []index.Cand {
	if len(top) < k {
		top = append(top, c)
		return top
	}
	worst := 0
	for i := 1; i < len(top); i++ {
		if top[i].Dist > top[worst].Dist {
			worst = i
		}
	}
	if c.Dist < top[worst].Dist {
		top[worst] = c
	}
	return top
}
