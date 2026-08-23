package engine

import (
	"path/filepath"
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/model"
)

func testCfg(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.SegmentMaxRows = 50
	cfg.SegmentMaxBytes = 1 << 20
	cfg.LogEnv = "development"
	return cfg
}

func TestIngestAndSearchRoundTrip(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	_, err = e.IngestDocument(IngestDocReq{
		Collection: "default", Title: "HNSW",
		Content: "分层小世界图用于近似最近邻检索，可与 BM25 做混合查询。",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := e.SearchText(model.SearchRequest{Collection: "default", Query: "近似最近邻 混合查询", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Hits) == 0 {
		t.Fatal("no hits")
	}
	if len(resp.Hits[0].Evidence.CharRanges) == 0 {
		t.Fatal("expected char evidence")
	}
}

func TestWALRecover(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(t)
	cfg.DataDir = dir
	e, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.IngestDocument(IngestDocReq{Collection: "default", Content: "WAL 保证崩溃恢复零丢失。向量检索仍然可用。"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Flush("test"); err != nil {
		t.Fatal(err)
	}
	e.Close()
	e2, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer e2.Close()
	resp, err := e2.SearchText(model.SearchRequest{Query: "崩溃恢复", TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Hits) == 0 {
		t.Fatalf("lost data after restart, dir=%s", filepath.Join(dir, "wal"))
	}
}

func TestEvalRecallSmall(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	res, err := e.EvalRecall(400, 20, 10)
	if err != nil {
		t.Fatal(err)
	}
	if res.RecallAtK < 0.90 {
		t.Fatalf("recall too low: %+v", res)
	}
}
