package imagefeat

import (
	"image"
	"math"
)

const dctN = 32
const hashN = 8

func perceptualHash(img image.Image, r image.Rectangle) []float32 {
	gray := downsampleGray(img, r, dctN)
	coeffs := dct2(gray, dctN)
	// 取低频 8x8（跳过 DC）作为连续特征，再追加 64-bit 符号位
	feat := make([]float32, 64+64)
	var acc float64
	k := 0
	for y := 0; y < hashN; y++ {
		for x := 0; x < hashN; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if k >= 63 {
				break
			}
			feat[k] = float32(coeffs[y*dctN+x])
			acc += coeffs[y*dctN+x]
			k++
		}
	}
	mean := acc / 63
	for i := 0; i < 64; i++ {
		y := i / 8
		x := i % 8
		if coeffs[y*dctN+x] > mean {
			feat[64+i] = 1
		} else {
			feat[64+i] = -1
		}
	}
	return feat
}

func downsampleGray(img image.Image, r image.Rectangle, n int) []float64 {
	out := make([]float64, n*n)
	w := r.Dx()
	h := r.Dy()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			sx := r.Min.X + x*w/n
			sy := r.Min.Y + y*h/n
			if sx >= r.Max.X {
				sx = r.Max.X - 1
			}
			if sy >= r.Max.Y {
				sy = r.Max.Y - 1
			}
			cr, cg, cb, _ := img.At(sx, sy).RGBA()
			out[y*n+x] = (0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)) / 65535
		}
	}
	return out
}

func dct2(pix []float64, n int) []float64 {
	tmp := make([]float64, n*n)
	out := make([]float64, n*n)
	for y := 0; y < n; y++ {
		dct1(pix[y*n:y*n+n], tmp[y*n:y*n+n])
	}
	col := make([]float64, n)
	dst := make([]float64, n)
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			col[y] = tmp[y*n+x]
		}
		dct1(col, dst)
		for y := 0; y < n; y++ {
			out[y*n+x] = dst[y]
		}
	}
	return out
}

func dct1(in, out []float64) {
	n := len(in)
	fn := float64(n)
	for k := 0; k < n; k++ {
		var s float64
		for i := 0; i < n; i++ {
			s += in[i] * math.Cos(math.Pi*(float64(i)+0.5)*float64(k)/fn)
		}
		if k == 0 {
			s *= math.Sqrt(1 / fn)
		} else {
			s *= math.Sqrt(2 / fn)
		}
		out[k] = s
	}
}
