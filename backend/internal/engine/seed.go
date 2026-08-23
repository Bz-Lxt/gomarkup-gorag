package engine

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/xavskye/gorag/pkg/logger"
)

type seedImage struct {
	caption string
	tags    []string
	paint   func(*image.RGBA)
}

func (e *Engine) seedImages() {
	items := []seedImage{
		{caption: "暖色日落色块", tags: []string{"sunset", "warm"}, paint: func(img *image.RGBA) {
			fillGradient(img, color.RGBA{255, 90, 30, 255}, color.RGBA{255, 200, 80, 255})
		}},
		{caption: "冷色海洋色块", tags: []string{"ocean", "cool"}, paint: func(img *image.RGBA) {
			fillGradient(img, color.RGBA{10, 40, 120, 255}, color.RGBA{40, 180, 220, 255})
		}},
		{caption: "森林绿色纹理", tags: []string{"forest", "green"}, paint: func(img *image.RGBA) {
			fillGradient(img, color.RGBA{10, 60, 20, 255}, color.RGBA{80, 180, 60, 255})
			stripe(img, color.RGBA{20, 90, 30, 255})
		}},
		{caption: "品红几何块面", tags: []string{"geo", "magenta"}, paint: func(img *image.RGBA) {
			fillSolid(img, color.RGBA{200, 30, 120, 255})
			rect(img, 20, 20, 80, 80, color.RGBA{255, 220, 40, 255})
		}},
		{caption: "石墨灰网格", tags: []string{"grid", "gray"}, paint: func(img *image.RGBA) {
			fillSolid(img, color.RGBA{40, 42, 48, 255})
			grid(img, color.RGBA{180, 180, 190, 255})
		}},
	}
	for _, it := range items {
		img := image.NewRGBA(image.Rect(0, 0, 160, 160))
		it.paint(img)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			logger.Warn("seed.image_encode", "err", err)
			continue
		}
		if _, err := e.IngestImage(IngestImageReq{
			Collection: "default", Caption: it.caption, Tags: it.tags, Reader: bytes.NewReader(buf.Bytes()),
		}); err != nil {
			logger.Warn("seed.image_ingest", "err", err)
		}
	}
}

func fillSolid(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func fillGradient(img *image.RGBA, a, b color.RGBA) {
	r := img.Bounds()
	h := float64(r.Dy())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		t := float64(y-r.Min.Y) / h
		c := color.RGBA{
			R: lerp(a.R, b.R, t), G: lerp(a.G, b.G, t), B: lerp(a.B, b.B, t), A: 255,
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func stripe(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 8 {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func rect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func grid(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 16 {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	for x := b.Min.X; x < b.Max.X; x += 16 {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t)
}
