package flat

import (
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

func TestTopKOrder(t *testing.T) {
	idx := New(4, model.MetricCosine)
	_ = idx.Insert(1, metric.L2Normalize([]float32{1, 0, 0, 0}))
	_ = idx.Insert(2, metric.L2Normalize([]float32{0.9, 0.1, 0, 0}))
	_ = idx.Insert(3, metric.L2Normalize([]float32{0, 1, 0, 0}))
	q := metric.L2Normalize([]float32{1, 0, 0, 0})
	got := idx.Search(q, 2, model.MetricCosine)
	if len(got) != 2 || got[0].ID != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestParallelMatchesSerial(t *testing.T) {
	idx := New(16, model.MetricCosine)
	for i := 0; i < 200; i++ {
		v := make([]float32, 16)
		v[i%16] = 1
		_ = idx.Insert(uint64(i+1), metric.L2Normalize(v))
	}
	q := metric.L2Normalize(append(make([]float32, 15), 1))
	a := idx.Search(q, 5, model.MetricCosine)
	b := idx.SearchSerial(q, 5, model.MetricCosine)
	if len(a) != len(b) {
		t.Fatalf("len %d vs %d", len(a), len(b))
	}
	set := map[uint64]struct{}{}
	for _, c := range b {
		set[c.ID] = struct{}{}
	}
	for _, c := range a {
		if _, ok := set[c.ID]; !ok {
			t.Fatalf("parallel id %d not in serial %+v vs %+v", c.ID, a, b)
		}
	}
	_ = time.Now()
}
