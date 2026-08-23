package provider

import (
	"errors"
	"net/http"
	"testing"
)

func TestClassifyNeverRetriesAuth(t *testing.T) {
	if ClassifyStatus(http.StatusUnauthorized) != ClassPermanent {
		t.Fatal("401 must be permanent")
	}
	if ClassifyStatus(http.StatusBadRequest) != ClassPermanent {
		t.Fatal("400 must be permanent")
	}
	if ClassifyStatus(http.StatusTooManyRequests) != ClassTransient {
		t.Fatal("429 should retry")
	}
	if ClassifyStatus(http.StatusBadGateway) != ClassTransient {
		t.Fatal("502 should retry")
	}
}

func TestDoNarrowStopsOnPermanent(t *testing.T) {
	n := 0
	err := DoNarrow(func() (Class, error) {
		n++
		return ClassPermanent, errors.New("bad key")
	})
	if err == nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
