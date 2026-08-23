package engine

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/xavskye/gorag/internal/hybrid"
	"github.com/xavskye/gorag/internal/index/flat"
	"github.com/xavskye/gorag/internal/index/hnsw"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

type EvalResult struct {
	N           int     `json:"n"`
	Dim         int     `json:"dim"`
	Queries     int     `json:"queries"`
	K           int     `json:"k"`
	RecallAtK   float64 `json:"recall_at_k"`
	HNSWP50MS   float64 `json:"hnsw_p50_ms"`
	HNSWP99MS   float64 `json:"hnsw_p99_ms"`
	FLATAccel   float64 `json:"flat_accel"`
	InsertVPS   float64 `json:"insert_vec_per_s"`
	PassRecall  bool    `json:"pass_recall"`
	PassLatency bool    `json:"pass_latency"`
}

func (e *Engine) EvalRecall(n, queries, k int) (*EvalResult, error) {
	if n <= 0 {
		n = 2000
	}
	if n > 10000 {
		n = 10000
	}
	if queries <= 0 {
		queries = 40
	}
	if k <= 0 {
		k = 10
	}
	dim := model.DefaultDim
	rng := rand.New(rand.NewSource(20260823))
	vecs := make([][]float32, n)
	// 混合：随机 + 聚簇，避免全随机让 HNSW 召回虚高/虚低。
	centers := make([][]float32, 12)
	for i := range centers {
		centers[i] = randUnit(rng, dim)
	}
	for i := 0; i < n; i++ {
		if i%5 == 0 {
			vecs[i] = randUnit(rng, dim)
			continue
		}
		c := centers[i%len(centers)]
		noise := randUnit(rng, dim)
		mix := make([]float32, dim)
		for d := 0; d < dim; d++ {
			mix[d] = c[d]*0.85 + noise[d]*0.15
		}
		vecs[i] = metric.L2Normalize(mix)
	}
	h := hnsw.New(dim, model.MetricCosine, hnsw.Params{
		M: e.Cfg.HNSWM, EfConstruction: e.Cfg.HNSWEfConstruction, EfSearch: e.Cfg.HNSWEfSearch,
	})
	f := flat.New(dim, model.MetricCosine)
	t0 := time.Now()
	for i, v := range vecs {
		id := uint64(i + 1)
		if err := h.Insert(id, v); err != nil {
			return nil, err
		}
		if err := f.Insert(id, v); err != nil {
			return nil, err
		}
	}
	insertVPS := float64(n) / time.Since(t0).Seconds()

	var recSum float64
	var hlat []time.Duration
	for q := 0; q < queries; q++ {
		qv := vecs[rng.Intn(n)]
		ts := time.Now()
		hs := h.Search(qv, k, e.Cfg.HNSWEfSearch)
		hlat = append(hlat, time.Since(ts))
		fs := f.Search(qv, k, model.MetricCosine)
		pred := make([]uint64, len(hs))
		truth := make([]uint64, len(fs))
		for i, c := range hs {
			pred[i] = c.ID
		}
		for i, c := range fs {
			truth[i] = c.ID
		}
		recSum += hybrid.RecallAtK(truth, pred, k)
	}
	// FLAT 加速比：同一次查询并发 vs 串行
	qv := vecs[0]
	_ = f.SearchSerial(qv, k, model.MetricCosine)
	_ = f.Search(qv, k, model.MetricCosine)
	var parSamples []time.Duration
	var serSamples []time.Duration
	for i := 0; i < 8; i++ {
		t1 := time.Now()
		_ = f.SearchSerial(qv, 20, model.MetricCosine)
		serSamples = append(serSamples, time.Since(t1))
		t2 := time.Now()
		_ = f.Search(qv, 20, model.MetricCosine)
		parSamples = append(parSamples, time.Since(t2))
	}
	accel := median(serSamples).Seconds() / maxDuration(median(parSamples), time.Microsecond).Seconds()
	p50, p99 := pct(hlat, 0.50), pct(hlat, 0.99)
	recall := recSum / float64(queries)
	return &EvalResult{
		N: n, Dim: dim, Queries: queries, K: k,
		RecallAtK: recall, HNSWP50MS: p50, HNSWP99MS: p99,
		FLATAccel: accel, InsertVPS: insertVPS,
		PassRecall: recall >= 0.95, PassLatency: p99 < 50,
	}, nil
}

func randUnit(rng *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(rng.NormFloat64())
	}
	return metric.L2Normalize(v)
}

func pct(ds []time.Duration, p float64) float64 {
	if len(ds) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), ds...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * p)
	return float64(cp[idx].Microseconds()) / 1000
}

func median(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), ds...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[len(cp)/2]
}

func maxDuration(a, floor time.Duration) time.Duration {
	if a < floor {
		return floor
	}
	return a
}

func (e *Engine) String() string { return fmt.Sprintf("engine(%d)", len(e.ents)) }
