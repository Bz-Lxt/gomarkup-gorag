package tokenize

import "testing"

func TestMixedTokenize(t *testing.T) {
	toks := Tokenize("Go语言向量检索 Vector Search")
	has := map[string]bool{}
	for _, tk := range toks {
		has[tk.Term] = true
	}
	for _, want := range []string{"go", "语言", "向量", "检索", "vector", "search"} {
		if !has[want] {
			t.Fatalf("missing term %q in %#v", want, toks)
		}
	}
}

func TestCharOffsets(t *testing.T) {
	s := "以图搜图"
	toks := Tokenize(s)
	if len(toks) == 0 {
		t.Fatal("empty")
	}
	for _, tk := range toks {
		if tk.Start < 0 || tk.End > len(s) || tk.Start >= tk.End {
			t.Fatalf("bad range %+v", tk)
		}
		if s[tk.Start:tk.End] == "" {
			t.Fatal("empty slice")
		}
	}
}

func TestStopwordsDropped(t *testing.T) {
	toks := Tokenize("the vector 的 检索")
	for _, tk := range toks {
		if tk.Term == "the" || tk.Term == "的" {
			t.Fatalf("stopword leaked: %s", tk.Term)
		}
	}
}

func TestSplitSentences(t *testing.T) {
	ss := SplitSentences("你好。世界！Go.")
	if len(ss) < 2 {
		t.Fatalf("got %d", len(ss))
	}
}
