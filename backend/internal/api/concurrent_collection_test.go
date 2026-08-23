package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/xavskye/gorag/internal/api"
	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
)

func TestConcurrentCollectionCreateHasSingleWinner(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousProcs) })

	cfg := &config.Config{
		DataDir:            t.TempDir(),
		SegmentMaxBytes:    1 << 20,
		SegmentMaxRows:     1000,
		SegmentMaxIdleSec:  3600,
		HNSWM:              16,
		HNSWEfConstruction: 200,
		HNSWEfSearch:       64,
		BudgetLimitCNY:     10,
		EmbeddingProvider:  "local",
		VisionProvider:     "local",
		LLMProvider:        "mock",
		DemoUser:           "concurrency-test",
		DemoPass:           "concurrency-test-password",
	}
	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	router := api.NewRouter(cfg, eng)

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(
		`{"username":"concurrency-test","password":"concurrency-test-password"}`,
	))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	router.ServeHTTP(loginResult, login)
	if loginResult.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResult.Code, loginResult.Body.String())
	}
	var session struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResult.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.Data.Token == "" {
		t.Fatal("login returned an empty token")
	}

	const callers = 24
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, callers)
	for i := 0; i < callers; i++ {
		go func() {
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/v1/collections", bytes.NewBufferString(
				`{"name":"shared-reports","dim":1024,"metric":"cosine","index_type":"hnsw"}`,
			))
			req.Header.Set("Authorization", "Bearer "+session.Data.Token)
			req.Header.Set("Content-Type", "application/json")
			result := httptest.NewRecorder()
			router.ServeHTTP(result, req)
			results <- result
		}()
	}
	close(start)

	created, conflicts := 0, 0
	for i := 0; i < callers; i++ {
		result := <-results
		switch result.Code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("create status = %d, body = %s", result.Code, result.Body.String())
		}
	}
	if created != 1 || conflicts != callers-1 {
		t.Fatalf("concurrent create results: created=%d conflicts=%d; want created=1 conflicts=%d", created, conflicts, callers-1)
	}
}
