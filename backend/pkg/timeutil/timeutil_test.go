package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestNowInBeijing(t *testing.T) {
	n := Now()
	_, off := n.Zone()
	if off != 8*3600 {
		t.Fatalf("offset=%d want 28800", off)
	}
}

func TestFormat(t *testing.T) {
	ts := time.Date(2026, 8, 23, 15, 4, 5, 0, Beijing)
	got := Format(ts)
	if got != "2026-08-23 15:04:05" {
		t.Fatalf("got %q", got)
	}
	if Format(time.Time{}) != "" {
		t.Fatal("zero should be empty")
	}
}

func TestFromUnixMilliRoundTrip(t *testing.T) {
	ms := UnixMilli()
	back := FromUnixMilli(ms)
	if back.UnixMilli() != ms {
		t.Fatalf("roundtrip %d != %d", back.UnixMilli(), ms)
	}
	if !strings.Contains(Format(back), "-") {
		t.Fatalf("format missing date: %s", Format(back))
	}
}
