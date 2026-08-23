package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/feature/text"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

type Embedder interface {
	Name() string
	Kind() string // local | openai
	Embed(s string) ([]float32, error)
}

type LocalEmbedder struct{}

func (LocalEmbedder) Name() string { return "local-hashing" }
func (LocalEmbedder) Kind() string { return "local" }
func (LocalEmbedder) Embed(s string) ([]float32, error) {
	return text.Embed(s, model.DefaultDim), nil
}

type OpenAIEmbedder struct {
	base   string
	key    string
	model  string
	client *http.Client
	ledger *cost.Ledger
}

func NewEmbedder(cfg *config.Config, ledger *cost.Ledger) Embedder {
	if cfg.EmbeddingProvider == "openai" && cfg.OpenAIAPIKey != "" {
		return &OpenAIEmbedder{
			base:   strings.TrimRight(cfg.OpenAIBaseURL, "/"),
			key:    cfg.OpenAIAPIKey,
			model:  cfg.OpenAIEmbedModel,
			client: &http.Client{Timeout: 20 * time.Second},
			ledger: ledger,
		}
	}
	return LocalEmbedder{}
}

func (o *OpenAIEmbedder) Name() string { return o.model }
func (o *OpenAIEmbedder) Kind() string { return "openai" }

func (o *OpenAIEmbedder) Embed(s string) ([]float32, error) {
	if err := o.ledger.Allow(0.001); err != nil {
		return LocalEmbedder{}.Embed(s)
	}
	var vec []float32
	var tokens int
	err := DoNarrow(func() (Class, error) {
		body, _ := json.Marshal(map[string]any{"model": o.model, "input": s})
		req, err := http.NewRequest(http.MethodPost, o.base+"/embeddings", bytes.NewReader(body))
		if err != nil {
			return ClassPermanent, err
		}
		req.Header.Set("Authorization", "Bearer "+o.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := o.client.Do(req)
		if err != nil {
			return ClassifyErr(err), err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		cls := ClassifyStatus(resp.StatusCode)
		if resp.StatusCode >= 400 {
			return cls, fmt.Errorf("embed http %d", resp.StatusCode)
		}
		var parsed struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
			Usage struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return ClassPermanent, err
		}
		if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
			return ClassPermanent, fmt.Errorf("empty embedding")
		}
		src := parsed.Data[0].Embedding
		f32 := make([]float32, len(src))
		for i, v := range src {
			f32[i] = float32(v)
		}
		vec = metric.PadOrTrim(f32, model.DefaultDim)
		tokens = parsed.Usage.TotalTokens
		return ClassOK, nil
	})
	cny := float64(tokens) / 1_000_000 * 0.15 * 7.2
	o.ledger.Record(model.CostRecord{Provider: "openai-embed", Model: o.model, Tokens: tokens, CNY: cny, OK: err == nil})
	if err != nil {
		return nil, model.Wrap(model.CodeProvider, "openai embed", err)
	}
	return vec, nil
}
