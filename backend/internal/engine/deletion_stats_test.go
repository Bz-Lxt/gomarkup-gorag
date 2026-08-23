package engine_test

import (
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
)

func TestStatsRemainAvailableAfterDeletingPopulatedCollection(t *testing.T) {
	cfg := &config.Config{
		DataDir:            t.TempDir(),
		SegmentMaxBytes:    1 << 20,
		SegmentMaxRows:     1000,
		SegmentMaxIdleSec:  3600,
		HNSWM:              16,
		HNSWEfConstruction: 200,
		HNSWEfSearch:       64,
		PatchGrid:          3,
		BudgetLimitCNY:     10,
		EmbeddingProvider:  "local",
		VisionProvider:     "local",
		LLMProvider:        "mock",
		MaxUploadBytes:     10 << 20,
		MaxImagePixels:     25_000_000,
	}

	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()

	if err := eng.CreateCollection(model.Collection{Name: "archive"}); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.IngestDocument(engine.IngestDocReq{
		Collection: "archive",
		Title:      "Retention",
		Content:    "Archived records remain searchable until their collection is removed.",
	}); err != nil {
		t.Fatal(err)
	}

	before := eng.Stats()
	if before.Entities != 1 || before.Vectors != 1 {
		t.Fatalf("unexpected populated stats: entities=%d vectors=%d", before.Entities, before.Vectors)
	}

	if err := eng.DeleteCollection("archive"); err != nil {
		t.Fatal(err)
	}
	after := eng.Stats()
	if after.Entities != 0 || after.Vectors != 0 {
		t.Fatalf("deleted collection remains in stats: entities=%d vectors=%d", after.Entities, after.Vectors)
	}
}
