package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/model"
)

type Token struct {
	Text  string
	Done  bool
	Err   error
	Mock  bool
	Model string
}

type LLM interface {
	Name() string
	Kind() string
	Stream(ctx context.Context, question string, contexts []string) <-chan Token
}

type MockLLM struct{}

func (MockLLM) Name() string { return "mock-extractive" }
func (MockLLM) Kind() string { return "mock" }

func (MockLLM) Stream(ctx context.Context, question string, contexts []string) <-chan Token {
	ch := make(chan Token, 16)
	go func() {
		defer close(ch)
		var b strings.Builder
		b.WriteString("[MOCK] 基于检索片段的抽取式回答：")
		b.WriteString(question)
		b.WriteString("\n\n")
		if len(contexts) == 0 {
			b.WriteString("知识库中暂无足够证据，请先入库文档或图片。")
		} else {
			b.WriteString("综合下列证据：\n")
			for i, c := range contexts {
				if i >= 4 {
					break
				}
				runes := []rune(c)
				if len(runes) > 120 {
					c = string(runes[:120]) + "…"
				}
				b.WriteString(fmt.Sprintf("%d) %s\n", i+1, strings.TrimSpace(c)))
			}
		}
		text := b.String()
		for len(text) > 0 {
			select {
			case <-ctx.Done():
				ch <- Token{Err: ctx.Err(), Done: true, Mock: true, Model: "mock"}
				return
			default:
			}
			_, n := utf8.DecodeRuneInString(text)
			ch <- Token{Text: text[:n], Mock: true, Model: "mock"}
			text = text[n:]
			time.Sleep(8 * time.Millisecond)
		}
		ch <- Token{Done: true, Mock: true, Model: "mock"}
	}()
	return ch
}

type OpenAILLM struct {
	base   string
	key    string
	model  string
	client *http.Client
	ledger *cost.Ledger
}

func NewLLM(cfg *config.Config, ledger *cost.Ledger) LLM {
	if cfg.LLMProvider == "openai" {
		var llm *OpenAILLM
		if cfg.OpenAIAPIKey != "" {
			llm = &OpenAILLM{
				base:   strings.TrimRight(cfg.OpenAIBaseURL, "/"),
				key:    cfg.OpenAIAPIKey,
				model:  cfg.OpenAILLMModel,
				client: &http.Client{Timeout: 60 * time.Second},
				ledger: ledger,
			}
		}
		return llm
	}
	return MockLLM{}
}

func (o *OpenAILLM) Name() string { return o.model }
func (o *OpenAILLM) Kind() string { return "openai" }

func (o *OpenAILLM) Stream(ctx context.Context, question string, contexts []string) <-chan Token {
	ch := make(chan Token, 16)
	go func() {
		defer close(ch)
		if err := o.ledger.Allow(0.02); err != nil {
			for tok := range (MockLLM{}).Stream(ctx, question, contexts) {
				ch <- tok
			}
			return
		}
		prompt := "你是知识库助手。仅依据下列检索片段作答，并在句末用 [n] 标注引用。\n"
		for i, c := range contexts {
			prompt += fmt.Sprintf("[%d] %s\n", i+1, c)
		}
		body, _ := json.Marshal(map[string]any{
			"model":  o.model,
			"stream": true,
			"messages": []map[string]string{
				{"role": "system", "content": prompt},
				{"role": "user", "content": question},
			},
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			ch <- Token{Err: err, Done: true}
			return
		}
		req.Header.Set("Authorization", "Bearer "+o.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := o.client.Do(req)
		if err != nil {
			ch <- Token{Err: err, Done: true}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			raw, _ := io.ReadAll(resp.Body)
			o.ledger.Record(model.CostRecord{Provider: "openai-llm", Model: o.model, OK: false, Reason: fmt.Sprintf("http %d", resp.StatusCode)})
			if ClassifyStatus(resp.StatusCode) == ClassPermanent {
				ch <- Token{Err: fmt.Errorf("llm permanent %d %s", resp.StatusCode, string(raw)), Done: true}
				return
			}
			for tok := range (MockLLM{}).Stream(ctx, question, contexts) {
				ch <- tok
			}
			return
		}
		sc := bufio.NewScanner(resp.Body)
		tokens := 0
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				break
			}
			var ev struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if json.Unmarshal([]byte(payload), &ev) == nil && len(ev.Choices) > 0 {
				if t := ev.Choices[0].Delta.Content; t != "" {
					tokens += utf8.RuneCountInString(t)
					ch <- Token{Text: t, Model: o.model}
				}
			}
		}
		cny := float64(tokens) / 1_000_000 * 0.6 * 7.2
		o.ledger.Record(model.CostRecord{Provider: "openai-llm", Model: o.model, Tokens: tokens, CNY: cny, OK: true})
		ch <- Token{Done: true, Model: o.model}
	}()
	return ch
}
