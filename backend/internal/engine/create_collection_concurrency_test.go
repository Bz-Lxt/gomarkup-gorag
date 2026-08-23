package engine_test

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
)

func TestConcurrentCreateCollectionRejectsDuplicate(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = t.TempDir()
	cfg.EmbeddingProvider = "local"
	cfg.VisionProvider = "local"
	cfg.LLMProvider = "mock"

	eng, err := engine.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eng.Close)

	const contenders = 4
	previousProcs := runtime.GOMAXPROCS(contenders)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousProcs) })

	collection := model.Collection{Name: "shared-" + strings.Repeat("x", 1<<20)}
	start := make(chan struct{})
	results := make(chan error, contenders)
	var ready sync.WaitGroup
	ready.Add(contenders)
	for range contenders {
		go func() {
			ready.Done()
			<-start
			results <- eng.CreateCollection(collection)
		}()
	}
	ready.Wait()
	close(start)

	succeeded, conflicted := 0, 0
	for range contenders {
		err := <-results
		switch {
		case err == nil:
			succeeded++
		case model.IsCode(err, model.CodeConflict):
			conflicted++
		default:
			t.Fatalf("unexpected create result: %v", err)
		}
	}
	if succeeded != 1 || conflicted != contenders-1 {
		t.Fatalf("concurrent duplicate creates: succeeded=%d conflicted=%d, want 1 and %d", succeeded, conflicted, contenders-1)
	}
}
