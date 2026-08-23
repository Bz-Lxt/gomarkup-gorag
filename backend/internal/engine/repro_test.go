package engine

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xavskye/gorag/internal/model"
)

func solidPNG(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// Regression: when CLIP is not configured, text-to-image hybrid search must
// degrade to the caption/tag scalar channel instead of panicking.
func TestTextToImageNoCLIPDegradesGracefully(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if e.Cfg.VisionProvider == "clip_api" && e.Cfg.CLIPBaseURL != "" && e.Cfg.CLIPAPIKey != "" {
		t.Skip("CLIP configured, skipping no-CLIP regression")
	}
	if e.CLIP.Enabled() {
		t.Fatalf("CLIP should be disabled when unconfigured, type=%T", e.CLIP)
	}
	_, err = e.IngestImage(IngestImageReq{
		Collection: "default", Caption: "一只猫", Tags: []string{"cat", "pet"},
		Reader: bytes.NewReader(solidPNG(64, 64, color.RGBA{100, 50, 200, 255})),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := e.SearchHybrid(model.SearchRequest{
		Collection: "default", Query: "猫 cat", Modality: model.ModalityImage, TopK: 5,
	}, nil)
	if err != nil {
		t.Fatalf("should degrade gracefully, got err: %v", err)
	}
	if resp == nil {
		t.Fatal("nil resp")
	}
	if len(resp.Hits) == 0 {
		t.Fatalf("expected keyword-channel hits, got resp=%+v", resp)
	}
	if resp.DegradeNote == "" {
		t.Fatal("expected degrade_note explaining fallback")
	}
	if resp.CrossModal {
		t.Fatal("cross_modal should be false when CLIP not configured")
	}
	// vector channel must be closed for images without CLIP
	for _, ch := range resp.Channels {
		if ch == "vector" {
			t.Fatalf("vector channel should be closed without CLIP, got channels=%v", resp.Channels)
		}
	}
}

// Regression: text-to-text and image-to-image (pure visual) search must still
// work when CLIP is not configured.
func TestNonCrossModalSearchWorksWhenCLIPDisabled(t *testing.T) {
	e, err := Open(testCfg(t))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if e.CLIP.Enabled() {
		t.Skip("CLIP configured, skipping")
	}
	_, err = e.IngestDocument(IngestDocReq{
		Collection: "default", Title: "t", Content: "向量检索与混合检索的融合",
	})
	if err != nil {
		t.Fatal(err)
	}
	// text-to-text: vector+keyword channels should work
	resp, err := e.SearchText(model.SearchRequest{Query: "向量检索", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Hits) == 0 {
		t.Fatal("text-to-text returned no hits")
	}
}
