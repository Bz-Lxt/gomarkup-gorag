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

func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.DemoUser, cfg.DemoPass = "admin", "gorag123"
	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)
	srv := httptest.NewServer(NewRouter(cfg, eng))
	t.Cleanup(srv.Close)
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "gorag123"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var env Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(env.Data)
	var tok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(data, &tok)
	if tok.Token == "" {
		t.Fatalf("no token: %+v", env)
	}
	return srv, tok.Token
}

func TestHealthAndSearch(t *testing.T) {
	srv, token := testServer(t)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health %d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/documents", bytes.NewBufferString(`{"content":"混合向量检索与知识库问答","title":"t"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("ingest %d", resp.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/search/text", bytes.NewBufferString(`{"query":"向量检索","top_k":5}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("search %d", resp.StatusCode)
	}
}

func TestUnauthorized(t *testing.T) {
	srv, _ := testServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got %d", resp.StatusCode)
	}
}
