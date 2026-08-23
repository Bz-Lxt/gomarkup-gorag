// Package text 提供确定性 hashing-trick embedding（1024 维）。
package text

import (
	"hash/fnv"
	"unicode/utf8"

	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/tokenize"
)

// Embed 将文本映射到固定维度单位向量。同输入必同输出。
func Embed(text string, dim int) []float32 {
	if dim <= 0 {
		dim = model.DefaultDim
	}
	acc := make([]float32, dim)
	if text == "" {
		acc[0] = 1
		return acc
	}
	add := func(s string, w float32) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(s))
		v := h.Sum64()
		i := int(v % uint64(dim))
		sign := float32(1)
		if v&1 == 1 {
			sign = -1
		}
		acc[i] += sign * w
		j := int((v >> 17) % uint64(dim))
		if j != i {
			acc[j] += sign * w * 0.35
		}
	}
	for _, t := range tokenize.Tokenize(text) {
		add("t:"+t.Term, 1)
	}
	// 字符 3-gram 增强短查询稳定性
	rs := []rune(text)
	for i := 0; i+2 < len(rs); i++ {
		add("g:"+string(rs[i:i+3]), 0.4)
	}
	if utf8.RuneCountInString(text) < 3 {
		add("raw:"+text, 1.2)
	}
	return metric.L2Normalize(acc)
}

func EmbedSentences(text string, dim int) []model.Sentence {
	spans := tokenize.SplitSentences(text)
	out := make([]model.Sentence, 0, len(spans))
	for _, sp := range spans {
		if sp[1] <= sp[0] {
			continue
		}
		chunk := text[sp[0]:sp[1]]
		out = append(out, model.Sentence{
			Start:  sp[0],
			End:    sp[1],
			Vector: Embed(chunk, dim),
		})
	}
	return out
}
