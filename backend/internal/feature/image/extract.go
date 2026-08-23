package imagefeat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
	"golang.org/x/image/webp"
)

type Result struct {
	Vector      []float32
	Patches     []model.Patch
	Width       int
	Height      int
	MIME        string
	ContentHash string
	Bytes       []byte
}

type Options struct {
	Dim        int
	Grid       int
	MaxBytes   int64
	MaxPixels  int
}

func Extract(r io.Reader, opt Options) (*Result, error) {
	if opt.Dim <= 0 {
		opt.Dim = model.DefaultDim
	}
	if opt.Grid <= 0 {
		opt.Grid = 3
	}
	if opt.MaxBytes <= 0 {
		opt.MaxBytes = 10 * 1024 * 1024
	}
	if opt.MaxPixels <= 0 {
		opt.MaxPixels = 25_000_000
	}
	limited := io.LimitReader(r, opt.MaxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, model.Wrap(model.CodeUploadInvalid, "read upload", err)
	}
	if int64(len(raw)) > opt.MaxBytes {
		return nil, model.NewError(model.CodeUploadInvalid, "file exceeds size limit")
	}
	if len(raw) < 8 {
		return nil, model.NewError(model.CodeUploadInvalid, "file too small")
	}
	mime := http.DetectContentType(raw)
	img, err := decode(raw, mime)
	if err != nil {
		return nil, model.Wrap(model.CodeUploadInvalid, "decode image", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 8 || h < 8 {
		return nil, model.NewError(model.CodeUploadInvalid, "image too small")
	}
	if w*h > opt.MaxPixels {
		return nil, model.NewError(model.CodeUploadInvalid, "image pixel count exceeds limit")
	}
	sum := sha256.Sum256(raw)
	res := &Result{
		Width:       w,
		Height:      h,
		MIME:        mime,
		ContentHash: hex.EncodeToString(sum[:]),
		Bytes:       raw,
		Vector:      descriptor(img, b, opt.Dim),
	}
	cellW := float64(w) / float64(opt.Grid)
	cellH := float64(h) / float64(opt.Grid)
	for gy := 0; gy < opt.Grid; gy++ {
		for gx := 0; gx < opt.Grid; gx++ {
			rect := image.Rect(
				b.Min.X+int(float64(gx)*cellW),
				b.Min.Y+int(float64(gy)*cellH),
				b.Min.X+int(float64(gx+1)*cellW),
				b.Min.Y+int(float64(gy+1)*cellH),
			)
			if rect.Dx() < 4 || rect.Dy() < 4 {
				continue
			}
			res.Patches = append(res.Patches, model.Patch{
				GridRow: gy,
				GridCol: gx,
				BBox: [4]float64{
					float64(gx) / float64(opt.Grid),
					float64(gy) / float64(opt.Grid),
					1 / float64(opt.Grid),
					1 / float64(opt.Grid),
				},
				Vector: descriptor(img, rect, opt.Dim),
			})
		}
	}
	return res, nil
}

func descriptor(img image.Image, r image.Rectangle, dim int) []float32 {
	feat := make([]float32, 0, 256+128+16)
	feat = append(feat, hsvHist(img, r)...)
	feat = append(feat, perceptualHash(img, r)...)
	feat = append(feat, edgeOrient(img, r)...)
	return metric.PadOrTrim(feat, dim)
}

func decode(raw []byte, mime string) (image.Image, error) {
	br := bytes.NewReader(raw)
	switch mime {
	case "image/jpeg":
		return jpeg.Decode(br)
	case "image/png":
		return png.Decode(br)
	case "image/webp":
		return webp.Decode(br)
	default:
		img, _, err := image.Decode(br)
		if err != nil {
			return nil, fmt.Errorf("unsupported mime %s: %w", mime, err)
		}
		return img, nil
	}
}
