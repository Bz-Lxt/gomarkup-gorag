package imagefeat

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
)

func solidPNG(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestExtractRedVsBlue(t *testing.T) {
	red, err := Extract(bytes.NewReader(solidPNG(64, 64, color.RGBA{220, 20, 20, 255})), Options{Dim: model.DefaultDim, Grid: 3})
	if err != nil {
		t.Fatal(err)
	}
	blue, err := Extract(bytes.NewReader(solidPNG(64, 64, color.RGBA{20, 40, 220, 255})), Options{Dim: model.DefaultDim, Grid: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(red.Patches) != 9 {
		t.Fatalf("patches=%d", len(red.Patches))
	}
	if math.Abs(metric.Dot(red.Vector, red.Vector)-1) > 1e-3 {
		t.Fatal("not normalized")
	}
	same, err := Extract(bytes.NewReader(solidPNG(64, 64, color.RGBA{220, 20, 20, 255})), Options{Dim: model.DefaultDim, Grid: 3})
	if err != nil {
		t.Fatal(err)
	}
	if metric.Dot(red.Vector, same.Vector) < 0.98 {
		t.Fatalf("identical images not similar: %v", metric.Dot(red.Vector, same.Vector))
	}
	if metric.Dot(red.Vector, blue.Vector) > metric.Dot(red.Vector, same.Vector) {
		t.Fatal("red closer to blue than to itself")
	}
}

func TestRejectTiny(t *testing.T) {
	_, err := Extract(bytes.NewReader(solidPNG(4, 4, color.White)), Options{})
	if err == nil {
		t.Fatal("expected too small")
	}
}
