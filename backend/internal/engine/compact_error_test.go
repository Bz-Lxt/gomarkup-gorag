package engine_test

import (
	"os"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
)

func TestCompactRejectsUnreadableInput(t *testing.T) {
	cfg := &config.Config{
		DataDir:            t.TempDir(),
		SegmentMaxBytes:    1 << 20,
		SegmentMaxRows:     100,
		SegmentMaxIdleSec:  60,
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
	t.Cleanup(eng.Close)

	var before []model.SegmentInfo
	for i, content := range []string{
		"first segment remains part of the compact input",
		"second segment provides another compact input",
	} {
		if _, err := eng.IngestDocument(engine.IngestDocReq{Content: content}); err != nil {
			t.Fatal(err)
		}
		if err := eng.Flush("test"); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			before = before[:0]
			for _, info := range eng.Stats().Segments {
				if info.State == model.SegPersisted {
					before = append(before, info)
				}
			}
			if len(before) == i+1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if len(before) != i+1 {
			t.Fatalf("persisted segments = %d, want %d", len(before), i+1)
		}
	}

	if err := os.WriteFile(before[0].FilePath, []byte("damaged segment"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := eng.Compact(); err == nil {
		t.Error("Compact returned nil with an unreadable input segment")
	}
	if got := len(eng.Stats().Segments); got != len(before) {
		t.Errorf("failed compact changed segment count from %d to %d", len(before), got)
	}
	if _, err := os.Stat(before[1].FilePath); err != nil {
		t.Errorf("failed compact removed the readable source segment: %v", err)
	}
}
