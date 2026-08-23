package store

import (
	"os"
	"testing"
)

func TestAssetsHashAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, err := NewAssets(dir)
	if err != nil {
		t.Fatal(err)
	}
	name, err := a.Put("0123456789abcdef0123456789abcdef", "image/png", []byte("pngdata"))
	if err != nil {
		t.Fatal(err)
	}
	if name == "" {
		t.Fatal("empty name")
	}
	p, err := a.Path("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "pngdata" {
		t.Fatalf("%s %v", b, err)
	}
	if _, err := a.Put("../etc/passwd", "image/png", []byte("x")); err == nil {
		t.Fatal("path traversal should fail")
	}
}

func TestManifestAlloc(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := m.AllocEntity()
	b := m.AllocEntity()
	if b != a+1 {
		t.Fatalf("%d %d", a, b)
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	m2, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m2.NextEntity != m.NextEntity {
		t.Fatalf("reload next=%d", m2.NextEntity)
	}
}
