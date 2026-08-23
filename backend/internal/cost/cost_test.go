package cost

import (
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func TestBudgetBlocks(t *testing.T) {
	l := New(1.0)
	if err := l.Allow(0.2); err != nil {
		t.Fatal(err)
	}
	l.Record(model.CostRecord{Provider: "openai", CNY: 0.9, OK: true})
	if err := l.Allow(0.2); err == nil {
		t.Fatal("expected budget error")
	}
}
