// Package hybrid 实现查询规划与倒数排名融合（RRF）。
package hybrid

import (
	"sort"

	"github.com/xavskye/gorag/internal/index"
	"github.com/xavskye/gorag/internal/invert"
)

const DefaultK = 60

type Ranked struct {
	ID    uint64
	Rank  int // 1-based
	Score float64
}

type Fused struct {
	ID       uint64
	RRF      float64
	VecRank  int
	KeyRank  int
	VecScore float64
	KeyScore float64
}

func FromIndex(cands []index.Cand) []Ranked {
	out := make([]Ranked, len(cands))
	for i, c := range cands {
		out[i] = Ranked{ID: c.ID, Rank: i + 1, Score: c.Dist}
	}
	return out
}

func FromInvert(hits []invert.Hit) []Ranked {
	out := make([]Ranked, len(hits))
	for i, h := range hits {
		out[i] = Ranked{ID: h.ID, Rank: i + 1, Score: h.Score}
	}
	return out
}

// Fuse 将多路排名按 RRF 合并。k 默认 60；权重按通道相乘。
func Fuse(k int, vectorW, keywordW float64, vector, keyword []Ranked) []Fused {
	if k <= 0 {
		k = DefaultK
	}
	if vectorW <= 0 {
		vectorW = 1
	}
	if keywordW <= 0 {
		keywordW = 1
	}
	type acc struct {
		rrf      float64
		vecRank  int
		keyRank  int
		vecScore float64
		keyScore float64
	}
	m := map[uint64]*acc{}
	add := func(list []Ranked, w float64, kind string) {
		for _, r := range list {
			if r.Rank <= 0 {
				continue
			}
			a := m[r.ID]
			if a == nil {
				a = &acc{}
				m[r.ID] = a
			}
			a.rrf += w / (float64(k) + float64(r.Rank))
			if kind == "v" {
				a.vecRank = r.Rank
				a.vecScore = r.Score
			} else {
				a.keyRank = r.Rank
				a.keyScore = r.Score
			}
		}
	}
	add(vector, vectorW, "v")
	add(keyword, keywordW, "k")
	out := make([]Fused, 0, len(m))
	for id, a := range m {
		out = append(out, Fused{
			ID: id, RRF: a.rrf, VecRank: a.vecRank, KeyRank: a.keyRank,
			VecScore: a.vecScore, KeyScore: a.keyScore,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RRF == out[j].RRF {
			return out[i].ID < out[j].ID
		}
		return out[i].RRF > out[j].RRF
	})
	return out
}

// RecallAtK 以 truth 为 Ground Truth，计算 pred 的 Recall@K。
func RecallAtK(truth, pred []uint64, k int) float64 {
	if k <= 0 {
		return 0
	}
	if len(truth) > k {
		truth = truth[:k]
	}
	if len(pred) > k {
		pred = pred[:k]
	}
	if len(truth) == 0 {
		return 1
	}
	set := map[uint64]struct{}{}
	for _, id := range pred {
		set[id] = struct{}{}
	}
	hit := 0
	for _, id := range truth {
		if _, ok := set[id]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(truth))
}
