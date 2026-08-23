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

func TestMetaFallsBackToMockWhenOpenAIKeyMissing(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
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
	router := NewRouter(cfg, eng)

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"gorag123"}`))
	login.Header.Set("Content-Type", "application/json")
	loginOut := httptest.NewRecorder()
	router.ServeHTTP(loginOut, login)
	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginOut.Body).Decode(&loginEnv); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	req.Header.Set("Authorization", "Bearer "+loginEnv.Data.Token)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, req)
	if out.Code != http.StatusOK {
		t.Fatalf("meta status = %d, want %d", out.Code, http.StatusOK)
	}
	var env struct {
		Data struct {
			Providers map[string]string `json:"providers"`
		} `json:"data"`
	}
	if err := json.NewDecoder(out.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if got := env.Data.Providers["llm"]; got != "mock" {
		t.Fatalf("llm provider = %q, want mock fallback", got)
	}
}
