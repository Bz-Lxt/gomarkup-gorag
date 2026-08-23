package metric

import (
	"math"
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func TestDotSelfIsOneAfterNorm(t *testing.T) {
	v := L2Normalize([]float32{3, 4, 0})
	if math.Abs(Dot(v, v)-1) > 1e-5 {
		t.Fatalf("dot=%v", Dot(v, v))
	}
}

func TestCosineOrthogonal(t *testing.T) {
	a := L2Normalize([]float32{1, 0})
	b := L2Normalize([]float32{0, 1})
	if math.Abs(Dot(a, b)) > 1e-5 {
		t.Fatalf("expected 0, got %v", Dot(a, b))
	}
	if Distance(a, b, model.MetricCosine) < 0.99 {
		t.Fatalf("cosine dist=%v", Distance(a, b, model.MetricCosine))
	}
}

func TestL2Known(t *testing.T) {
	a := []float32{0, 0}
	b := []float32{3, 4}
	if math.Abs(L2(a, b)-5) > 1e-5 {
		t.Fatalf("l2=%v", L2(a, b))
	}
}

func TestValidateDim(t *testing.T) {
	if err := ValidateDim([]float32{1, 2}, 2); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDim([]float32{1}, 2); err == nil {
		t.Fatal("expected dim error")
	}
	if err := ValidateDim([]float32{float32(math.NaN())}, 1); err == nil {
		t.Fatal("expected nan error")
	}
}

func TestPadOrTrim(t *testing.T) {
	v := PadOrTrim([]float32{1, 0, 0}, 8)
	if len(v) != 8 {
		t.Fatalf("len=%d", len(v))
	}
	if math.Abs(Dot(v, v)-1) > 1e-4 {
		t.Fatalf("not normalized")
	}
}
