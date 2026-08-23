package imagefeat

import (
	"image"
	"math"
)

// edgeOrient 8 方向梯度直方图 + 边缘密度。
func edgeOrient(img image.Image, r image.Rectangle) []float32 {
	hist := make([]float32, 16)
	var n, energy float32
	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		for x := r.Min.X + 1; x < r.Max.X-1; x++ {
			l := luma(img, x-1, y)
			ri := luma(img, x+1, y)
			u := luma(img, x, y-1)
			d := luma(img, x, y+1)
			gx := ri - l
			gy := d - u
			mag := float32(math.Hypot(float64(gx), float64(gy)))
			if mag < 0.04 {
				continue
			}
			ang := math.Atan2(float64(gy), float64(gx))
			if ang < 0 {
				ang += math.Pi
			}
			bin := int(ang / math.Pi * 8)
			if bin >= 8 {
				bin = 7
			}
			hist[bin] += mag
			energy += mag
			n++
		}
	}
	if energy > 0 {
		for i := 0; i < 8; i++ {
			hist[i] /= energy
		}
	}
	if n > 0 {
		hist[8] = energy / n
	}
	// 简单纹理：水平/垂直差分比
	hist[9] = clamp01(hist[0] + hist[4])
	hist[10] = clamp01(hist[2] + hist[6])
	return hist
}

func luma(img image.Image, x, y int) float32 {
	r, g, b, _ := img.At(x, y).RGBA()
	return float32((0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535)
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
