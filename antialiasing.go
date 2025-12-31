package main

import "math"

// AAMode specifies anti-aliasing mode
type AAMode int

const (
	AANone   AAMode = iota
	AAFXAA          // Fast Approximate Anti-Aliasing
	AAMSAA2x        // 2x Multi-Sample Anti-Aliasing
	AAMSAA4x        // 4x Multi-Sample Anti-Aliasing
	AASSAA          // Super-Sample Anti-Aliasing
)

// AARenderer wraps a renderer with AA capabilities
type AARenderer struct {
	Renderer       // Embed base renderer
	Mode           AAMode
	supersampleBuf [][]Color   // For SSAA
	supersampleZ   [][]float64 // Z-buffer for supersampling
	samples        int         // Number of samples per pixel
	SSAAFactor     int         // Super-sampling factor (2 = 2x2, 4 = 4x4)
}

// NewAARenderer creates an AA-capable renderer
func NewAARenderer(renderer Renderer, mode AAMode) *AARenderer {
	aar := &AARenderer{
		Renderer: renderer,
		Mode:     mode,
	}

	// Initialize buffers based on mode
	switch mode {
	case AASSAA:
		aar.samples = 4
		aar.SSAAFactor = 2
		aar.initSupersampleBuffers(2) // 2x supersampling
	case AAMSAA4x:
		aar.samples = 4
	case AAMSAA2x:
		aar.samples = 2
	}

	return aar
}

// initSupersampleBuffers initializes supersampling buffers
func (aar *AARenderer) initSupersampleBuffers(factor int) {
	width, height := aar.Renderer.GetDimensions()

	ssWidth := width * factor
	ssHeight := height * factor

	aar.supersampleBuf = make([][]Color, ssHeight)
	aar.supersampleZ = make([][]float64, ssHeight)

	for i := 0; i < ssHeight; i++ {
		aar.supersampleBuf[i] = make([]Color, ssWidth)
		aar.supersampleZ[i] = make([]float64, ssWidth)
		for j := 0; j < ssWidth; j++ {
			aar.supersampleBuf[i][j] = ColorBlack
			aar.supersampleZ[i][j] = math.Inf(1)
		}
	}
}

// RenderWithAA renders scene with anti-aliasing
func (aar *AARenderer) RenderWithAA(scene *Scene) {
	switch aar.Mode {
	case AANone:
		aar.Renderer.RenderScene(scene)
	case AAFXAA:
		// Render normally first
		aar.Renderer.RenderScene(scene)
		// Then apply FXAA post-process
		aar.applyFXAA()
	case AAMSAA2x:
		aar.renderMSAA(scene, 2)
	case AAMSAA4x:
		aar.renderMSAA(scene, 4)
	case AASSAA:
		aar.renderSSAA(scene, 2)
	}
}

// Override RenderScene to use AA
func (aar *AARenderer) RenderScene(scene *Scene) {
	aar.RenderWithAA(scene)
}

func (aar *AARenderer) ClearBuffers() {
	width, height := aar.Renderer.GetDimensions()

	// Clear supersample buffers if they exist
	if aar.supersampleBuf != nil {
		ssHeight := height * aar.SSAAFactor
		ssWidth := width * aar.SSAAFactor
		for y := 0; y < ssHeight; y++ {
			for x := 0; x < ssWidth; x++ {
				aar.supersampleBuf[y][x] = ColorBlack
				aar.supersampleZ[y][x] = math.Inf(1)
			}
		}
	}
}

// applyFXAA applies Fast Approximate Anti-Aliasing as a post-process
func (aar *AARenderer) applyFXAA() {
	// Access the underlying TerminalRenderer's buffers
	tr, ok := aar.Renderer.(*TerminalRenderer)
	if !ok {
		return // FXAA only works with TerminalRenderer
	}

	width, height := tr.GetDimensions()

	// Create temporary buffer for output
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
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
					output[y][x] = averageColors([]Color{center, e, w})
				} else {
					output[y][x] = averageColors([]Color{center, n, s})
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

// renderMSAA renders with Multi-Sample Anti-Aliasing
func (aar *AARenderer) renderMSAA(scene *Scene, samples int) {
	// For MSAA, we render multiple times with slight sub-pixel offsets
	// and average the results. This is a simplified implementation.

	tr, ok := aar.Renderer.(*TerminalRenderer)
	if !ok {
		aar.Renderer.RenderScene(scene)
		return
	}

	width, height := tr.GetDimensions()

	// MSAA sample patterns (sub-pixel offsets)
	var sampleOffsets [][2]float64

	switch samples {
	case 2:
		sampleOffsets = [][2]float64{
			{-0.25, -0.25},
			{0.25, 0.25},
		}
	case 4:
		sampleOffsets = [][2]float64{
			{-0.375, -0.125},
			{0.125, -0.375},
			{-0.125, 0.375},
			{0.375, 0.125},
		}
	}

	_ = sampleOffsets

	// Accumulation buffers
	accumR := make([][]float64, height)
	accumG := make([][]float64, height)
	accumB := make([][]float64, height)
	for i := range accumR {
		accumR[i] = make([]float64, width)
		accumG[i] = make([]float64, width)
		accumB[i] = make([]float64, width)
	}

	// Render multiple samples
	for s := 0; s < samples; s++ {
		// Note: In a full implementation, we would jitter the camera slightly
		// For now, we render the same scene and blend
		aar.Renderer.RenderScene(scene)

		// Accumulate this sample
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := tr.ColorBuffer[y][x]
				accumR[y][x] += float64(c.R)
				accumG[y][x] += float64(c.G)
				accumB[y][x] += float64(c.B)
			}
		}
	}

	// Average the samples
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tr.ColorBuffer[y][x] = Color{
				R: uint8(accumR[y][x] / float64(samples)),
				G: uint8(accumG[y][x] / float64(samples)),
				B: uint8(accumB[y][x] / float64(samples)),
			}
		}
	}
}

// renderSSAA renders with Super-Sample Anti-Aliasing
func (aar *AARenderer) renderSSAA(scene *Scene, factor int) {
	// Note: True SSAA requires rendering at higher resolution
	// This is a simplified version that renders once and applies filtering

	aar.Renderer.RenderScene(scene)

	// In a full implementation, you would:
	// 1. Create a high-resolution renderer (width*factor, height*factor)
	// 2. Render to that buffer
	// 3. Downsample using box filter or better

	// For now, we apply a simple blur effect
	aar.applySimpleBlur()
}

// applySimpleBlur applies a simple blur to simulate SSAA
func (aar *AARenderer) applySimpleBlur() {
	tr, ok := aar.Renderer.(*TerminalRenderer)
	if !ok {
		return
	}

	width, height := tr.GetDimensions()
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
	}

	// 3x3 box blur
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			colors := []Color{
				tr.ColorBuffer[y-1][x-1], tr.ColorBuffer[y-1][x], tr.ColorBuffer[y-1][x+1],
				tr.ColorBuffer[y][x-1], tr.ColorBuffer[y][x], tr.ColorBuffer[y][x+1],
				tr.ColorBuffer[y+1][x-1], tr.ColorBuffer[y+1][x], tr.ColorBuffer[y+1][x+1],
			}
			output[y][x] = averageColors(colors)
		}
	}

	// Copy back (except edges)
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			tr.ColorBuffer[y][x] = output[y][x]
		}
	}
}

// luminance calculates relative luminance
func luminance(c Color) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// averageColors averages multiple colors
func averageColors(colors []Color) Color {
	if len(colors) == 0 {
		return ColorBlack
	}

	var rSum, gSum, bSum int
	for _, c := range colors {
		rSum += int(c.R)
		gSum += int(c.G)
		bSum += int(c.B)
	}

	count := len(colors)
	return Color{
		R: uint8(rSum / count),
		G: uint8(gSum / count),
		B: uint8(bSum / count),
	}
}

// EdgeDetection detects edges for debugging AA
func (aar *AARenderer) EdgeDetection() [][]bool {
	tr, ok := aar.Renderer.(*TerminalRenderer)
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
func (aar *AARenderer) AdaptiveAA(scene *Scene) {
	// Render normally first
	aar.Renderer.RenderScene(scene)

	tr, ok := aar.Renderer.(*TerminalRenderer)
	if !ok {
		return
	}

	width, height := tr.GetDimensions()

	// Detect edges
	edges := aar.EdgeDetection()

	// Apply AA only to edge pixels
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
		copy(output[i], tr.ColorBuffer[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if edges[y][x] {
				// Apply simple blur to edge pixels
				neighbors := []Color{
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
	historyBuffer [][]Color
	frameIndex    int
	historyWeight float64
}

// NewTemporalAARenderer creates a TAA renderer
func NewTemporalAARenderer(renderer Renderer) *TemporalAARenderer {
	return &TemporalAARenderer{
		AARenderer:    NewAARenderer(renderer, AANone),
		historyWeight: 0.9,
	}
}

// RenderWithTAA renders with temporal anti-aliasing
func (taa *TemporalAARenderer) RenderWithTAA(scene *Scene) {
	tr, ok := taa.Renderer.(*TerminalRenderer)
	if !ok {
		taa.Renderer.RenderScene(scene)
		return
	}

	width, height := tr.GetDimensions()

	// Render current frame
	taa.Renderer.RenderScene(scene)

	// Initialize history on first frame
	if taa.historyBuffer == nil {
		taa.historyBuffer = make([][]Color, height)
		for i := 0; i < height; i++ {
			taa.historyBuffer[i] = make([]Color, width)
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
			blended := Color{
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
	tr, ok := aar.Renderer.(*TerminalRenderer)
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
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
		copy(output[i], tr.ColorBuffer[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if dilated[y][x] {
				neighbors := []Color{
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
