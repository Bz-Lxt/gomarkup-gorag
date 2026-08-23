package segment

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/model"
)

func TestCodecRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "seg_1.bin")
	ents := []model.Entity{{
		ID: 1, Collection: "demo", Modality: model.ModalityText,
		Vector: make([]float32, 8), Content: "hello",
	}}
	crc, err := WriteFile(p, FileHeader{ID: 1, RowCount: 1, IndexType: model.IndexFLAT}, ents)
	if err != nil {
		t.Fatal(err)
	}
	h, got, crc2, err := ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if crc != crc2 || h.ID != 1 || len(got) != 1 || got[0].Content != "hello" {
		t.Fatalf("hdr=%+v ents=%+v", h, got)
	}
}

func TestBufferTriggersRows(t *testing.T) {
	b := NewBuffer(1, 1<<30, 2, time.Hour)
	full, _ := b.Append(model.Entity{ID: 1})
	if full {
		t.Fatal("early")
	}
	full, reason := b.Append(model.Entity{ID: 2})
	if !full || reason != "max_rows" {
		t.Fatalf("full=%v reason=%s", full, reason)
	}
}

func TestBadMagicRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.bin")
	if err := os.WriteFile(p, []byte("XXXX"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ReadFile(p); err == nil {
		t.Fatal("expected error")
	}
}
