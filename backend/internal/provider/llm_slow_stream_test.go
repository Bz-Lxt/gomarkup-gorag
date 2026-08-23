package provider_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/provider"
)

func TestOpenAILLMKeepsSlowStreamUntilDone(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first-\"}}]}\n\n")
		flusher.Flush()

		timer := time.NewTimer(1200 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
			return
		case <-timer.C:
		}
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"second\"}}]}\n\ndata: [DONE]\n\n")
		flusher.Flush()
	}))
	t.Cleanup(upstream.Close)

	llm := provider.NewLLM(&config.Config{
		LLMProvider:     "openai",
		OpenAIBaseURL:    upstream.URL,
		OpenAIAPIKey:     "test-key",
		OpenAILLMModel:   "test-model",
	}, cost.New(100))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var text strings.Builder
	done := false
	for tok := range llm.Stream(ctx, "question", nil) {
		if tok.Err != nil {
			t.Fatalf("unexpected stream error: %v", tok.Err)
		}
		text.WriteString(tok.Text)
		done = done || tok.Done
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("caller context ended before the stream: %v", err)
	}
	if got, want := text.String(), "first-second"; got != want {
		t.Fatalf("stream reported done=%v with %q, want %q", done, got, want)
	}
	if !done {
		t.Fatal("stream closed without a done token")
	}
}
