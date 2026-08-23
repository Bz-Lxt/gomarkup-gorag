package provider_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/provider"
)

var errStreamClosed = errors.New("stream closed without a terminal token")

func TestOpenAILLMOverlappingStreamsKeepContextsIndependent(t *testing.T) {
	llm := provider.NewLLM(&config.Config{
		LLMProvider:   "openai",
		OpenAIAPIKey:  "test-key",
		OpenAILLMModel: "test-model",
	}, cost.New(0))
	if got := llm.Kind(); got != "openai" {
		t.Fatalf("provider kind = %q, want openai", got)
	}

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	firstStarted, firstFinished := observeStream(llm.Stream(
		firstCtx,
		strings.Repeat("first answer keeps generating ", 100),
		nil,
	))
	waitForStart(t, firstStarted, "first stream")

	secondCtx, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	secondStarted, secondFinished := observeStream(llm.Stream(
		secondCtx,
		strings.Repeat("second answer keeps generating ", 100),
		nil,
	))
	waitForStart(t, secondStarted, "second stream")

	var premature *provider.Token
	select {
	case tok := <-firstFinished:
		premature = &tok
	case <-time.After(300 * time.Millisecond):
	}

	cancelFirst()
	cancelSecond()
	if premature == nil {
		waitForFinish(t, firstFinished, "first stream")
	}
	waitForFinish(t, secondFinished, "second stream")

	if premature != nil {
		if errors.Is(premature.Err, context.Canceled) {
			t.Fatal("starting a second stream canceled the first stream's context")
		}
		t.Fatalf("first stream ended while its context was active: done=%v err=%v", premature.Done, premature.Err)
	}
}

func observeStream(ch <-chan provider.Token) (<-chan struct{}, <-chan provider.Token) {
	started := make(chan struct{})
	finished := make(chan provider.Token, 1)
	go func() {
		signaled := false
		for tok := range ch {
			if tok.Text != "" && !signaled {
				close(started)
				signaled = true
			}
			if tok.Err != nil || tok.Done {
				finished <- tok
				return
			}
		}
		finished <- provider.Token{Err: errStreamClosed}
	}()
	return started, finished
}

func waitForStart(t *testing.T, started <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("%s did not produce a token", name)
	}
}

func waitForFinish(t *testing.T, finished <-chan provider.Token, name string) {
	t.Helper()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatalf("%s did not stop after cancellation", name)
	}
}
