package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
)

func TestDeleteMissingCollectionPreservesNotFoundResponse(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/collections/missing", nil)
	req.SetPathValue("name", "missing")
	resp := httptest.NewRecorder()
	(&Handlers{Cfg: cfg, Eng: eng}).DeleteCollection(resp, req)

	var env Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusNotFound {
		t.Fatalf("delete missing collection returned HTTP %d, want %d (body: %+v)", resp.Code, http.StatusNotFound, env)
	}
	if env.Code != model.CodeNotFound {
		t.Fatalf("delete missing collection returned code %d, want %d", env.Code, model.CodeNotFound)
	}
}
