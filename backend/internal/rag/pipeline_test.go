package rag

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
)

func TestMockStreamContainsFlag(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	_, err = eng.IngestDocument(engine.IngestDocReq{Collection: "default", Title: "RRF", Content: "倒数排名融合把向量与关键词列表合并。"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, meta, err := Run(ctx, eng, Query{Question: "什么是融合", TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !meta.Mock {
		t.Fatal("expected mock")
	}
	var b strings.Builder
	for tok := range ch {
		b.WriteString(tok.Text)
	}
	if !strings.Contains(b.String(), "[MOCK]") {
		t.Fatalf("missing flag: %s", b.String())
	}
}

func TestEmptyQuestion(t *testing.T) {
	_, _, err := Run(context.Background(), nil, Query{})
	if err == nil {
		t.Fatal("expected validation")
	}
}
