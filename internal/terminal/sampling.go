package terminal

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

type sampleOffset struct{ x, y float64 }

var centerSamples = []sampleOffset{{.5, .5}}
var twoSamples = []sampleOffset{{.25, .25}, {.75, .75}}
var fourSamples = []sampleOffset{{.375, .125}, {.875, .375}, {.125, .625}, {.625, .875}}

func (r *TerminalRenderer) sampleOffsets() []sampleOffset {
	switch r.sampleCount {
	case 2:
		return twoSamples
	case 4:
		return fourSamples
	default:
		return centerSamples
	}
}
func (r *TerminalRenderer) clearSamples() {
	if r.sampleCount <= 1 {
		return
	}
	count := r.Width * r.Height * r.sampleCount
	if len(r.sampleDepth) != count {
		r.sampleDepth = make([]float64, count)
		r.sampleColor = make([]engine.Color, count)
	}
	for i := range r.sampleDepth {
		r.sampleDepth[i] = math.Inf(1)
		r.sampleColor[i] = engine.ColorBlack
	}
}
func (r *TerminalRenderer) resolvedGlyph(c engine.Color, depth float64) rune {
	if math.IsInf(depth, 1) {
		return ' '
	}
	if r.UseColor {
		return engine.FILLED_CHAR
	}
	brightness := (int(c.R) + int(c.G) + int(c.B)) / 3
	return rune(engine.SHADING_RAMP[brightness*(len(engine.SHADING_RAMP)-1)/255])
}
func (r *TerminalRenderer) resolveSamples() {
	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			var red, green, blue int
			nearest := math.Inf(1)
			for sample := 0; sample < r.sampleCount; sample++ {
				i := (y*r.Width+x)*r.sampleCount + sample
				c := r.sampleColor[i]
				red += int(c.R)
				green += int(c.G)
				blue += int(c.B)
				nearest = math.Min(nearest, r.sampleDepth[i])
			}
			c := engine.Color{R: uint8(red / r.sampleCount), G: uint8(green / r.sampleCount), B: uint8(blue / r.sampleCount)}
			r.ColorBuffer[y][x] = c
			r.ZBuffer[y][x] = nearest
			r.Surface[y][x] = r.resolvedGlyph(c, nearest)
		}
	}
}

// writeCell treats points and wireframe lines as full-cell primitives in MSAA.
func (r *TerminalRenderer) writeCell(x, y int, z float64, c engine.Color, char rune) {
	if !r.acceptLODCell(x, y) {
		return
	}
	if r.sampleCount > 1 {
		for sample := 0; sample < r.sampleCount; sample++ {
			i := (y*r.Width+x)*r.sampleCount + sample
			if z < r.sampleDepth[i] {
				r.sampleDepth[i] = z
				r.sampleColor[i] = c
			}
		}
	}
	if z < r.ZBuffer[y][x] {
		r.ZBuffer[y][x] = z
		r.ColorBuffer[y][x] = c
		if r.UseColor {
			char = engine.FILLED_CHAR
		}
		r.Surface[y][x] = char
	}
}

// acceptLODCell uses complementary coverage, avoiding alpha/depth ordering artifacts.
func (r *TerminalRenderer) acceptLODCell(x, y int) bool {
	if !r.lodMask {
		return true
	}
	pattern := [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}
	incoming := (float64(pattern[y%4][x%4])+.5)/16 < r.lodAlpha
	return incoming == r.lodIncoming
}

func edge(ax, ay, bx, by, px, py float64) float64 { return (bx-ax)*(py-ay) - (by-ay)*(px-ax) }

// fillTriangleWithPerPixelLighting retains subpixel vertices and interpolates
// reciprocal depth. MSAA shades once at the centroid of covered samples per cell.
func (r *TerminalRenderer) fillTriangleWithPerPixelLighting(t *engine.Triangle, camera *engine.Camera, normal engine.Point, _ engine.IMaterial) {
	x0, y0, z0 := camera.ProjectPointFloat(t.P0, r.Height, r.Width)
	x1, y1, z1 := camera.ProjectPointFloat(t.P1, r.Height, r.Width)
	x2, y2, z2 := camera.ProjectPointFloat(t.P2, r.Height, r.Width)
	if z0 <= 0 || z1 <= 0 || z2 <= 0 {
		return
	}
	area := edge(x0, y0, x1, y1, x2, y2)
	if math.Abs(area) < 1e-12 {
		return
	}
	minX := max(r.ClipMinX, int(math.Floor(math.Min(x0, math.Min(x1, x2)))))
	maxX := min(r.ClipMaxX-1, int(math.Floor(math.Max(x0, math.Max(x1, x2)))))
	minY := max(r.ClipMinY, int(math.Floor(math.Min(y0, math.Min(y1, y2)))))
	maxY := min(r.ClipMaxY-1, int(math.Floor(math.Max(y0, math.Max(y1, y2)))))
	samples := r.sampleOffsets()
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !r.acceptLODCell(x, y) {
				continue
			}
			var depths [4]float64
			var visible [4]bool
			var weight0, weight1, weight2 float64
			count, visibleCount := 0, 0
			for s, offset := range samples {
				px, py := float64(x)+offset.x, float64(y)+offset.y
				a := edge(x1, y1, x2, y2, px, py) / area
				b := edge(x2, y2, x0, y0, px, py) / area
				c := 1 - a - b
				if !insideEdge(a, x1, y1, x2, y2, area) || !insideEdge(b, x2, y2, x0, y0, area) || !insideEdge(c, x0, y0, x1, y1, area) {
					continue
				}
				weight0 += a
				weight1 += b
				weight2 += c
				count++
				invZ := a/z0 + b/z1 + c/z2
				z := 1 / invZ
				previous := r.ZBuffer[y][x]
				if r.sampleCount > 1 {
					previous = r.sampleDepth[(y*r.Width+x)*r.sampleCount+s]
				}
				if z < camera.Near || z > camera.Far || z >= previous {
					continue
				}
				visible[s] = true
				depths[s] = z
				visibleCount++
			}
			if visibleCount == 0 {
				continue
			}
			a, b, c := weight0/float64(count), weight1/float64(count), weight2/float64(count)
			invZ := a/z0 + b/z1 + c/z2
			a, b, c = a/z0/invZ, b/z1/invZ, c/z2/invZ
			world := engine.Point{X: a*t.P0.X + b*t.P1.X + c*t.P2.X, Y: a*t.P0.Y + b*t.P1.Y + c*t.P2.Y, Z: a*t.P0.Z + b*t.P1.Z + c*t.P2.Z}
			u, v := a*t.UV0.U+b*t.UV1.U+c*t.UV2.U, a*t.UV0.V+b*t.UV1.V+c*t.UV2.V
			color := r.shadeFragment(t, camera, normal, world, u, v)
			for s := range samples {
				if visible[s] {
					if r.sampleCount > 1 {
						i := (y*r.Width+x)*r.sampleCount + s
						r.sampleDepth[i] = depths[s]
						r.sampleColor[i] = color
					} else {
						r.ZBuffer[y][x] = depths[s]
						r.ColorBuffer[y][x] = color
						r.Surface[y][x] = r.resolvedGlyph(color, depths[s])
					}
				}
			}
		}
	}
}

// insideEdge implements the top-left rule so shared edges belong to one face.
func insideEdge(weight, ax, ay, bx, by, area float64) bool {
	if weight > 0 {
		return true
	}
	if weight < 0 {
		return false
	}
	if area < 0 {
		ax, bx = bx, ax
		ay, by = by, ay
	}
	return by < ay || (by == ay && bx > ax)
}
