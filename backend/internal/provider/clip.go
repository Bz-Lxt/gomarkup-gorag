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
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

type CLIP interface {
	Enabled() bool
	Name() string
	EmbedText(s string) ([]float32, error)
}

type disabledCLIP struct{}

func (disabledCLIP) Enabled() bool                      { return false }
func (disabledCLIP) Name() string                       { return "off" }
func (disabledCLIP) EmbedText(string) ([]float32, error) {
	return nil, model.NewError(model.CodeUnimplemented, "clip not configured")
}

type HTTPCLIP struct {
	base   string
	key    string
	client *http.Client
	ledger *cost.Ledger
}

func NewCLIP(cfg *config.Config, ledger *cost.Ledger) CLIP {
	if cfg.VisionProvider == "clip_api" && cfg.CLIPBaseURL != "" && cfg.CLIPAPIKey != "" {
		return &HTTPCLIP{
			base:   strings.TrimRight(cfg.CLIPBaseURL, "/"),
			key:    cfg.CLIPAPIKey,
			client: &http.Client{Timeout: 20 * time.Second},
			ledger: ledger,
		}
	}
	return disabledCLIP{}
}

func (c *HTTPCLIP) Enabled() bool { return true }
func (c *HTTPCLIP) Name() string  { return "clip_api" }

func (c *HTTPCLIP) EmbedText(s string) ([]float32, error) {
	if err := c.ledger.Allow(0.002); err != nil {
		return nil, err
	}
	var vec []float32
	err := DoNarrow(func() (Class, error) {
		body, _ := json.Marshal(map[string]any{"text": s})
		req, err := http.NewRequest(http.MethodPost, c.base+"/embed/text", bytes.NewReader(body))
		if err != nil {
			return ClassPermanent, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.client.Do(req)
		if err != nil {
			return ClassifyErr(err), err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return ClassifyStatus(resp.StatusCode), fmt.Errorf("clip http %d", resp.StatusCode)
		}
		var parsed struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return ClassPermanent, err
		}
		f32 := make([]float32, len(parsed.Embedding))
		for i, v := range parsed.Embedding {
			f32[i] = float32(v)
		}
		vec = metric.PadOrTrim(f32, model.DefaultDim)
		return ClassOK, nil
	})
	c.ledger.Record(model.CostRecord{Provider: "clip", Model: "clip_api", Tokens: 0, CNY: 0.002, OK: err == nil})
	if err != nil {
		return nil, model.Wrap(model.CodeProvider, "clip embed", err)
	}
	return vec, nil
}
