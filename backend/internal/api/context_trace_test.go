package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/api"
	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRAGPreservesClientTraceWithRequestDeadline(t *testing.T) {
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	traceObserved := make(chan struct{}, 1)
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if trace := httptrace.ContextClientTrace(r.Context()); trace != nil && trace.WroteRequest != nil {
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				"data: {\"choices\":[{\"delta\":{\"content\":\"trace-ok\"}}]}\n\n" +
					"data: [DONE]\n\n",
			)),
			Request: r,
		}, nil
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.LLMProvider = "openai"
	cfg.OpenAIAPIKey = "test-key"
	cfg.OpenAIBaseURL = "http://llm.example"
	cfg.EmbeddingProvider = "local"
	cfg.DemoUser = "trace-user"
	cfg.DemoPass = "trace-pass"

	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	router := api.NewRouter(cfg, eng)

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(
		`{"username":"trace-user","password":"trace-pass"}`,
	))
	login.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginRec.Code, loginRec.Body.String())
	}
	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginEnv); err != nil {
		t.Fatal(err)
	}
	if loginEnv.Data.Token == "" {
		t.Fatal("login returned an empty token")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rag/query", bytes.NewBufferString(
		`{"question":"trace propagation"}`,
	))
	req.Header.Set("Authorization", "Bearer "+loginEnv.Data.Token)
	req.Header.Set("Content-Type", "application/json")
	ctx := httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		WroteRequest: func(httptrace.WroteRequestInfo) {
			select {
			case traceObserved <- struct{}{}:
			default:
			}
		},
	})
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("RAG status = %d, body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `event: token`) || !strings.Contains(body, `trace-ok`) || !strings.Contains(body, `event: done`) {
		t.Fatalf("unexpected RAG stream: %s", body)
	}
	select {
	case <-traceObserved:
	default:
		t.Fatal("outbound LLM request did not preserve the request's client trace")
	}
}
