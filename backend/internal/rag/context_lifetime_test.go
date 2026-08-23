package rag_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/provider"
	"github.com/xavskye/gorag/internal/rag"
)

type gatedEchoLLM struct {
	release <-chan struct{}
}

func (g gatedEchoLLM) Name() string { return "gated-echo" }
func (g gatedEchoLLM) Kind() string { return "test" }

func (g gatedEchoLLM) Stream(ctx context.Context, _ string, contexts []string) <-chan provider.Token {
	ch := make(chan provider.Token, 2)
	go func() {
		defer close(ch)
		select {
		case <-g.release:
		case <-ctx.Done():
			ch <- provider.Token{Err: ctx.Err(), Done: true}
			return
		}
		ch <- provider.Token{Text: strings.Join(contexts, "\n")}
		ch <- provider.Token{Done: true}
	}()
	return ch
}

func TestRunKeepsRetrievedContextsUntilStreamConsumesThem(t *testing.T) {
	cfg := &config.Config{
		DataDir:            t.TempDir(),
		SegmentMaxBytes:    1 << 20,
		SegmentMaxRows:     100,
		SegmentMaxIdleSec:  3600,
		HNSWM:              16,
		HNSWEfConstruction: 200,
		HNSWEfSearch:       64,
		BudgetLimitCNY:     10,
		EmbeddingProvider:  "local",
		LLMProvider:        "mock",
	}
	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()

	const sentinel = "retrieved-context-sentinel"
	if _, err := eng.IngestDocument(engine.IngestDocReq{
		Collection: "default",
		Title:      "queued context",
		Content:    sentinel + " remains available to an asynchronous model",
	}); err != nil {
		t.Fatal(err)
	}

	release := make(chan struct{})
	eng.LLM = gatedEchoLLM{release: release}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, meta, err := rag.Run(ctx, eng, rag.Query{Question: sentinel, TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Citations) != 1 || !strings.Contains(meta.Citations[0].Hit.Content, sentinel) {
		close(release)
		t.Fatalf("retrieval metadata lost the matching context: %+v", meta.Citations)
	}

	close(release)
	var answer strings.Builder
	for tok := range stream {
		if tok.Err != nil {
			t.Fatalf("stream returned an error: %v", tok.Err)
		}
		answer.WriteString(tok.Text)
	}
	if !strings.Contains(answer.String(), sentinel) {
		t.Fatalf("stream received empty or stale retrieval context: %q", answer.String())
	}
}
