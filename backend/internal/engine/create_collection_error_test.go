package engine

import (
	"strings"
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func TestCreateCollectionReportsWALFailure(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.Close)

	if err := e.WAL.Close(); err != nil {
		t.Fatal(err)
	}
	err = e.CreateCollection(model.Collection{Name: "documents"})
	if err == nil {
		t.Fatal("CreateCollection succeeded while durable storage was unavailable")
	}
	if !strings.Contains(err.Error(), "wal closed") {
		t.Fatalf("CreateCollection returned the wrong persistence error: %v", err)
	}
}
