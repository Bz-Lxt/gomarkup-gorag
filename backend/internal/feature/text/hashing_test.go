package text

import (
	"math"
	"testing"

	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

func TestDeterministic(t *testing.T) {
	a := Embed("多模态向量检索", model.DefaultDim)
	b := Embed("多模态向量检索", model.DefaultDim)
	if len(a) != model.DefaultDim {
		t.Fatalf("dim=%d", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("not deterministic")
		}
	}
	if math.Abs(metric.Dot(a, a)-1) > 1e-4 {
		t.Fatal("not normalized")
	}
}

func TestSimilarTextsCloser(t *testing.T) {
	q := Embed("向量检索系统", model.DefaultDim)
	a := Embed("这是一个向量检索与知识库系统", model.DefaultDim)
	b := Embed("今天午餐吃了西红柿炒鸡蛋", model.DefaultDim)
	if metric.Dot(q, a) <= metric.Dot(q, b) {
		t.Fatalf("sim(a)=%v sim(b)=%v", metric.Dot(q, a), metric.Dot(q, b))
	}
}
