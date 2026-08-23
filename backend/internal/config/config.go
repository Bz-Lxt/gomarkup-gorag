package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr            string
	DataDir             string
	LogLevel            string
	LogEnv              string
	SegmentMaxBytes     int64
	SegmentMaxRows      int
	SegmentMaxIdleSec   int
	HNSWM               int
	HNSWEfConstruction  int
	HNSWEfSearch        int
	PatchGrid           int
	BudgetLimitCNY      float64
	EmbeddingProvider   string
	VisionProvider      string
	LLMProvider         string
	OpenAIBaseURL       string
	OpenAIAPIKey        string
	OpenAIEmbedModel    string
	OpenAILLMModel      string
	CLIPBaseURL         string
	CLIPAPIKey          string
	DemoUser            string
	DemoPass            string
	MaxUploadBytes      int64
	MaxImagePixels      int
}

func Load() (*Config, error) {
	c := &Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DataDir:            env("DATA_DIR", "./data"),
		LogLevel:           env("LOG_LEVEL", "info"),
		LogEnv:             env("LOG_ENV", "development"),
		SegmentMaxBytes:    envI64("SEGMENT_MAX_BYTES", 4*1024*1024),
		SegmentMaxRows:     envInt("SEGMENT_MAX_ROWS", 1000),
		SegmentMaxIdleSec:  envInt("SEGMENT_MAX_IDLE_SEC", 30),
		HNSWM:              envInt("HNSW_M", 16),
		HNSWEfConstruction: envInt("HNSW_EF_CONSTRUCTION", 200),
		HNSWEfSearch:       envInt("HNSW_EF_SEARCH", 64),
		PatchGrid:          envInt("PATCH_GRID", 3),
		BudgetLimitCNY:     envF64("BUDGET_LIMIT_CNY", 10.00),
		EmbeddingProvider:  strings.ToLower(env("EMBEDDING_PROVIDER", "local")),
		VisionProvider:     strings.ToLower(env("VISION_PROVIDER", "local")),
		LLMProvider:        strings.ToLower(env("LLM_PROVIDER", "mock")),
		OpenAIBaseURL:      env("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbedModel:   env("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		OpenAILLMModel:     env("OPENAI_LLM_MODEL", "gpt-4o-mini"),
		CLIPBaseURL:        env("CLIP_BASE_URL", ""),
		CLIPAPIKey:         os.Getenv("CLIP_API_KEY"),
		DemoUser:           env("DEMO_USER", "admin"),
		DemoPass:           env("DEMO_PASS", "gorag123"),
		MaxUploadBytes:     envI64("MAX_UPLOAD_BYTES", 10*1024*1024),
		MaxImagePixels:     envInt("MAX_IMAGE_PIXELS", 25_000_000),
	}
	if c.HNSWM < 4 {
		return nil, fmt.Errorf("HNSW_M must be >= 4")
	}
	if c.PatchGrid < 2 || c.PatchGrid > 8 {
		return nil, fmt.Errorf("PATCH_GRID must be in [2,8]")
	}
	if c.SegmentMaxBytes < 1024 {
		return nil, fmt.Errorf("SEGMENT_MAX_BYTES too small")
	}
	return c, nil
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func envI64(k string, def int64) int64 {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return def
}

func envF64(k string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return n
		}
	}
	return def
}
