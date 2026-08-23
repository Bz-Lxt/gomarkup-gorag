// Package hnsw 手写 Hierarchical NSW 增量图索引。
package hnsw

import (
	"math"
	"math/rand"
	"sync"

	"github.com/xavskye/gorag/internal/index"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

type node struct {
	id      uint64
	vec     []float32
	level   int
	friends [][]uint64
	deleted bool
}

type Index struct {
	dim            int
	metric         model.Metric
	m              int
	mMax           int
	mMax0          int
	efConstruction int
	efSearch       int
	ml             float64
	enter          uint64
	maxLevel       int
	nodes          map[uint64]*node
	order          []uint64
	rng            *rand.Rand
	mu             sync.RWMutex
}

type Params struct {
	M              int
	EfConstruction int
	EfSearch       int
}

func DefaultParams() Params {
	return Params{M: 16, EfConstruction: 200, EfSearch: 64}
}

func New(dim int, m model.Metric, p Params) *Index {
	if p.M < 4 {
		p.M = 16
	}
	if p.EfConstruction < 16 {
		p.EfConstruction = 200
	}
	if p.EfSearch < 8 {
		p.EfSearch = 64
	}
	return &Index{
		dim:            dim,
		metric:         m,
		m:              p.M,
		mMax:           p.M,
		mMax0:          p.M * 2,
		efConstruction: p.EfConstruction,
		efSearch:       p.EfSearch,
		ml:             1.0 / math.Log(float64(p.M)),
		nodes:          make(map[uint64]*node),
		rng:            rand.New(rand.NewSource(42)),
	}
}

func (idx *Index) SetEfSearch(ef int) {
	if ef < 1 {
		ef = 1
	}
	idx.mu.Lock()
	idx.efSearch = ef
	idx.mu.Unlock()
}

func (idx *Index) Params() (m, efc, efs int) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.m, idx.efConstruction, idx.efSearch
}

func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	n := 0
	for _, nd := range idx.nodes {
		if !nd.deleted {
			n++
		}
	}
	return n
}

func (idx *Index) randomLevel() int {
	r := idx.rng.Float64()
	if r < 1e-12 {
		r = 1e-12
	}
	return int(math.Floor(-math.Log(r) * idx.ml))
}

func (idx *Index) dist(a, b []float32) float64 {
	return metric.Distance(a, b, idx.metric)
}

func (idx *Index) Insert(id uint64, vec []float32) error {
	if err := metric.ValidateDim(vec, idx.dim); err != nil {
		return err
	}
	cp := append([]float32(nil), vec...)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if old, ok := idx.nodes[id]; ok {
		old.vec = cp
		old.deleted = false
		return nil
	}
	level := idx.randomLevel()
	nd := &node{id: id, vec: cp, level: level, friends: make([][]uint64, level+1)}
	if len(idx.nodes) == 0 {
		idx.nodes[id] = nd
		idx.order = append(idx.order, id)
		idx.enter = id
		idx.maxLevel = level
		return nil
	}
	// 先入表，保证后续回边可解析
	idx.nodes[id] = nd
	idx.order = append(idx.order, id)

	curr := idx.pickEnter()
	for lc := idx.maxLevel; lc > level; lc-- {
		near := idx.searchLayer(cp, []uint64{curr}, 1, lc)
		if len(near) > 0 {
			curr = near[0].ID
		}
	}
	for lc := min(level, idx.maxLevel); lc >= 0; lc-- {
		cands := idx.searchLayer(cp, []uint64{curr}, idx.efConstruction, lc)
		neighbors := selectHeuristic(cp, cands, idx.maxAt(lc), idx)
		nd.friends[lc] = uniqueIDs(idsOf(neighbors))
		for _, nb := range nd.friends[lc] {
			on := idx.nodes[nb]
			if on == nil || nb == id {
				continue
			}
			on.ensureLayer(lc)
			on.friends[lc] = uniqueIDs(append(on.friends[lc], id))
			if len(on.friends[lc]) > idx.maxAt(lc) {
				on.friends[lc] = idx.prune(on, lc)
			}
		}
		if len(cands) > 0 {
			curr = cands[0].ID
		}
	}
	if level > idx.maxLevel {
		idx.maxLevel = level
		idx.enter = id
	}
	return nil
}

func (idx *Index) Delete(id uint64) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	nd, ok := idx.nodes[id]
	if !ok {
		return false
	}
	nd.deleted = true
	return true
}

func (idx *Index) Search(query []float32, k int, ef int) []index.Cand {
	if k <= 0 {
		k = 10
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if len(idx.nodes) == 0 {
		return nil
	}
	if ef < k {
		ef = k
	}
	if ef < idx.efSearch {
		ef = idx.efSearch
	}
	curr := idx.pickEnter()
	for lc := idx.maxLevel; lc > 0; lc-- {
		near := idx.searchLayer(query, []uint64{curr}, 1, lc)
		if len(near) > 0 {
			curr = near[0].ID
		}
	}
	cands := idx.searchLayer(query, []uint64{curr}, ef, 0)
	out := make([]index.Cand, 0, k)
	for _, c := range cands {
		if nd := idx.nodes[c.ID]; nd != nil && !nd.deleted {
			out = append(out, c)
			if len(out) >= k {
				break
			}
		}
	}
	return out
}

func (idx *Index) pickEnter() uint64 {
	if nd := idx.nodes[idx.enter]; nd != nil && !nd.deleted {
		return idx.enter
	}
	for _, id := range idx.order {
		if nd := idx.nodes[id]; nd != nil && !nd.deleted {
			return id
		}
	}
	return idx.enter
}

func (idx *Index) searchLayer(q []float32, eps []uint64, ef int, layer int) []index.Cand {
	if ef < 1 {
		ef = 1
	}
	visited := make(map[uint64]struct{}, ef*8)
	var cand, w []slItem
	pushStart := func(id uint64) {
		nd := idx.nodes[id]
		if nd == nil {
			return
		}
		if _, ok := visited[id]; ok {
			return
		}
		visited[id] = struct{}{}
		d := idx.dist(q, nd.vec)
		cand = append(cand, slItem{id, d})
		if !nd.deleted {
			w = append(w, slItem{id, d})
		}
	}
	for _, ep := range eps {
		pushStart(ep)
	}
	if len(w) == 0 && len(cand) == 0 {
		return nil
	}
	for len(cand) > 0 {
		ci := argMin(cand)
		c := cand[ci]
		cand = removeAt(cand, ci)
		if len(w) > 0 && c.d > w[argMax(w)].d {
			break
		}
		nd := idx.nodes[c.id]
		if nd == nil {
			continue
		}
		for _, nb := range nd.layer(layer) {
			if _, ok := visited[nb]; ok {
				continue
			}
			visited[nb] = struct{}{}
			on := idx.nodes[nb]
			if on == nil {
				continue
			}
			d := idx.dist(q, on.vec)
			worst := math.Inf(1)
			if len(w) > 0 {
				worst = w[argMax(w)].d
			}
			if d < worst || len(w) < ef {
				cand = append(cand, slItem{nb, d})
				if !on.deleted {
					w = append(w, slItem{nb, d})
					if len(w) > ef {
						w = removeAt(w, argMax(w))
					}
				}
			}
		}
	}
	out := make([]index.Cand, len(w))
	for i, it := range w {
		out[i] = index.Cand{ID: it.id, Dist: it.d}
	}
	return index.SortedByDist(out)
}

func (idx *Index) prune(nd *node, layer int) []uint64 {
	var cands []index.Cand
	for _, nb := range nd.friends[layer] {
		on := idx.nodes[nb]
		if on == nil || on.deleted || nb == nd.id {
			continue
		}
		cands = append(cands, index.Cand{ID: nb, Dist: idx.dist(nd.vec, on.vec)})
	}
	return idsOf(selectHeuristic(nd.vec, cands, idx.maxAt(layer), idx))
}

func (idx *Index) maxAt(layer int) int {
	if layer == 0 {
		return idx.mMax0
	}
	return idx.mMax
}

func (n *node) layer(lc int) []uint64 {
	if n == nil || lc < 0 || lc >= len(n.friends) {
		return nil
	}
	return n.friends[lc]
}

func (n *node) ensureLayer(lc int) {
	for len(n.friends) <= lc {
		n.friends = append(n.friends, nil)
	}
}

func selectHeuristic(q []float32, cands []index.Cand, m int, idx *Index) []index.Cand {
	s := index.SortedByDist(cands)
	if len(s) <= m {
		return s
	}
	// 论文扩展启发式：优先保留彼此不太重复的近邻，提升连通性。
	var picked []index.Cand
	for _, c := range s {
		if len(picked) >= m {
			break
		}
		ok := true
		cv := idx.nodes[c.ID]
		if cv == nil {
			continue
		}
		for _, p := range picked {
			pv := idx.nodes[p.ID]
			if pv == nil {
				continue
			}
			if idx.dist(cv.vec, pv.vec) < c.Dist {
				ok = false
				break
			}
		}
		if ok {
			picked = append(picked, c)
		}
	}
	for _, c := range s {
		if len(picked) >= m {
			break
		}
		dup := false
		for _, p := range picked {
			if p.ID == c.ID {
				dup = true
				break
			}
		}
		if !dup {
			picked = append(picked, c)
		}
	}
	return picked
}

func idsOf(c []index.Cand) []uint64 {
	out := make([]uint64, len(c))
	for i, x := range c {
		out[i] = x.ID
	}
	return out
}

func uniqueIDs(in []uint64) []uint64 {
	seen := map[uint64]struct{}{}
	out := make([]uint64, 0, len(in))
	for _, id := range in {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func argMinItem[T any](s []T, dist func(T) float64) int {
	mi := 0
	for i := 1; i < len(s); i++ {
		if dist(s[i]) < dist(s[mi]) {
			mi = i
		}
	}
	return mi
}

type slItem struct {
	id uint64
	d  float64
}

func argMin(s []slItem) int {
	return argMinItem(s, func(it slItem) float64 { return it.d })
}

func argMax(s []slItem) int {
	mi := 0
	for i := 1; i < len(s); i++ {
		if s[i].d > s[mi].d {
			mi = i
		}
	}
	return mi
}

func removeAt(s []slItem, i int) []slItem {
	return append(s[:i], s[i+1:]...)
}
