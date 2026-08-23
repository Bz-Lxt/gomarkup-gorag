// Package tokenize 实现纯 Go 混合分词：ASCII 词边界 + CJK unigram/bigram。
package tokenize

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xavskye/gorag/internal/model"
)

type Token struct {
	Term  string
	Start int
	End   int
}

func Tokenize(text string) []Token {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	var out []Token
	var ascii []rune
	asciiStart := 0
	flushASCII := func(end int) {
		if len(ascii) == 0 {
			return
		}
		term := strings.ToLower(string(ascii))
		if !isStop(term) && len(term) > 0 {
			out = append(out, Token{Term: term, Start: asciiStart, End: end})
		}
		ascii = ascii[:0]
	}
	var cjk []rune
	var cjkStarts []int
	flushCJK := func() {
		if len(cjk) == 0 {
			return
		}
		for i, r := range cjk {
			term := string(r)
			if !isStop(term) {
				out = append(out, Token{Term: term, Start: cjkStarts[i], End: cjkStarts[i] + utf8.RuneLen(r)})
			}
			if i+1 < len(cjk) {
				bi := string(cjk[i]) + string(cjk[i+1])
				if !isStop(bi) {
					end := cjkStarts[i+1] + utf8.RuneLen(cjk[i+1])
					out = append(out, Token{Term: bi, Start: cjkStarts[i], End: end})
				}
			}
		}
		cjk = cjk[:0]
		cjkStarts = cjkStarts[:0]
	}

	bytePos := 0
	for i, r := range runes {
		size := utf8.RuneLen(r)
		switch {
		case isASCIIWord(r):
			flushCJK()
			if len(ascii) == 0 {
				asciiStart = bytePos
			}
			ascii = append(ascii, unicode.ToLower(r))
		case isCJK(r):
			flushASCII(bytePos)
			cjk = append(cjk, r)
			cjkStarts = append(cjkStarts, bytePos)
		default:
			flushASCII(bytePos)
			flushCJK()
		}
		bytePos += size
		_ = i
	}
	flushASCII(bytePos)
	flushCJK()
	return out
}

func Terms(text string) []string {
	toks := Tokenize(text)
	out := make([]string, 0, len(toks))
	seen := map[string]struct{}{}
	for _, t := range toks {
		if _, ok := seen[t.Term]; ok {
			continue
		}
		seen[t.Term] = struct{}{}
		out = append(out, t.Term)
	}
	return out
}

func ToTermPos(toks []Token) []model.TermPos {
	out := make([]model.TermPos, len(toks))
	for i, t := range toks {
		out[i] = model.TermPos{Term: t.Term, Start: t.Start, End: t.End}
	}
	return out
}

func isASCIIWord(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x30FF) || // kana
		(r >= 0x3400 && r <= 0x4DBF)
}

// SplitSentences 按中英文标点切句，返回 [start,end) 字节区间。
func SplitSentences(text string) [][2]int {
	if text == "" {
		return nil
	}
	var out [][2]int
	start := 0
	for i, r := range text {
		if r == '。' || r == '！' || r == '？' || r == '\n' || r == '.' || r == '!' || r == '?' {
			end := i + utf8.RuneLen(r)
			if end > start {
				out = append(out, [2]int{start, end})
			}
			start = end
		}
	}
	if start < len(text) {
		out = append(out, [2]int{start, len(text)})
	}
	return out
}
