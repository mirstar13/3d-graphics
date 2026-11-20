package terminal

import (
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// AARenderer decorates completed frames with a selected color filter.
// MSAA uses per-sample coverage/depth; SSAA renders at twice each dimension.
type AARenderer struct {
	engine.Renderer
	Mode        engine.AAMode
	ssaaSurface [][]rune
	ssaaColor   [][]engine.Color
	ssaaDepth   [][]float64
}

// NewAARenderer creates a filter without allocating unused supersampling buffers.
func NewAARenderer(renderer engine.Renderer, mode engine.AAMode) *AARenderer {
	return &AARenderer{Renderer: renderer, Mode: mode}
}

// RenderWithAA renders scene with anti-aliasing
func (aar *AARenderer) RenderWithAA(scene *engine.Scene) {
	switch aar.Mode {
	case engine.AANone:
		aar.Renderer.RenderScene(scene)
	case engine.AAFXAA:
		// Render normally first
		aar.Renderer.RenderScene(scene)
		// Then apply FXAA post-process
		aar.applyFXAA()
	case engine.AAMSAA2x:
		aar.renderMSAA(scene, 2)
	case engine.AAMSAA4x:
		aar.renderMSAA(scene, 4)
	case engine.AASSAA:
		aar.renderSSAA(scene)
	}
}

// Override RenderScene to use AA
func (aar *AARenderer) RenderScene(scene *engine.Scene) {
	aar.RenderWithAA(scene)
}

// applyFXAA applies Fast Approximate Anti-Aliasing as a post-process
func (aar *AARenderer) applyFXAA() {
	// Access the underlying TerminalRenderer's buffers
	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		return // FXAA only works with TerminalRenderer
	}

	width, height := tr.GetDimensions()

	// Create temporary buffer for output
	output := make([][]engine.Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]engine.Color, width)
		copy(output[i], tr.ColorBuffer[i])
	}

	// FXAA parameters
	const (
		edgeThresholdMin = 0.0312
		edgeThreshold    = 0.125
		subpixelQuality  = 0.75
	)

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Sample 3x3 neighborhood
			center := tr.ColorBuffer[y][x]

			n := tr.ColorBuffer[y-1][x]
			s := tr.ColorBuffer[y+1][x]
			e := tr.ColorBuffer[y][x+1]
			w := tr.ColorBuffer[y][x-1]

			ne := tr.ColorBuffer[y-1][x+1]
			nw := tr.ColorBuffer[y-1][x-1]
			se := tr.ColorBuffer[y+1][x+1]
			sw := tr.ColorBuffer[y+1][x-1]

			// Calculate luminance
			lumCenter := luminance(center)
			lumN := luminance(n)
			lumS := luminance(s)
			lumE := luminance(e)
			lumW := luminance(w)
			lumNE := luminance(ne)
			lumNW := luminance(nw)
			lumSE := luminance(se)
			lumSW := luminance(sw)

			// Find min/max luminance
			lumMin := math.Min(lumCenter, math.Min(math.Min(lumN, lumS), math.Min(lumE, lumW)))
			lumMax := math.Max(lumCenter, math.Max(math.Max(lumN, lumS), math.Max(lumE, lumW)))
			lumRange := lumMax - lumMin

			// Skip if range is too small (not an edge)
			if lumRange < math.Max(edgeThresholdMin, lumMax*edgeThreshold) {
				output[y][x] = center
				continue
			}

			// Calculate edge direction
			horizontal := math.Abs(-2*lumW + -2*lumCenter + -2*lumE + lumNW + lumNE + lumSW + lumSE)
			vertical := math.Abs(-2*lumN + -2*lumCenter + -2*lumS + lumNW + lumNE + lumSW + lumSE)

			isHorizontal := horizontal >= vertical

			// Calculate blend factor
			var blend float64
			if isHorizontal {
				blend = math.Abs((-2*lumN + lumNW + lumNE) - (-2*lumS + lumSW + lumSE))
			} else {
				blend = math.Abs((-2*lumW + lumNW + lumSW) - (-2*lumE + lumNE + lumSE))
			}
			blend = blend / (2 * lumRange)

			// Simple blend along detected edge
			if blend > 0.5 {
				// Average with neighbors along edge direction
				if isHorizontal {
					output[y][x] = averageColors([]engine.Color{center, e, w})
				} else {
					output[y][x] = averageColors([]engine.Color{center, n, s})
				}
			} else {
				output[y][x] = center
			}
		}
	}

	// Copy output back to color buffer
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			tr.ColorBuffer[y][x] = output[y][x]
		}
	}
}

// renderMSAA resolves per-sample coverage and depth after one geometry pass.
func (aar *AARenderer) renderMSAA(scene *engine.Scene, samples int) {
	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		aar.Renderer.RenderScene(scene)
		return
	}
	tr.sampleCount = samples
	defer func() { tr.sampleCount = 0 }()
	aar.Renderer.RenderScene(scene)
	tr.resolveSamples()
}

// renderSSAA shades a 2x2 grid per output cell and box-filters it to native size.
func (aar *AARenderer) renderSSAA(scene *engine.Scene) {
	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		aar.Renderer.RenderScene(scene)
		return
	}
	width, height := tr.GetDimensions()
	surface, colors, depth := tr.Surface, tr.ColorBuffer, tr.ZBuffer
	if len(aar.ssaaSurface) != height*2 || (height > 0 && len(aar.ssaaSurface[0]) != width*2) {
		tr.Resize(width*2, height*2)
		aar.ssaaSurface, aar.ssaaColor, aar.ssaaDepth = tr.Surface, tr.ColorBuffer, tr.ZBuffer
	} else {
		tr.Width, tr.Height = width*2, height*2
		tr.Surface, tr.ColorBuffer, tr.ZBuffer = aar.ssaaSurface, aar.ssaaColor, aar.ssaaDepth
	}
	defer func() {
		tr.Width, tr.Height = width, height
		tr.Surface, tr.ColorBuffer, tr.ZBuffer = surface, colors, depth
		tr.SetClipBounds(0, 0, width, height)
	}()
	aar.Renderer.RenderScene(scene)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var sumR, sumG, sumB int
			nearest := math.Inf(1)
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					c := tr.ColorBuffer[y*2+dy][x*2+dx]
					sumR += int(c.R)
					sumG += int(c.G)
					sumB += int(c.B)
					nearest = math.Min(nearest, tr.ZBuffer[y*2+dy][x*2+dx])
				}
			}
			c := engine.Color{R: uint8(sumR / 4), G: uint8(sumG / 4), B: uint8(sumB / 4)}
			colors[y][x] = c
			depth[y][x] = nearest
			surface[y][x] = tr.resolvedGlyph(c, nearest)
		}
	}
}

// luminance calculates relative luminance
func luminance(c engine.Color) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// averageColors averages multiple colors
func averageColors(colors []engine.Color) engine.Color {
	if len(colors) == 0 {
		return engine.ColorBlack
	}

	var rSum, gSum, bSum int
	for _, c := range colors {
		rSum += int(c.R)
		gSum += int(c.G)
		bSum += int(c.B)
	}

	count := len(colors)
	return engine.Color{
		R: uint8(rSum / count),
		G: uint8(gSum / count),
		B: uint8(bSum / count),
	}
}

// EdgeDetection detects edges for debugging AA
func (aar *AARenderer) EdgeDetection() [][]bool {
	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		return nil
	}

	width, height := tr.GetDimensions()

	edges := make([][]bool, height)
	for i := 0; i < height; i++ {
		edges[i] = make([]bool, width)
	}

	threshold := 30.0 // Luminance difference threshold

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			center := luminance(tr.ColorBuffer[y][x])

			// Check neighbors
			n := luminance(tr.ColorBuffer[y-1][x])
			s := luminance(tr.ColorBuffer[y+1][x])
			e := luminance(tr.ColorBuffer[y][x+1])
			w := luminance(tr.ColorBuffer[y][x-1])

			maxDiff := math.Max(
				math.Max(math.Abs(center-n), math.Abs(center-s)),
				math.Max(math.Abs(center-e), math.Abs(center-w)),
			)

			edges[y][x] = maxDiff > threshold
		}
	}

	return edges
}

// AdaptiveAA applies AA only where edges are detected
func (aar *AARenderer) AdaptiveAA(scene *engine.Scene) {
	// Render normally first
	aar.Renderer.RenderScene(scene)

	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		return
	}

	width, height := tr.GetDimensions()

	// Detect edges
	edges := aar.EdgeDetection()

	// Apply AA only to edge pixels
	output := make([][]engine.Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]engine.Color, width)
		copy(output[i], tr.ColorBuffer[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if edges[y][x] {
				// Apply simple blur to edge pixels
				neighbors := []engine.Color{
					tr.ColorBuffer[y-1][x],
					tr.ColorBuffer[y+1][x],
					tr.ColorBuffer[y][x-1],
					tr.ColorBuffer[y][x+1],
					tr.ColorBuffer[y][x],
				}
				output[y][x] = averageColors(neighbors)
			}
		}
	}

	// Copy back
	for y := 1; y < height-1; y++ {
		copy(tr.ColorBuffer[y], output[y])
	}
}

// TemporalAA applies temporal anti-aliasing (requires frame history)
type TemporalAARenderer struct {
	*AARenderer
	historyBuffer [][]engine.Color
	frameIndex    int
	historyWeight float64
}

// NewTemporalAARenderer creates a TAA renderer
func NewTemporalAARenderer(renderer engine.Renderer) *TemporalAARenderer {
	return &TemporalAARenderer{
		AARenderer:    NewAARenderer(renderer, engine.AANone),
		historyWeight: 0.9,
	}
}

// RenderWithTAA renders with temporal anti-aliasing
func (taa *TemporalAARenderer) RenderWithTAA(scene *engine.Scene) {
	tr, ok := terminalBuffer(taa.Renderer)
	if !ok {
		taa.Renderer.RenderScene(scene)
		return
	}

	width, height := tr.GetDimensions()

	// Render current frame
	taa.Renderer.RenderScene(scene)

	// Initialize history on first frame
	if len(taa.historyBuffer) != height || (height > 0 && len(taa.historyBuffer[0]) != width) {
		taa.historyBuffer = make([][]engine.Color, height)
		for i := 0; i < height; i++ {
			taa.historyBuffer[i] = make([]engine.Color, width)
			copy(taa.historyBuffer[i], tr.ColorBuffer[i])
		}
		return
	}

	// Blend with history
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			current := tr.ColorBuffer[y][x]
			history := taa.historyBuffer[y][x]

			// Weighted blend
			blended := engine.Color{
				R: uint8(float64(history.R)*taa.historyWeight + float64(current.R)*(1.0-taa.historyWeight)),
				G: uint8(float64(history.G)*taa.historyWeight + float64(current.G)*(1.0-taa.historyWeight)),
				B: uint8(float64(history.B)*taa.historyWeight + float64(current.B)*(1.0-taa.historyWeight)),
			}

			tr.ColorBuffer[y][x] = blended
			taa.historyBuffer[y][x] = blended
		}
	}

	taa.frameIndex++
}

// MorphologicalAA applies morphological anti-aliasing (good for terminal rendering)
func (aar *AARenderer) MorphologicalAA() {
	tr, ok := terminalBuffer(aar.Renderer)
	if !ok {
		return
	}

	width, height := tr.GetDimensions()

	// Create edge buffer
	edges := aar.EdgeDetection()

	// Dilate edges
	dilated := make([][]bool, height)
	for i := 0; i < height; i++ {
		dilated[i] = make([]bool, width)
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Dilate if any neighbor is edge
			dilated[y][x] = edges[y][x] ||
				edges[y-1][x] || edges[y+1][x] ||
				edges[y][x-1] || edges[y][x+1]
		}
	}

	// Blend dilated edge pixels
	output := make([][]engine.Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]engine.Color, width)
		copy(output[i], tr.ColorBuffer[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if dilated[y][x] {
				neighbors := []engine.Color{
					tr.ColorBuffer[y-1][x],
					tr.ColorBuffer[y+1][x],
					tr.ColorBuffer[y][x-1],
					tr.ColorBuffer[y][x+1],
					tr.ColorBuffer[y][x],
				}
				output[y][x] = averageColors(neighbors)
			}
		}
	}

	for y := 1; y < height-1; y++ {
		copy(tr.ColorBuffer[y], output[y])
	}
}
