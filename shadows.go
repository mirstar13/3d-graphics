package main

import "math"

// ShadowMap represents a depth buffer for shadow mapping
type ShadowMap struct {
	Width       int
	Height      int
	DepthBuffer [][]float64
	LightMatrix Matrix4x4
	LightPos    Point
	Resolution  int
	Bias        float64
	PCFSamples  int
}

// NewShadowMap creates a new shadow map
func NewShadowMap(resolution int) *ShadowMap {
	depthBuffer := make([][]float64, resolution)
	for i := range depthBuffer {
		depthBuffer[i] = make([]float64, resolution)
		for j := range depthBuffer[i] {
			depthBuffer[i][j] = math.Inf(1)
		}
	}

	return &ShadowMap{
		Width:       resolution,
		Height:      resolution,
		DepthBuffer: depthBuffer,
		Resolution:  resolution,
		Bias:        0.005,
		PCFSamples:  2,
	}
}

// Clear clears the shadow map
func (sm *ShadowMap) Clear() {
	for i := range sm.DepthBuffer {
		for j := range sm.DepthBuffer[i] {
			sm.DepthBuffer[i][j] = math.Inf(1)
		}
	}
}

// SetupLightView calculates the light view matrix
func (sm *ShadowMap) SetupLightView(light *Light, target Point, near, far float64) {
	// Calculate light direction
	lightDir := Point{
		X: target.X - light.Position.X,
		Y: target.Y - light.Position.Y,
		Z: target.Z - light.Position.Z,
	}
	lightDir.X, lightDir.Y, lightDir.Z = normalizeVector(lightDir.X, lightDir.Y, lightDir.Z)

	// Create light view matrix (look at target from light position)
	// Handle case where light is directly above/below target
	up := Point{X: 0, Y: 1, Z: 0}
	if math.Abs(lightDir.X) < 0.01 && math.Abs(lightDir.Z) < 0.01 {
		up = Point{X: 0, Y: 0, Z: 1}
	}
	viewMatrix := CreateLookAtMatrix(light.Position, target, up)

	// Create orthographic projection for directional lights
	// Size of the shadow map frustum (adjust based on scene size)
	// Increased from 20.0 to 40.0 to cover larger scenes and reduce edge clipping.
	// This trades shadow resolution for coverage area - adjust based on scene requirements.
	size := 40.0
	projMatrix := CreateOrthographicMatrix(-size, size, -size, size, near, far)

	// Combine view and projection matrices
	sm.LightMatrix = projMatrix.Multiply(viewMatrix)
	sm.LightPos = light.Position
}

// ProjectToShadowMap projects a world point to shadow map coordinates
func (sm *ShadowMap) ProjectToShadowMap(worldPos Point) (x, y int, depth float64, valid bool) {
	// Transform to light space
	transformed := sm.LightMatrix.MultiplyPoint(worldPos)

	// Convert to shadow map coordinates
	x = int((transformed.X + 1.0) * float64(sm.Width) * 0.5)
	y = int((transformed.Y + 1.0) * float64(sm.Height) * 0.5)
	depth = transformed.Z

	// Check bounds
	if x < 0 || x >= sm.Width || y < 0 || y >= sm.Height {
		return 0, 0, 0, false
	}

	return x, y, depth, true
}

// WriteDepth writes a depth value to the shadow map
func (sm *ShadowMap) WriteDepth(x, y int, depth float64) {
	if x >= 0 && x < sm.Width && y >= 0 && y < sm.Height {
		if depth < sm.DepthBuffer[y][x] {
			sm.DepthBuffer[y][x] = depth
		}
	}
}

// SampleDepth samples the shadow map at a position
func (sm *ShadowMap) SampleDepth(x, y int) float64 {
	if x < 0 || x >= sm.Width || y < 0 || y >= sm.Height {
		return math.Inf(1)
	}
	return sm.DepthBuffer[y][x]
}

// IsInShadow checks if a point is in shadow (simple test)
func (sm *ShadowMap) IsInShadow(worldPos Point) bool {
	x, y, depth, valid := sm.ProjectToShadowMap(worldPos)
	if !valid {
		return false
	}

	shadowDepth := sm.SampleDepth(x, y)
	return depth > shadowDepth+sm.Bias
}

// CalculateShadow calculates shadow factor (0 = full shadow, 1 = no shadow)
func (sm *ShadowMap) CalculateShadow(worldPos Point) float64 {
	x, y, depth, valid := sm.ProjectToShadowMap(worldPos)
	if !valid {
		return 1.0 // Outside shadow map = no shadow
	}

	// PCF (Percentage Closer Filtering)
	if sm.PCFSamples <= 1 {
		// Simple shadow test
		shadowDepth := sm.SampleDepth(x, y)
		if depth > shadowDepth+sm.Bias {
			return 0.0 // In shadow
		}
		return 1.0 // Not in shadow
	}

	// PCF sampling
	shadowFactor := 0.0
	sampleCount := 0

	for dx := -sm.PCFSamples; dx <= sm.PCFSamples; dx++ {
		for dy := -sm.PCFSamples; dy <= sm.PCFSamples; dy++ {
			sx := x + dx
			sy := y + dy
			if sx >= 0 && sx < sm.Width && sy >= 0 && sy < sm.Height {
				shadowDepth := sm.SampleDepth(sx, sy)
				if depth <= shadowDepth+sm.Bias {
					shadowFactor += 1.0
				}
				sampleCount++
			}
		}
	}

	if sampleCount == 0 {
		return 1.0
	}

	return shadowFactor / float64(sampleCount)
}

// ShadowRenderer interface for rendering shadows
type ShadowRenderer interface {
	RenderShadowMap(light *Light, scene *Scene) *ShadowMap
}

// SimpleShadowRenderer implements basic shadow mapping
type SimpleShadowRenderer struct {
	ShadowMaps map[*Light]*ShadowMap
	Resolution int
}

// NewSimpleShadowRenderer creates a shadow renderer
func NewSimpleShadowRenderer(resolution int) *SimpleShadowRenderer {
	return &SimpleShadowRenderer{
		ShadowMaps: make(map[*Light]*ShadowMap),
		Resolution: resolution,
	}
}

// RenderShadowMap renders a shadow map for a light
func (sr *SimpleShadowRenderer) RenderShadowMap(light *Light, scene *Scene) *ShadowMap {
	// Get or create shadow map for this light
	shadowMap, exists := sr.ShadowMaps[light]
	if !exists {
		shadowMap = NewShadowMap(sr.Resolution)
		sr.ShadowMaps[light] = shadowMap
	}

	shadowMap.Clear()

	// Setup light view
	sceneCenter := Point{X: 0, Y: 0, Z: 0} // Could calculate from scene bounds
	shadowMap.SetupLightView(light, sceneCenter, 0.1, 100.0)

	// Render scene from light's perspective
	renderables := scene.GetRenderableNodes()
	for _, node := range renderables {
		// Transform object and rasterize to shadow map
		sr.renderNodeToShadowMap(node, shadowMap)
	}

	return shadowMap
}

// renderNodeToShadowMap renders a node to the shadow map
func (sr *SimpleShadowRenderer) renderNodeToShadowMap(node *SceneNode, shadowMap *ShadowMap) {
	transformed := node.TransformSceneObject()

	switch obj := transformed.(type) {
	case *Mesh:
		// Rasterize mesh triangles to shadow map
		for i := 0; i < len(obj.Indices); i += 3 {
			if i+2 >= len(obj.Indices) {
				break
			}

			v0 := obj.Vertices[obj.Indices[i]]
			v1 := obj.Vertices[obj.Indices[i+1]]
			v2 := obj.Vertices[obj.Indices[i+2]]

			// Project triangle vertices
			x0, y0, z0, valid0 := shadowMap.ProjectToShadowMap(v0)
			x1, y1, z1, valid1 := shadowMap.ProjectToShadowMap(v1)
			x2, y2, z2, valid2 := shadowMap.ProjectToShadowMap(v2)

			if !valid0 || !valid1 || !valid2 {
				continue
			}

			// Rasterize triangle to depth buffer
			sr.rasterizeDepthTriangle(shadowMap, x0, y0, z0, x1, y1, z1, x2, y2, z2)
		}

	case *Triangle:
		// Rasterize single triangle
		x0, y0, z0, valid0 := shadowMap.ProjectToShadowMap(obj.P0)
		x1, y1, z1, valid1 := shadowMap.ProjectToShadowMap(obj.P1)
		x2, y2, z2, valid2 := shadowMap.ProjectToShadowMap(obj.P2)

		if valid0 && valid1 && valid2 {
			sr.rasterizeDepthTriangle(shadowMap, x0, y0, z0, x1, y1, z1, x2, y2, z2)
		}
	}
}

// rasterizeDepthTriangle rasterizes a triangle to the depth buffer
func (sr *SimpleShadowRenderer) rasterizeDepthTriangle(
	shadowMap *ShadowMap,
	x0, y0 int, z0 float64,
	x1, y1 int, z1 float64,
	x2, y2 int, z2 float64,
) {
	// Find bounding box
	minX := minInt(minInt(x0, x1), x2)
	maxX := maxInt(maxInt(x0, x1), x2)
	minY := minInt(minInt(y0, y1), y2)
	maxY := maxInt(maxInt(y0, y1), y2)

	// Clamp to shadow map bounds
	minX = maxInt(minX, 0)
	maxX = minInt(maxX, shadowMap.Width-1)
	minY = maxInt(minY, 0)
	maxY = minInt(maxY, shadowMap.Height-1)

	// Rasterize
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			// Calculate barycentric coordinates
			w0, w1, w2 := calculateBarycentricInt(x, y, x0, y0, x1, y1, x2, y2)

			// Check if point is inside triangle
			if w0 >= 0 && w1 >= 0 && w2 >= 0 {
				// Interpolate depth
				depth := w0*z0 + w1*z1 + w2*z2

				// Write to depth buffer
				shadowMap.WriteDepth(x, y, depth)
			}
		}
	}
}

// calculateBarycentricInt calculates barycentric coordinates for integer points
func calculateBarycentricInt(px, py, x0, y0, x1, y1, x2, y2 int) (float64, float64, float64) {
	v0x := float64(x1 - x0)
	v0y := float64(y1 - y0)
	v1x := float64(x2 - x0)
	v1y := float64(y2 - y0)
	v2x := float64(px - x0)
	v2y := float64(py - y0)

	d00 := v0x*v0x + v0y*v0y
	d01 := v0x*v1x + v0y*v1y
	d11 := v1x*v1x + v1y*v1y
	d20 := v2x*v0x + v2y*v0y
	d21 := v2x*v1x + v2y*v1y

	denom := d00*d11 - d01*d01
	if math.Abs(denom) < 1e-10 {
		return -1, -1, -1
	}

	v := (d11*d20 - d01*d21) / denom
	w := (d00*d21 - d01*d20) / denom
	u := 1.0 - v - w

	return u, v, w
}

// CreateLookAtMatrix creates a view matrix looking at a target
func CreateLookAtMatrix(eye, target, up Point) Matrix4x4 {
	// Calculate forward vector (from eye to target)
	forward := Point{
		X: target.X - eye.X,
		Y: target.Y - eye.Y,
		Z: target.Z - eye.Z,
	}
	forward.X, forward.Y, forward.Z = normalizeVector(forward.X, forward.Y, forward.Z)

	// Calculate right vector
	rightX, rightY, rightZ := crossProduct(forward.X, forward.Y, forward.Z, up.X, up.Y, up.Z)
	rightX, rightY, rightZ = normalizeVector(rightX, rightY, rightZ)

	// Recalculate up vector
	upX, upY, upZ := crossProduct(rightX, rightY, rightZ, forward.X, forward.Y, forward.Z)

	// Create view matrix
	mat := Matrix4x4{}
	mat.M[0], mat.M[1], mat.M[2] = rightX, upX, -forward.X
	mat.M[4], mat.M[5], mat.M[6] = rightY, upY, -forward.Y
	mat.M[8], mat.M[9], mat.M[10] = rightZ, upZ, -forward.Z
	mat.M[3] = -dotProduct(rightX, rightY, rightZ, eye.X, eye.Y, eye.Z)
	mat.M[7] = -dotProduct(upX, upY, upZ, eye.X, eye.Y, eye.Z)
	mat.M[11] = dotProduct(forward.X, forward.Y, forward.Z, eye.X, eye.Y, eye.Z)
	mat.M[15] = 1.0

	return mat
}
