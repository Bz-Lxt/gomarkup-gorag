package filter

import (
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func TestParseAndMatch(t *testing.T) {
	n, err := Parse(`tag == "cat" && score > 0.5`)
	if err != nil {
		t.Fatal(err)
	}
	e := &model.Entity{Tags: []string{"cat"}, Modality: model.ModalityImage}
	if !Match(n, e, 0.8) {
		t.Fatal("should match")
	}
	if Match(n, e, 0.1) {
		t.Fatal("score too low")
	}
	e.Tags = []string{"dog"}
	if Match(n, e, 0.9) {
		t.Fatal("tag mismatch")
	}
}

func TestRejectUnknownField(t *testing.T) {
	_, err := Parse(`password == "x"`)
	if err == nil {
		t.Fatal("expected whitelist reject")
	}
}

func TestEmptyIsNil(t *testing.T) {
	n, err := Parse("  ")
	if err != nil || n != nil {
		t.Fatalf("n=%v err=%v", n, err)
	}
}
