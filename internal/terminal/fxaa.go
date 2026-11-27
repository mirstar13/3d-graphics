package terminal

import "github.com/mirstar13/go-3d-graphics/internal/engine"

type fxaaDevice interface {
	// apply modifies colors only after a complete, successful readback.
	apply([][]engine.Color) error
	name() string
	close()
}

// Use fixed-point luminance so the CPU filter and GLSL agree on threshold
// boundaries. The coefficients are in thousandths and imply 31.2 for the
// minimum contrast threshold in these units.
func fixedLuminance(c engine.Color) int {
	return 299*int(c.R) + 587*int(c.G) + 114*int(c.B)
}

func filterFXAA(colors [][]engine.Color, output []engine.Color) []engine.Color {
	w, h := len(colors[0]), len(colors)
	if cap(output) < w*h {
		output = make([]engine.Color, w*h)
	}
	output = output[:w*h]
	for y := 1; y < h-1; y++ {
		top, row, bottom := colors[y-1], colors[y], colors[y+1]
		for x := 1; x < w-1; x++ {
			center := row[x]
			output[y*w+x] = center
			c, n, s, e, west := fixedLuminance(center), fixedLuminance(top[x]), fixedLuminance(bottom[x]), fixedLuminance(row[x+1]), fixedLuminance(row[x-1])
			high, low := max(c, n, s, e, west), min(c, n, s, e, west)
			contrast := high - low
			if contrast <= 31 || 8*contrast < high {
				continue
			}
			nw, ne, sw, se := fixedLuminance(top[x-1]), fixedLuminance(top[x+1]), fixedLuminance(bottom[x-1]), fixedLuminance(bottom[x+1])
			corners := nw + ne + sw + se
			horizontal := engine.Abs(corners-2*(west+c+e)) >= engine.Abs(corners-2*(n+c+s))
			if horizontal && engine.Abs((nw+ne-2*n)-(sw+se-2*s)) > contrast {
				output[y*w+x] = averageThree(center, row[x+1], row[x-1])
			}
			if !horizontal && engine.Abs((nw+sw-2*west)-(ne+se-2*e)) > contrast {
				output[y*w+x] = averageThree(center, top[x], bottom[x])
			}
		}
	}
	for y := 1; y < h-1; y++ {
		copy(colors[y][1:w-1], output[y*w+1:y*w+w-1])
	}
	return output
}

func averageThree(a, b, c engine.Color) engine.Color {
	return engine.Color{
		R: uint8((int(a.R) + int(b.R) + int(c.R)) / 3),
		G: uint8((int(a.G) + int(b.G) + int(c.G)) / 3),
		B: uint8((int(a.B) + int(b.B) + int(c.B)) / 3),
	}
}
