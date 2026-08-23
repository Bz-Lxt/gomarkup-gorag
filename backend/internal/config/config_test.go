package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SEGMENT_MAX_BYTES", "4194304")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HNSWM != 16 || cfg.PatchGrid != 3 {
		t.Fatalf("%+v", cfg)
	}
}

func TestRejectBadGrid(t *testing.T) {
	t.Setenv("PATCH_GRID", "99")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
	_ = os.Unsetenv("PATCH_GRID")
}
