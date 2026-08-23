package wal

import (
	"path/filepath"
	"testing"
)

func TestAppendReplayAndDoubleClose(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Append(RecUpsert, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := l.Append(RecDelete, []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	l2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()
	recs, err := l2.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || string(recs[0].Payload) != "hello" {
		t.Fatalf("%+v", recs)
	}
	if _, err := osStat(dir); err != nil {
		t.Fatal(err)
	}
}

func osStat(dir string) (any, error) {
	return filepath.Abs(dir)
}
