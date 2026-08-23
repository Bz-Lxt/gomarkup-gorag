package engine

import (
	"strings"
	"time"

	"github.com/xavskye/gorag/internal/filter"
	"github.com/xavskye/gorag/internal/hybrid"
	"github.com/xavskye/gorag/internal/index"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/tokenize"
	"github.com/xavskye/gorag/pkg/timeutil"
)

func (e *Engine) SearchText(req model.SearchRequest) (*model.SearchResponse, error) {
	start := time.Now()
	if err := e.normalizeReq(&req); err != nil {
		return nil, err
	}
	qvec, err := e.Embed.Embed(req.Query)
	if err != nil {
		return nil, err
	}
	return e.searchHybrid(req, qvec, false, start)
}

func (e *Engine) SearchImage(qvec []float32, req model.SearchRequest) (*model.SearchResponse, error) {
	start := time.Now()
	if err := e.normalizeReq(&req); err != nil {
		return nil, err
	}
	req.Modality = model.ModalityImage
	return e.searchHybrid(req, qvec, false, start)
}

func (e *Engine) SearchHybrid(req model.SearchRequest, qvec []float32) (*model.SearchResponse, error) {
	start := time.Now()
	if err := e.normalizeReq(&req); err != nil {
		return nil, err
	}
	cross := false
	note := ""
	if qvec == nil && req.Query != "" {
		if req.Modality == model.ModalityImage && e.CLIP.Enabled() {
			v, err := e.CLIP.EmbedText(req.Query)
			if err == nil {
				qvec = v
				cross = true
			} else {
				note = "clip failed, fallback to scalar channel"
			}
		}
		if qvec == nil {
			var err error
			qvec, err = e.Embed.Embed(req.Query)
			if err != nil {
				return nil, err
			}
		}
	}
	if req.Modality == model.ModalityImage && !e.CLIP.Enabled() {
		note = "cross_modal=false：未配置 CLIP，以文搜图走 caption/tag 标量通道"
	}
	resp, err := e.searchHybrid(req, qvec, cross, start)
	if err != nil {
		return nil, err
	}
	resp.CrossModal = cross
	resp.DegradeNote = note
	return resp, nil
}

func (e *Engine) normalizeReq(req *model.SearchRequest) error {
	if req.Collection == "" {
		req.Collection = "default"
	}
	e.mu.RLock()
	_, ok := e.cols[req.Collection]
	e.mu.RUnlock()
	if !ok {
		return model.NewError(model.CodeNotFound, "collection not found")
	}
	if req.TopK <= 0 || req.TopK > 200 {
		req.TopK = 12
	}
	if !req.Metric.Valid() {
		req.Metric = model.MetricCosine
	}
	if !req.IndexType.Valid() {
		req.IndexType = model.IndexHNSW
	}
	if req.RRFK <= 0 {
		req.RRFK = hybrid.DefaultK
	}
	if req.VectorW <= 0 {
		req.VectorW = 1
	}
	if req.KeywordW <= 0 {
		req.KeywordW = 1
	}
	return nil
}

func (e *Engine) searchHybrid(req model.SearchRequest, qvec []float32, cross bool, start time.Time) (*model.SearchResponse, error) {
	ast, err := filter.Parse(req.Filter)
	if err != nil {
		return nil, err
	}
	pool := req.TopK * 4
	if pool < 32 {
		pool = 32
	}
	var vcands []index.Cand
	if qvec != nil && (req.Modality != model.ModalityImage || cross || req.Query == "" || req.Modality == "") {
		if req.IndexType == model.IndexFLAT {
			vcands = e.flat.Search(qvec, pool, req.Metric)
		} else {
			if req.EfSearch > 0 {
				e.hnsw.SetEfSearch(req.EfSearch)
			}
			vcands = e.hnsw.Search(qvec, pool, req.EfSearch)
		}
	}
	var khits []hybrid.Ranked
	if req.Query != "" {
		khits = hybrid.FromInvert(e.inv.Search(req.Query, pool))
	}
	// 以文搜图且无 CLIP：仅保留图像实体的关键词命中，向量通道对图关闭
	useVector := qvec != nil
	if req.Modality == model.ModalityImage && !cross && req.Query != "" {
		useVector = false
	}
	var vr []hybrid.Ranked
	if useVector {
		vr = hybrid.FromIndex(vcands)
	}
	fused := hybrid.Fuse(req.RRFK, req.VectorW, req.KeywordW, vr, khits)
	hits := make([]model.SearchHit, 0, req.TopK)
	for _, f := range fused {
		ent := e.get(model.EntityID(f.ID))
		if ent == nil || ent.Deleted {
			continue
		}
		if req.Collection != "" && ent.Collection != req.Collection {
			continue
		}
		if req.Modality != "" && ent.Modality != req.Modality {
			continue
		}
		score := f.RRF
		if !filter.Match(ast, ent, score) {
			continue
		}
		hit := e.toHit(ent, f, qvec, req, cross)
		hits = append(hits, hit)
		if len(hits) >= req.TopK {
			break
		}
	}
	resp := &model.SearchResponse{
		Hits: hits, CrossModal: cross, TookMS: time.Since(start).Milliseconds(),
		Channels: channelNames(useVector, req.Query != ""),
	}
	if req.CompareFLAT && qvec != nil {
		gt := e.flat.Search(qvec, req.TopK, req.Metric)
		resp.FLATHits = make([]model.SearchHit, 0, len(gt))
		pred := make([]uint64, 0, len(hits))
		truth := make([]uint64, 0, len(gt))
		for _, c := range gt {
			ent := e.get(model.EntityID(c.ID))
			if ent == nil {
				continue
			}
			resp.FLATHits = append(resp.FLATHits, e.toHit(ent, hybrid.Fused{ID: c.ID, VecRank: len(truth) + 1}, qvec, req, cross))
			truth = append(truth, c.ID)
		}
		for _, h := range hits {
			pred = append(pred, uint64(h.ID))
		}
		resp.RecallAtK = hybrid.RecallAtK(truth, pred, req.TopK)
	}
	e.observe(time.Since(start))
	return resp, nil
}

func (e *Engine) toHit(ent *model.Entity, f hybrid.Fused, qvec []float32, req model.SearchRequest, cross bool) model.SearchHit {
	sim := f.RRF
	if qvec != nil && len(ent.Vector) > 0 {
		sim = metric.Similarity(qvec, ent.Vector, req.Metric)
	}
	hit := model.SearchHit{
		ID: ent.ID, Score: sim, Modality: ent.Modality,
		Channels:   model.ChannelScores{Vector: f.VecRank, Keyword: f.KeyRank, RRF: f.RRF},
		CrossModal: cross, Collection: ent.Collection, SourceRef: ent.SourceRef,
		Content: ent.Content, Caption: ent.Caption, Tags: ent.Tags,
	}
	if t, ok := ent.Scalar["title"].(string); ok {
		hit.Title = t
	}
	if ent.ContentHash != "" {
		hit.AssetURL = "/api/v1/assets/" + ent.ContentHash
	}
	hit.Evidence = e.evidence(ent, qvec, req.Query, req.Metric)
	return hit
}

func (e *Engine) evidence(ent *model.Entity, qvec []float32, query string, m model.Metric) model.Evidence {
	ev := model.Evidence{BBox: []model.BBoxEvidence{}, CharRanges: []model.CharRange{}}
	if ent.Modality == model.ModalityImage && qvec != nil && len(ent.Patches) > 0 {
		type sc struct {
			p model.Patch
			s float64
		}
		var ranked []sc
		for _, p := range ent.Patches {
			ranked = append(ranked, sc{p: p, s: metric.Similarity(qvec, p.Vector, m)})
		}
		for i := 1; i < len(ranked); i++ {
			j := i
			for j > 0 && ranked[j].s > ranked[j-1].s {
				ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
				j--
			}
		}
		top := 3
		if top > len(ranked) {
			top = len(ranked)
		}
		for i := 0; i < top; i++ {
			if ranked[i].s < 0.05 {
				continue
			}
			ev.BBox = append(ev.BBox, model.BBoxEvidence{Box: ranked[i].p.BBox, Score: ranked[i].s})
		}
	}
	if query != "" && ent.Content != "" {
		seen := map[[2]int]struct{}{}
		for _, tk := range tokenize.Tokenize(query) {
			for _, pos := range ent.Terms {
				if pos.Term == tk.Term {
					key := [2]int{pos.Start, pos.End}
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					ev.CharRanges = append(ev.CharRanges, model.CharRange{Start: pos.Start, End: pos.End, Kind: "term"})
				}
			}
		}
		if qvec != nil && len(ent.Sentences) > 0 {
			bestI, best := -1, -1.0
			for i, s := range ent.Sentences {
				sc := metric.Similarity(qvec, s.Vector, m)
				if sc > best {
					best, bestI = sc, i
				}
			}
			if bestI >= 0 {
				s := ent.Sentences[bestI]
				ev.CharRanges = append(ev.CharRanges, model.CharRange{Start: s.Start, End: s.End, Kind: "sentence"})
			}
		}
	}
	return ev
}

func channelNames(vec bool, key bool) []string {
	var out []string
	if vec {
		out = append(out, "vector")
	}
	if key {
		out = append(out, "keyword")
	}
	if len(out) == 2 {
		out = append(out, "rrf")
	}
	return out
}

func (e *Engine) SeedDemo() {
	e.mu.RLock()
	n := len(e.ents)
	e.mu.RUnlock()
	if n > 0 {
		return
	}
	docs := []IngestDocReq{
		{Collection: "default", Title: "混合检索入门", Tags: []string{"rag"}, Content: "GoRag 将 BM25 关键词检索与 HNSW 向量检索通过 RRF 融合。以文搜文返回字符级证据区间，以图搜图返回真实计算的 patch 边界框。"},
		{Collection: "default", Title: "Segment 管道", Tags: []string{"storage"}, Content: "写入先进入内存 Buffer。当达到字节、行数或空闲超时任一阈值，Goroutine 异步构建索引并持久化带 CRC 的 Segment。WAL 保证崩溃恢复零丢失。"},
		{Collection: "default", Title: "多模态边界", Tags: []string{"vision"}, Content: "本地图像特征是 HSV 直方图、感知哈希与边缘方向的拼接，表达视觉相似而非语义相似。以文搜图默认走 caption 与标签的标量通道，配置 CLIP 后才启用跨模态向量。"},
	}
	for _, d := range docs {
		_, _ = e.IngestDocument(d)
	}
	e.seedImages()
	_ = timeutil.Format(timeutil.Now())
	_ = strings.TrimSpace("ok")
}
