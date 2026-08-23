// Package metric 实现 Cosine 与 L2，并提供向量归一化。
package metric

import (
	"fmt"
	"math"

	"github.com/xavskye/gorag/internal/model"
)

// Distance 返回「越小越近」的距离。Cosine 使用 1-dot（要求已 L2 归一化）。
func Distance(a, b []float32, m model.Metric) float64 {
	switch m {
	case model.MetricL2:
		return L2(a, b)
	default:
		return 1 - Dot(a, b)
	}
}

// Similarity 返回「越大越近」的分数，供排序与展示。
func Similarity(a, b []float32, m model.Metric) float64 {
	switch m {
	case model.MetricL2:
		d := L2(a, b)
		return 1 / (1 + d)
	default:
		return Dot(a, b)
	}
}

func Dot(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var s float64
	i := 0
	for ; i+4 <= n; i += 4 {
		s += float64(a[i])*float64(b[i]) +
			float64(a[i+1])*float64(b[i+1]) +
			float64(a[i+2])*float64(b[i+2]) +
			float64(a[i+3])*float64(b[i+3])
	}
	for ; i < n; i++ {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

func L2(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var s float64
	i := 0
	for ; i+4 <= n; i += 4 {
		d0 := float64(a[i]) - float64(b[i])
		d1 := float64(a[i+1]) - float64(b[i+1])
		d2 := float64(a[i+2]) - float64(b[i+2])
		d3 := float64(a[i+3]) - float64(b[i+3])
		s += d0*d0 + d1*d1 + d2*d2 + d3*d3
	}
	for ; i < n; i++ {
		d := float64(a[i]) - float64(b[i])
		s += d * d
	}
	return math.Sqrt(s)
}

func L2Normalize(v []float32) []float32 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if s == 0 {
		return v
	}
	inv := float32(1 / math.Sqrt(s))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x * inv
	}
	return out
}

func ValidateDim(v []float32, dim int) error {
	if v == nil {
		return fmt.Errorf("vector is nil")
	}
	if len(v) != dim {
		return fmt.Errorf("vector dim %d != %d", len(v), dim)
	}
	for i, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return fmt.Errorf("vector[%d] is not finite", i)
		}
	}
	return nil
}

// PadOrTrim 将任意长度特征填充/截断到 dim，再 L2 归一化。
func PadOrTrim(src []float32, dim int) []float32 {
	out := make([]float32, dim)
	if len(src) == 0 {
		out[0] = 1
		return out
	}
	if len(src) >= dim {
		copy(out, src[:dim])
	} else {
		copy(out, src)
		for i := len(src); i < dim; i++ {
			out[i] = src[i%len(src)] * 0.15
		}
	}
	return L2Normalize(out)
}
