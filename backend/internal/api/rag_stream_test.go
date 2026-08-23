package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
)

func TestRAGStreamCompletesAfterUpstreamDone(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"complete answer\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(upstream.Close)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.EmbeddingProvider = "local"
	cfg.LLMProvider = "openai"
	cfg.OpenAIBaseURL = upstream.URL
	cfg.OpenAIAPIKey = "test-key"
	cfg.OpenAILLMModel = "test-model"
	cfg.BudgetLimitCNY = 1
	cfg.DemoUser = "admin"
	cfg.DemoPass = "gorag123"

	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)

	srv := httptest.NewServer(NewRouter(cfg, eng))
	t.Cleanup(srv.Close)
	client := srv.Client()
	client.Timeout = 3 * time.Second

	loginReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"gorag123"}`))
	if err != nil {
		t.Fatal(err)
	}
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := client.Do(loginReq)
	if err != nil {
		t.Fatal(err)
	}
	defer loginResp.Body.Close()
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil {
		t.Fatal(err)
	}
	if login.Data.Token == "" {
		t.Fatalf("login returned no token (status %d)", loginResp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/rag/query", bytes.NewBufferString(`{"question":"is the stream complete?"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+login.Data.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read RAG stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RAG status %d: %s", resp.StatusCode, body)
	}
	stream := string(body)
	if !strings.Contains(stream, `"text":"complete answer"`) {
		t.Fatalf("RAG stream missing upstream token: %q", stream)
	}
	if !strings.Contains(stream, "event: done\ndata: {}") {
		t.Fatalf("RAG stream did not finish with done event: %q", stream)
	}
}
