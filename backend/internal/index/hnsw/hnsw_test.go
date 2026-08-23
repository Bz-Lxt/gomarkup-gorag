package hnsw

import (
	"math/rand"
	"testing"

	"github.com/xavskye/gorag/internal/index/flat"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

func TestExactSelfAmongMany(t *testing.T) {
	const n, dim = 80, 64
	h := New(dim, model.MetricCosine, DefaultParams())
	vecs := make([][]float32, n)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		for d := range v {
			v[d] = float32(rng.NormFloat64())
		}
		vecs[i] = metric.L2Normalize(v)
		if err := h.Insert(uint64(i+1), vecs[i]); err != nil {
			t.Fatal(err)
		}
	}
	miss := 0
	for i, v := range vecs {
		got := h.Search(v, 1, 64)
		if len(got) == 0 || got[0].ID != uint64(i+1) {
			miss++
		}
	}
	if miss > 5 {
		t.Fatalf("self-recall misses=%d/%d", miss, n)
	}
}

func TestInsertSearchSelf(t *testing.T) {
	idx := New(8, model.MetricCosine, DefaultParams())
	v := metric.L2Normalize([]float32{1, 0, 0, 0, 0, 0, 0, 0})
	if err := idx.Insert(7, v); err != nil {
		t.Fatal(err)
	}
	got := idx.Search(v, 1, 16)
	if len(got) != 1 || got[0].ID != 7 {
		t.Fatalf("%+v", got)
	}
}

func TestTombstone(t *testing.T) {
	idx := New(4, model.MetricCosine, DefaultParams())
	v := metric.L2Normalize([]float32{0, 1, 0, 0})
	_ = idx.Insert(1, v)
	idx.Delete(1)
	if len(idx.Search(v, 3, 8)) != 0 {
		t.Fatal("deleted still returned")
	}
}

func TestBetterThanRandomVsFlat(t *testing.T) {
	const n, dim = 120, 32
	h := New(dim, model.MetricCosine, Params{M: 16, EfConstruction: 120, EfSearch: 64})
	f := flat.New(dim, model.MetricCosine)
	for i := 0; i < n; i++ {
		v := metric.L2Normalize(makeRamp(dim, i*3+1))
		_ = h.Insert(uint64(i+1), v)
		_ = f.Insert(uint64(i+1), v)
	}
	q := metric.L2Normalize(makeRamp(dim, 7*3+1))
	hs := h.Search(q, 10, 64)
	fs := f.Search(q, 10, model.MetricCosine)
	if len(hs) == 0 || len(fs) == 0 {
		t.Fatal("empty")
	}
	set := map[uint64]struct{}{}
	for _, c := range fs {
		set[c.ID] = struct{}{}
	}
	hit := 0
	for _, c := range hs {
		if _, ok := set[c.ID]; ok {
			hit++
		}
	}
	if hit < 4 {
		t.Fatalf("overlap %d/10 hs=%+v fs=%+v", hit, hs, fs)
	}
}

func makeRamp(dim, seed int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32((i+seed)%17) / 17
	}
	return v
}
