package imagefeat

import (
	"image"
	"image/color"
	"math"
)

// hsvHist 8x8x4 = 256 bins.
func hsvHist(img image.Image, r image.Rectangle) []float32 {
	bins := make([]float32, 256)
	var n float32
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			h, s, v := rgbToHSV(img.At(x, y))
			hi := int(h * 8)
			si := int(s * 8)
			vi := int(v * 4)
			if hi > 7 {
				hi = 7
			}
			if si > 7 {
				si = 7
			}
			if vi > 3 {
				vi = 3
			}
			bins[hi*32+si*4+vi]++
			n++
		}
	}
	if n > 0 {
		for i := range bins {
			bins[i] /= n
		}
	}
	return bins
}

func rgbToHSV(c color.Color) (h, s, v float64) {
	r16, g16, b16, _ := c.RGBA()
	rf := float64(r16) / 65535
	gf := float64(g16) / 65535
	bf := float64(b16) / 65535
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	v = max
	d := max - min
	if max == 0 {
		s = 0
	} else {
		s = d / max
	}
	if d == 0 {
		h = 0
		return
	}
	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h /= 6
	if h < 0 {
		h += 1
	}
	return
}
