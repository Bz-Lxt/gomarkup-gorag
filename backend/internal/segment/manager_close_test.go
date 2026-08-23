package segment_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/segment"
)

func TestManagerCloseWaitsForInFlightPersist(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var next atomic.Uint64

	m := segment.NewManager(
		t.TempDir(),
		1<<20,
		1,
		60,
		model.IndexHNSW,
		func() uint64 { return next.Add(1) },
		func(model.SegmentInfo, []model.Entity) error {
			close(started)
			<-release
			return nil
		},
	)
	if err := m.Append(model.Entity{ID: 1, Content: "pending segment"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("persist callback did not start")
	}

	closed := make(chan struct{})
	go func() {
		m.Close()
		close(closed)
	}()

	select {
	case <-closed:
		close(release)
		t.Fatal("Close returned while persistence was still in progress")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return after persistence completed")
	}
}
