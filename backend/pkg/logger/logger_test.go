package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestProductionMasksDebug(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf, slog.LevelInfo)
	Debug("hidden")
	Info("visible")
	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatal("debug leaked at info level")
	}
	if !strings.Contains(out, "visible") {
		t.Fatal("info missing")
	}
}

func TestInitProductionClampsDebug(t *testing.T) {
	Init("debug", "production")
	mu.RLock()
	lv := level
	mu.RUnlock()
	if lv < slog.LevelInfo {
		t.Fatalf("production still at %v", lv)
	}
}
