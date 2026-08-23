package cost

import (
	"sync"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/logger"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type Ledger struct {
	mu     sync.Mutex
	limit  float64
	spent  float64
	items  []model.CostRecord
}

func New(limit float64) *Ledger {
	return &Ledger{limit: limit}
}

func (l *Ledger) Allow(estimate float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.spent+estimate > l.limit {
		return model.NewError(model.CodeBudgetExceeded, "budget limit exceeded, fallback to mock")
	}
	return nil
}

func (l *Ledger) Record(rec model.CostRecord) {
	if rec.At.IsZero() {
		rec.At = timeutil.NowNaive()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, rec)
	if rec.OK {
		l.spent += rec.CNY
	}
	logger.Info("cost.record", "provider", rec.Provider, "cny", rec.CNY, "ok", rec.OK)
}

func (l *Ledger) Spent() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.spent
}

func (l *Ledger) Limit() float64 { return l.limit }

func (l *Ledger) List() []model.CostRecord {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]model.CostRecord, len(l.items))
	copy(out, l.items)
	return out
}
