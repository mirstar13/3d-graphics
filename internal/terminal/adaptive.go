package terminal

import (
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

type QualityTier uint8

const (
	FullQuality QualityTier = iota
	ReducedShading
	ReducedGrid
)

// AdaptiveQuality reacts to the previous frame and leaves a quiet interval for recovery.
type AdaptiveQuality struct {
	Tier              QualityTier
	position, forward engine.Point
	ready             bool
	misses, quiet     int
}

func (a *AdaptiveQuality) Update(camera *engine.Camera, work time.Duration, presentationBehind bool, budget time.Duration) QualityTier {
	position, forward := camera.GetPosition(), camera.GetForwardVectorPoint()
	if !a.ready {
		a.position, a.forward, a.ready = position, forward, true
		a.quiet = 8
		return a.Tier
	}
	dx, dy, dz := position.X-a.position.X, position.Y-a.position.Y, position.Z-a.position.Z
	dot := engine.DotProduct(forward.X, forward.Y, forward.Z, a.forward.X, a.forward.Y, a.forward.Z)
	moving := dx*dx+dy*dy+dz*dz > 1e-8 || dot < .999999
	a.position, a.forward = position, forward
	return a.UpdateMotion(moving, work, presentationBehind, budget)
}

// UpdateMotion handles controllers that distinguish user-driven movement from
// small physics adjustments to the camera while the controls are idle.
func (a *AdaptiveQuality) UpdateMotion(moving bool, work time.Duration, presentationBehind bool, budget time.Duration) QualityTier {
	if !moving {
		a.quiet++
		if a.quiet >= 8 {
			a.Tier = FullQuality
			a.misses = 0
			return a.Tier
		}
	} else {
		a.quiet = 0
	}
	behind := work > budget || presentationBehind
	if a.Tier == ReducedShading {
		behind = presentationBehind
	}
	if !behind || a.Tier == ReducedGrid {
		a.misses = 0
		return a.Tier
	}
	a.misses++
	if a.misses >= 4 {
		a.misses = 0
		a.Tier++
	}
	return a.Tier
}

type frameGrid struct {
	width, height int
	glyphs        [][]rune
	colors        [][]engine.Color
	depth         [][]float64
}

func (r *TerminalRenderer) swapGrid(g *frameGrid) {
	r.Width, g.width = g.width, r.Width
	r.Height, g.height = g.height, r.Height
	r.Surface, g.glyphs = g.glyphs, r.Surface
	r.ColorBuffer, g.colors = g.colors, r.ColorBuffer
	r.ZBuffer, g.depth = g.depth, r.ZBuffer
}

// RenderQuality expands only the scene buffers. Draw overlays after it returns.
func (r *TerminalRenderer) RenderQuality(tier QualityTier, draw func()) (rasterWidth, rasterHeight int) {
	r.cheapShading = tier != FullQuality
	defer func() { r.cheapShading = false }()
	w, h := r.Width, r.Height
	if tier != ReducedGrid || min(w, h) < 4 {
		draw()
		return w, h
	}
	r.swapGrid(&r.adaptiveGrid)
	// Avoid touching presentation history when switching raster buffers.
	sw, sh := w*3/4, h*3/4
	if r.Width != sw || r.Height != sh {
		r.allocateGrid(sw, sh)
	}
	draw()
	r.swapGrid(&r.adaptiveGrid)
	src := &r.adaptiveGrid
	for y := range r.Surface {
		sy := y * sh / h
		for x := range r.Surface[y] {
			sx := x * sw / w
			r.Surface[y][x] = src.glyphs[sy][sx]
			r.ColorBuffer[y][x] = src.colors[sy][sx]
			r.ZBuffer[y][x] = src.depth[sy][sx]
		}
	}
	return sw, sh
}

// ReducedShadingActive reports whether the current draw omits FXAA.
func (r *TerminalRenderer) ReducedShadingActive() bool {
	return r.cheapShading
}
