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
	Renderer
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
		aar.RenderScene(scene)
	case AAFXAA:
		aar.RenderScene(scene)
		aar.applyFXAA()
	case AAMSAA2x:
		aar.renderMSAA(scene, 2)
	case AAMSAA4x:
		aar.renderMSAA(scene, 4)
	case AASSAA:
		aar.renderSSAA(scene, 2)
	}
}

func (aar *AARenderer) ClearBuffers() {
	// Clear supersample buffers if they exist
	for y := range aar.supersampleBuf {
		for x := range aar.supersampleBuf[y] {
			aar.supersampleBuf[y][x] = ColorBlack
		}
	}
	for y := range aar.supersampleZ {
		for x := range aar.supersampleZ[y] {
			aar.supersampleZ[y][x] = math.Inf(1)
		}
	}
}

// applyFXAA applies Fast Approximate Anti-Aliasing
func (aar *AARenderer) applyFXAA() {
	width, height := aar.Renderer.GetDimensions()

	// Create temporary buffer for output
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
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
			center := aar.supersampleBuf[y][x]

			n := aar.supersampleBuf[y-1][x]
			s := aar.supersampleBuf[y+1][x]
			e := aar.supersampleBuf[y][x+1]
			w := aar.supersampleBuf[y][x-1]

			ne := aar.supersampleBuf[y-1][x+1]
			nw := aar.supersampleBuf[y-1][x-1]
			se := aar.supersampleBuf[y+1][x+1]
			sw := aar.supersampleBuf[y+1][x-1]

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

			// Skip if range is too small
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

			// Simple blend
			if blend > 0.5 {
				// Average with neighbors
				avgColor := averageColors([]Color{center, n, s, e, w})
				output[y][x] = avgColor
			} else {
				output[y][x] = center
			}
		}
	}

	// Copy output back to color buffer
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			aar.supersampleBuf[y][x] = output[y][x]
		}
	}
}

// renderMSAA renders with Multi-Sample Anti-Aliasing
func (aar *AARenderer) renderMSAA(scene *Scene, samples int) {
	width, height := aar.Renderer.GetDimensions()

	// Clear buffers
	aar.ClearBuffers()

	// MSAA sample patterns (offsets within pixel)
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

	// Render scene multiple times with jittered camera
	sampleColors := make([][][]Color, samples)

	for s := 0; s < samples; s++ {
		// Jitter camera slightly
		jitterX := sampleOffsets[s][0] / float64(width)
		jitterY := sampleOffsets[s][1] / float64(height)

		// TODO: Apply jitter to projection matrix
		_ = jitterX
		_ = jitterY

		// Render
		aar.RenderScene(scene)

		// Store sample
		sampleColors[s] = make([][]Color, height)
		for y := 0; y < height; y++ {
			sampleColors[s][y] = make([]Color, width)
			copy(sampleColors[s][y], aar.supersampleBuf[y])
		}

		// Clear for next sample
		aar.ClearBuffers()
	}

	// Resolve samples by averaging
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			colors := make([]Color, samples)
			for s := 0; s < samples; s++ {
				colors[s] = sampleColors[s][y][x]
			}
			aar.supersampleBuf[y][x] = averageColors(colors)
		}
	}
}

// renderSSAA renders with Super-Sample Anti-Aliasing
func (aar *AARenderer) renderSSAA(scene *Scene, factor int) {
	width, height := aar.Renderer.GetDimensions()

	// Ensure supersample buffers are initialized
	if aar.supersampleBuf == nil {
		aar.initSupersampleBuffers(factor)
	}

	ssWidth := width * factor
	ssHeight := height * factor

	// Clear supersample buffers
	for y := 0; y < ssHeight; y++ {
		for x := 0; x < ssWidth; x++ {
			aar.supersampleBuf[y][x] = ColorBlack
			aar.supersampleZ[y][x] = math.Inf(1)
		}
	}

	// Note: True SSAA would require rendering at higher resolution
	// which needs a separate high-res renderer. For now, we render
	// at normal resolution and apply the downsampling filter.
	aar.RenderScene(scene)

	// For a proper implementation, you would need to:
	// 1. Create a new renderer at ssWidth x ssHeight
	// 2. Render the scene to that renderer
	// 3. Copy colors to supersampleBuf
	// 4. Downsample

	// Current simplified approach: just render normally
	// The AA effect comes from the downsample averaging
	aar.downsample(factor)
}

// downsample downsamples the supersample buffer
func (aar *AARenderer) downsample(factor int) {
	width, height := aar.Renderer.GetDimensions()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Average factor x factor block
			var rSum, gSum, bSum int
			count := 0

			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					sx := x*factor + dx
					sy := y*factor + dy

					if sy < len(aar.supersampleBuf) && sx < len(aar.supersampleBuf[sy]) {
						c := aar.supersampleBuf[sy][sx]
						rSum += int(c.R)
						gSum += int(c.G)
						bSum += int(c.B)
						count++
					}
				}
			}

			if count > 0 {
				aar.supersampleBuf[y][x] = Color{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
				}
			}
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
	width, height := aar.Renderer.GetDimensions()

	edges := make([][]bool, height)
	for i := 0; i < height; i++ {
		edges[i] = make([]bool, width)
	}

	threshold := 30.0 // Luminance difference threshold

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			center := luminance(aar.supersampleBuf[y][x])

			// Check neighbors
			n := luminance(aar.supersampleBuf[y-1][x])
			s := luminance(aar.supersampleBuf[y+1][x])
			e := luminance(aar.supersampleBuf[y][x+1])
			w := luminance(aar.supersampleBuf[y][x-1])

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
	width, height := aar.Renderer.GetDimensions()

	// Render normally first
	aar.RenderScene(scene)

	// Detect edges
	edges := aar.EdgeDetection()

	// Apply AA only to edge pixels
	output := make([][]Color, height)
	for i := 0; i < height; i++ {
		output[i] = make([]Color, width)
		copy(output[i], aar.supersampleBuf[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if edges[y][x] {
				// Apply simple blur to edge pixels
				neighbors := []Color{
					aar.supersampleBuf[y-1][x],
					aar.supersampleBuf[y+1][x],
					aar.supersampleBuf[y][x-1],
					aar.supersampleBuf[y][x+1],
					aar.supersampleBuf[y][x],
				}
				output[y][x] = averageColors(neighbors)
			}
		}
	}

	// Copy back
	aar.supersampleBuf = output
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
	width, height := taa.Renderer.GetDimensions()

	// Render current frame
	taa.RenderScene(scene)

	// Initialize history on first frame
	if taa.historyBuffer == nil {
		taa.historyBuffer = make([][]Color, height)
		for i := 0; i < height; i++ {
			taa.historyBuffer[i] = make([]Color, width)
			copy(taa.historyBuffer[i], taa.supersampleBuf[i])
		}
		return
	}

	// Blend with history
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			current := taa.supersampleBuf[y][x]
			history := taa.historyBuffer[y][x]

			// Weighted blend
			blended := Color{
				R: uint8(float64(history.R)*taa.historyWeight + float64(current.R)*(1.0-taa.historyWeight)),
				G: uint8(float64(history.G)*taa.historyWeight + float64(current.G)*(1.0-taa.historyWeight)),
				B: uint8(float64(history.B)*taa.historyWeight + float64(current.B)*(1.0-taa.historyWeight)),
			}

			taa.supersampleBuf[y][x] = blended
			taa.historyBuffer[y][x] = blended
		}
	}

	taa.frameIndex++
}

// MorphologicalAA applies morphological anti-aliasing (good for terminal rendering)
func (aar *AARenderer) MorphologicalAA() {
	width, height := aar.Renderer.GetDimensions()

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
		copy(output[i], aar.supersampleBuf[i])
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if dilated[y][x] {
				neighbors := []Color{
					aar.supersampleBuf[y-1][x],
					aar.supersampleBuf[y+1][x],
					aar.supersampleBuf[y][x-1],
					aar.supersampleBuf[y][x+1],
					aar.supersampleBuf[y][x],
				}
				output[y][x] = averageColors(neighbors)
			}
		}
	}

	aar.supersampleBuf = output
}
