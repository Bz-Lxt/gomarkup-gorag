package provider

import (
	"errors"
	"net"
	"net/http"
	"time"
)

// Classify 将错误分为 transient（可重试）与 permanent（鉴权/校验，禁止重试）。
type Class int

const (
	ClassOK Class = iota
	ClassTransient
	ClassPermanent
)

func ClassifyStatus(code int) Class {
	switch {
	case code == http.StatusTooManyRequests:
		return ClassTransient
	case code >= 500:
		return ClassTransient
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return ClassPermanent
	case code >= 400 && code < 500:
		return ClassPermanent
	default:
		return ClassOK
	}
}

func ClassifyErr(err error) Class {
	if err == nil {
		return ClassOK
	}
	var ne net.Error
	if errors.As(err, &ne) && (ne.Timeout() || ne.Temporary()) {
		return ClassTransient
	}
	return ClassPermanent
}

func Backoff(attempt int) time.Duration {
	d := time.Duration(200*(1<<attempt)) * time.Millisecond
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	return d
}

// DoNarrow 仅对瞬时错误重试，最多 3 次。
func DoNarrow(fn func() (Class, error)) error {
	var last error
	for i := 0; i < 3; i++ {
		cls, err := fn()
		if err == nil {
			return nil
		}
		last = err
		if cls != ClassTransient || i == 2 {
			return last
		}
		time.Sleep(Backoff(i))
	}
	return last
}
