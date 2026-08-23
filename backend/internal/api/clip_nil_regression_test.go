package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
)

func TestHybridImageSearchWithoutCLIPReturnsDegradedResponse(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.VisionProvider = "local"
	cfg.CLIPBaseURL = ""
	cfg.CLIPAPIKey = ""

	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/hybrid", bytes.NewBufferString(`{"query":"red bicycle","modality":"image","top_k":5}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("hybrid search panicked without an optional CLIP provider: %v", recovered)
			}
		}()
		(&Handlers{Cfg: cfg, Eng: eng}).SearchHybrid(rec, req)
	}()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Code int `json:"code"`
		Data struct {
			CrossModal  bool   `json:"cross_modal"`
			DegradeNote string `json:"degrade_note"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 || response.Data.CrossModal || response.Data.DegradeNote == "" {
		t.Fatalf("expected scalar-channel degradation, got %+v", response)
	}
}
