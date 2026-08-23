// Package invert 自研倒排索引 + BM25。
package invert

import (
	"math"
	"sort"
	"sync"

	"github.com/xavskye/gorag/internal/tokenize"
)

type posting struct {
	ID  uint64
	TF  int
	Len int
}

type Index struct {
	mu        sync.RWMutex
	postings  map[string][]posting
	docLen    map[uint64]int
	deleted   map[uint64]struct{}
	docs      int
	totalLen  int
	k1        float64
	b         float64
}

func New() *Index {
	return &Index{
		postings: make(map[string][]posting),
		docLen:   make(map[uint64]int),
		deleted:  make(map[uint64]struct{}),
		k1:       1.5,
		b:        0.75,
	}
}

func (idx *Index) Add(id uint64, text string) {
	toks := tokenize.Tokenize(text)
	tf := map[string]int{}
	for _, t := range toks {
		tf[t.Term]++
	}
	dl := len(toks)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if old, ok := idx.docLen[id]; ok {
		idx.totalLen -= old
		idx.docs--
	}
	delete(idx.deleted, id)
	idx.docLen[id] = dl
	idx.totalLen += dl
	idx.docs++
	for term, n := range tf {
		idx.postings[term] = append(idx.postings[term], posting{ID: id, TF: n, Len: dl})
	}
}

func (idx *Index) Delete(id uint64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.deleted[id] = struct{}{}
}

type Hit struct {
	ID    uint64
	Score float64
}

func (idx *Index) Search(query string, topK int) []Hit {
	if topK <= 0 {
		topK = 10
	}
	qTerms := tokenize.Terms(query)
	if len(qTerms) == 0 {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	avgdl := 1.0
	if idx.docs > 0 {
		avgdl = float64(idx.totalLen) / float64(idx.docs)
	}
	scores := map[uint64]float64{}
	N := float64(idx.docs)
	if N < 1 {
		return nil
	}
	for _, term := range qTerms {
		plist := idx.postings[term]
		df := 0
		for _, p := range plist {
			if _, gone := idx.deleted[p.ID]; !gone {
				df++
			}
		}
		if df == 0 {
			continue
		}
		idf := math.Log((N-float64(df)+0.5)/(float64(df)+0.5) + 1)
		for _, p := range plist {
			if _, gone := idx.deleted[p.ID]; gone {
				continue
			}
			tf := float64(p.TF)
			dl := float64(p.Len)
			num := tf * (idx.k1 + 1)
			den := tf + idx.k1*(1-idx.b+idx.b*dl/avgdl)
			scores[p.ID] += idf * num / den
		}
	}
	hits := make([]Hit, 0, len(scores))
	for id, s := range scores {
		hits = append(hits, Hit{ID: id, Score: s})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits
}

func (idx *Index) DocCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docs - len(idx.deleted)
}
