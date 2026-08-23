package engine

import (
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

type reusingEmbedder struct {
	buf []float32
}

func (e *reusingEmbedder) Name() string { return "reusing-test" }
func (e *reusingEmbedder) Kind() string { return "test" }

func (e *reusingEmbedder) Embed(s string) ([]float32, error) {
	if e.buf == nil {
		e.buf = make([]float32, model.DefaultDim)
	}
	clear(e.buf)
	if s == "second" {
		e.buf[1] = 1
	} else {
		e.buf[0] = 1
	}
	return e.buf, nil
}

func TestCompareFLATKeepsInsertedVectorsIndependent(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Embed = &reusingEmbedder{}

	first, err := e.IngestDocument(IngestDocReq{Content: "first"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.IngestDocument(IngestDocReq{Content: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if first.EntityIDs[0] == second.EntityIDs[0] {
		t.Fatal("ingests returned the same entity ID")
	}

	resp, err := e.SearchText(model.SearchRequest{
		Query:       "second",
		TopK:        1,
		CompareFLAT: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.FLATHits) != 1 {
		t.Fatalf("expected one FLAT hit, got %d", len(resp.FLATHits))
	}
	if got := resp.FLATHits[0].ID; got != second.EntityIDs[0] {
		t.Fatalf("FLAT nearest neighbor changed after the embedder buffer was reused: got %d, want %d", got, second.EntityIDs[0])
	}
}
