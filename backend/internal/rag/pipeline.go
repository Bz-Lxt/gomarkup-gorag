package rag

import (
	"context"
	"fmt"

	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/provider"
)

type Query struct {
	Collection string `json:"collection"`
	Question   string `json:"question"`
	TopK       int    `json:"top_k"`
}

type Citation struct {
	Index int             `json:"index"`
	Hit   model.SearchHit `json:"hit"`
}

type ResultMeta struct {
	Mock      bool       `json:"mock"`
	Model     string     `json:"model"`
	Citations []Citation `json:"citations"`
	Estimate  float64    `json:"estimate_cny"`
}

func Run(ctx context.Context, eng *engine.Engine, q Query) (<-chan provider.Token, ResultMeta, error) {
	if q.Question == "" {
		return nil, ResultMeta{}, model.NewError(model.CodeValidation, "question required")
	}
	if q.TopK <= 0 {
		q.TopK = 6
	}
	resp, err := eng.SearchText(model.SearchRequest{
		Collection: q.Collection, Query: q.Question, TopK: q.TopK,
	})
	if err != nil {
		return nil, ResultMeta{}, err
	}
	ctxs := make([]string, 0, len(resp.Hits))
	cites := make([]Citation, 0, len(resp.Hits))
	for i, h := range resp.Hits {
		body := h.Content
		if body == "" {
			body = h.Caption
		}
		ctxs = append(ctxs, fmt.Sprintf("%s %s", h.Title, body))
		cites = append(cites, Citation{Index: i + 1, Hit: h})
	}
	est := 0.0
	if eng.LLM.Kind() != "mock" {
		est = 0.02
	}
	defer clear(ctxs)
	ch := eng.LLM.Stream(ctx, q.Question, ctxs)
	return ch, ResultMeta{Mock: eng.LLM.Kind() == "mock", Model: eng.LLM.Name(), Citations: cites, Estimate: est}, nil
}
