package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/provider"
)

func TestOpenAILLMPropagatesHTTPTraceContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"answer\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	llm := provider.NewLLM(&config.Config{
		LLMProvider:   "openai",
		OpenAIBaseURL: upstream.URL,
		OpenAIAPIKey:  "test-key",
		OpenAILLMModel: "test-model",
	}, cost.New(10))

	var gotConn atomic.Bool
	trace := &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) {
			gotConn.Store(true)
		},
	}
	ctx := httptrace.WithClientTrace(context.Background(), trace)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var text strings.Builder
	done := false
	for tok := range llm.Stream(ctx, "question", []string{"context"}) {
		if tok.Err != nil {
			t.Fatalf("stream failed: %v", tok.Err)
		}
		text.WriteString(tok.Text)
		done = done || tok.Done
	}

	if text.String() != "answer" {
		t.Fatalf("unexpected stream text %q", text.String())
	}
	if !done {
		t.Fatal("stream ended without done token")
	}
	if !gotConn.Load() {
		t.Fatal("outbound HTTP request did not preserve the caller's trace context")
	}
}
