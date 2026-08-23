package provider

import (
	"testing"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/cost"
	"github.com/xavskye/gorag/internal/model"
)

func TestNewCLIPDisabledWhenUnconfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
	}{
		{"empty", &config.Config{}},
		{"wrong provider", &config.Config{VisionProvider: "local", CLIPBaseURL: "http://x", CLIPAPIKey: "k"}},
		{"missing url", &config.Config{VisionProvider: "clip_api", CLIPAPIKey: "k"}},
		{"missing key", &config.Config{VisionProvider: "clip_api", CLIPBaseURL: "http://x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCLIP(tc.cfg, cost.New(10))
			if c.Enabled() {
				t.Fatalf("Enabled() should be false, type=%T", c)
			}
			if c.Name() != "off" {
				t.Fatalf("Name() should be \"off\", got %q", c.Name())
			}
			_, err := c.EmbedText("cat")
			if err == nil {
				t.Fatal("EmbedText should error when unconfigured")
			}
			if !model.IsCode(err, model.CodeUnimplemented) {
				t.Fatalf("expected CodeUnimplemented, got %v", err)
			}
		})
	}
}

func TestNewCLIPEnabledWhenConfigured(t *testing.T) {
	cfg := &config.Config{
		VisionProvider: "clip_api", CLIPBaseURL: "http://localhost:8000", CLIPAPIKey: "secret",
	}
	c := NewCLIP(cfg, cost.New(10))
	if !c.Enabled() {
		t.Fatalf("Enabled() should be true when configured, type=%T", c)
	}
	if c.Name() != "clip_api" {
		t.Fatalf("Name() should be \"clip_api\", got %q", c.Name())
	}
}
