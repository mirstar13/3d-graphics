package main

import (
	"bufio"
	"fmt"
	"math"
	"strings"
)

// TerminalRenderer renders to a terminal using ANSI escape codes
type TerminalRenderer struct {
	Writer         *bufio.Writer
	Height         int
	Width          int
	Surface        [][]rune
	ColorBuffer    [][]Color
	ZBuffer        [][]float64
	Charset        [9]rune
	UseColor       bool
	LightingSystem *LightingSystem
	ShadowRenderer *SimpleShadowRenderer
	Camera         *Camera
	ShowDebugInfo  bool
	debugBuffer    strings.Builder
	lastDebugLine  string

	// Clipping bounds (inclusive min, exclusive max)
	ClipMinX, ClipMinY int
	ClipMaxX, ClipMaxY int
}

// NewTerminalRenderer creates a new terminal renderer
func NewTerminalRenderer(writer *bufio.Writer, height, width int) *TerminalRenderer {
	surface := make([][]rune, height)
	colorBuffer := make([][]Color, height)
	zBuffer := make([][]float64, height)

	for i := range surface {
		surface[i] = make([]rune, width)
		colorBuffer[i] = make([]Color, width)
		zBuffer[i] = make([]float64, width)
		for j := range surface[i] {
			surface[i][j] = DefaultCharset[0]
			colorBuffer[i][j] = ColorBlack
			zBuffer[i][j] = math.Inf(1)
		}
	}

	return &TerminalRenderer{
		Writer:        writer,
		Height:        height,
		Width:         width,
		Surface:       surface,
		ColorBuffer:   colorBuffer,
		ZBuffer:       zBuffer,
		Charset:       DefaultCharset,
		UseColor:      true,
		ShowDebugInfo:  true,
		ShadowRenderer: NewSimpleShadowRenderer(512), // Moderate resolution for CPU rendering
		ClipMinX:       0,
		ClipMinY:       0,
		ClipMaxX:       width,
		ClipMaxY:       height,
	}
}

// SetClipBounds sets the clipping region for subsequent draw calls
func (r *TerminalRenderer) SetClipBounds(minX, minY, maxX, maxY int) {
	r.ClipMinX = clampInt(minX, 0, r.Width)
	r.ClipMinY = clampInt(minY, 0, r.Height)
	r.ClipMaxX = clampInt(maxX, 0, r.Width)
	r.ClipMaxY = clampInt(maxY, 0, r.Height)
}

// Initialize sets up the terminal renderer
func (r *TerminalRenderer) Initialize() error {
	if r.Writer == nil {
		return fmt.Errorf("writer is nil")
	}
	// Enter alternate screen buffer
	r.Writer.WriteString("\033[?1049h")
	// Hide cursor
	r.Writer.WriteString("\033[?25l")
	// Clear screen
	r.Writer.WriteString("\033[2J\033[H")
	r.Writer.Flush()
	return nil
}

// Shutdown cleans up the terminal renderer
func (r *TerminalRenderer) Shutdown() {
	if r.Writer != nil {
		r.Writer.WriteString("\033[?25h")   // Show cursor
		r.Writer.WriteString("\033[?1049l") // Exit alternate screen
		r.Writer.Flush()
	}
}

// BeginFrame clears all buffers
func (r *TerminalRenderer) BeginFrame() {
	// Reset clipping to full screen
	r.ClipMinX = 0
	r.ClipMinY = 0
	r.ClipMaxX = r.Width
	r.ClipMaxY = r.Height

	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			r.Surface[y][x] = ' '
			r.ColorBuffer[y][x] = ColorBlack
			r.ZBuffer[y][x] = math.Inf(1)
		}
	}
}

// EndFrame does nothing for terminal renderer
func (r *TerminalRenderer) EndFrame() {
	// No-op for terminal
}

func (r *TerminalRenderer) GetRenderContext() *RenderContext {
	return &RenderContext{
		Camera:         r.Camera,
		LightingSystem: r.LightingSystem,
		ViewFrustum:    nil,
	}
}

// Present writes the frame to the terminal
func (r *TerminalRenderer) Present() {
	builder := strings.Builder{}
	builder.Grow(r.Height * r.Width * 25)

	// Move cursor to home position
	builder.WriteString("\033[H")

	if r.UseColor {
		currentColor := ColorBlack
		for i := 0; i < r.Height; i++ {
			for j := 0; j < r.Width; j++ {
				char := r.Surface[i][j]
				color := r.ColorBuffer[i][j]

				if color != currentColor {
					builder.WriteString(color.ToANSI())
					currentColor = color
				}
				builder.WriteRune(char)
			}

			builder.WriteString("\033[K")
			if i < r.Height-1 {
				builder.WriteByte('\n')
			}
		}
		builder.WriteString(ColorReset())
	} else {
		for i := 0; i < r.Height; i++ {
			for j := 0; j < r.Width; j++ {
				builder.WriteRune(r.Surface[i][j])
			}
			builder.WriteString("\033[K")
			if i < r.Height-1 {
				builder.WriteByte('\n')
			}
		}
	}

	r.Writer.WriteString(builder.String())
	r.Writer.Flush()

	if r.ShowDebugInfo && r.Camera != nil {
		r.showDebugLine()
	}
}

// RenderScene renders an entire scene
func (r *TerminalRenderer) RenderScene(scene *Scene) {
	r.BeginFrame()

	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)

		// Generate shadow maps
		if r.ShadowRenderer != nil {
			for _, light := range r.LightingSystem.Lights {
				if light.IsEnabled {
					r.ShadowRenderer.RenderShadowMap(light, scene)
				}
			}
		}
	}

	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNode(node, worldMatrix, scene.Camera)
	}

	r.EndFrame()
}

func (r *TerminalRenderer) RenderSceneWithSpatialCulling(scene *Scene) {
	// Implementation matches original but ensures BeginFrame is called
	r.RenderScene(scene)
}

// renderNode renders a single scene node
func (r *TerminalRenderer) renderNode(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Triangle:
		r.RenderTriangle(obj, worldMatrix, camera)
	case *Quad:
		r.renderQuad(obj, worldMatrix, camera)
	case *Mesh:
		r.RenderMesh(obj, worldMatrix, camera)
	case *InstancedMesh:
		r.RenderInstancedMesh(obj, worldMatrix, camera)
	case *Line:
		r.RenderLine(obj, worldMatrix, camera)
	case *Circle:
		r.renderCircle(obj, worldMatrix, camera)
	case *Point:
		r.RenderPoint(obj, worldMatrix, camera)
	}
}

// RenderTriangle renders a single triangle
func (r *TerminalRenderer) RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	p0 := worldMatrix.TransformPointAffine(tri.P0)
	p1 := worldMatrix.TransformPointAffine(tri.P1)
	p2 := worldMatrix.TransformPointAffine(tri.P2)

	// Transform normal if needed
	var transformedNormal *Point
	if tri.UseSetNormal && tri.Normal != nil {
		tn := worldMatrix.TransformDirection(*tri.Normal)
		transformedNormal = &tn
	}

	r.renderTriangleInternal(p0, p1, p2, tri, transformedNormal, camera)
}

// renderTriangleInternal handles the core rendering logic for pre-transformed vertices
func (r *TerminalRenderer) renderTriangleInternal(p0, p1, p2 Point, originalTri *Triangle, transformedNormal *Point, camera *Camera) {
	v0 := camera.TransformToViewSpace(p0)
	v1 := camera.TransformToViewSpace(p1)
	v2 := camera.TransformToViewSpace(p2)

	if v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near {
		return
	}

	// Use object pool to reduce allocations
	transformed := AcquireTriangle()
	defer ReleaseTriangle(transformed)

	transformed.P0 = p0
	transformed.P1 = p1
	transformed.P2 = p2
	transformed.char = originalTri.char
	transformed.Material = originalTri.Material
	transformed.UseSetNormal = originalTri.UseSetNormal
	if originalTri.HasUVs {
		transformed.SetUVs(originalTri.UV0, originalTri.UV1, originalTri.UV2)
	}

	if originalTri.UseSetNormal && transformedNormal != nil {
		transformed.Normal = transformedNormal
	}

	if originalTri.Material.IsWireframe() {
		r.renderTriangleWireframe(transformed, camera)
	} else {
		r.rasterizeTriangleWithLighting(transformed, camera)
	}
}

// RenderLine renders a single line
func (r *TerminalRenderer) RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera) {
	start := worldMatrix.TransformPointAffine(line.Start)
	end := worldMatrix.TransformPointAffine(line.End)

	transformedLine := &Line{Start: start, End: end}

	clipped, visible := ClipLineToNearPlane(transformedLine, camera)
	if !visible {
		return
	}

	r.renderLineProjected(clipped, camera, ColorWhite)
}

// RenderPoint renders a single point
func (r *TerminalRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera) {
	p := worldMatrix.TransformPointAffine(*point)

	x, y, z := camera.ProjectPoint(p, r.Height, r.Width)
	if x == -1 {
		return
	}

	if x >= r.ClipMinX && x < r.ClipMaxX && y >= r.ClipMinY && y < r.ClipMaxY {
		if z < r.ZBuffer[y][x] {
			r.Surface[y][x] = r.Charset[7]
			r.ZBuffer[y][x] = z
		}
	}
}

// RenderMesh renders a complete mesh
func (r *TerminalRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// Optimization: Pre-transform vertices once per mesh instead of per triangle
	// This reduces matrix multiplications by a factor of ~6 (depending on mesh topology)
	transformedVertices := make([]Point, len(mesh.Vertices))

	// Pre-calculate mesh position offsets
	offsetX, offsetY, offsetZ := mesh.Position.X, mesh.Position.Y, mesh.Position.Z
	hasOffset := offsetX != 0 || offsetY != 0 || offsetZ != 0

	for i, v := range mesh.Vertices {
		transformed := worldMatrix.TransformPointAffine(v)
		if hasOffset {
			transformed.X += offsetX
			transformed.Y += offsetY
			transformed.Z += offsetZ
		}
		transformedVertices[i] = transformed
	}

	// Reusable triangle struct for metadata passing to internal renderer
	// We only set metadata (Material, UVs) on this, not vertices
	tempTri := AcquireTriangle()
	defer ReleaseTriangle(tempTri)
	tempTri.char = 'o'
	tempTri.Material = mesh.Material
	// Note: Mesh struct doesn't have per-face normals easily accessible here without extra logic,
	// so we assume UseSetNormal is false unless we compute them.
	// RenderMesh original code didn't set Normal/UseSetNormal (it was nil/false by default).

	hasUVs := len(mesh.UVs) > 0

	// Render triangles from indexed geometry
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				// Use pre-transformed vertices
				p0 := transformedVertices[idx0]
				p1 := transformedVertices[idx1]
				p2 := transformedVertices[idx2]

				// Update UVs on the reuseable triangle if needed
				if hasUVs {
					if idx0 < len(mesh.UVs) && idx1 < len(mesh.UVs) && idx2 < len(mesh.UVs) {
						tempTri.SetUVs(mesh.UVs[idx0], mesh.UVs[idx1], mesh.UVs[idx2])
					}
				}
				
				// Call internal renderer directly, skipping redundant transforms and allocations
				r.renderTriangleInternal(p0, p1, p2, tempTri, nil, camera)
			}
		}
	}
}

// RenderInstancedMesh renders multiple instances of the same mesh
func (r *TerminalRenderer) RenderInstancedMesh(instMesh *InstancedMesh, worldMatrix Matrix4x4, camera *Camera) {
	if !instMesh.Enabled || instMesh.BaseMesh == nil {
		return
	}

	// Render each instance with its own transform
	for _, instance := range instMesh.Instances {
		// Combine world matrix with instance transform
		finalMatrix := worldMatrix.Multiply(instance.Transform)
		
		// Temporarily override material color if instance has custom color
		originalMat := instMesh.BaseMesh.Material
		if instance.Color.R != 0 || instance.Color.G != 0 || instance.Color.B != 0 {
			tempMat := NewMaterial()
			tempMat.DiffuseColor = instance.Color
			instMesh.BaseMesh.Material = &tempMat
		}
		
		// Render the mesh with instance transform
		r.RenderMesh(instMesh.BaseMesh, finalMatrix, camera)
		
		// Restore original material
		instMesh.BaseMesh.Material = originalMat
	}
}

// SetLightingSystem sets the lighting system
func (r *TerminalRenderer) SetLightingSystem(ls *LightingSystem) {
	r.LightingSystem = ls
}

// SetCamera sets the camera
func (r *TerminalRenderer) SetCamera(camera *Camera) {
	r.Camera = camera
}

// GetDimensions returns renderer dimensions
func (r *TerminalRenderer) GetDimensions() (width, height int) {
	return r.Width, r.Height
}

// SetUseColor enables/disables color rendering
func (r *TerminalRenderer) SetUseColor(useColor bool) {
	r.UseColor = useColor
}

// SetShowDebugInfo enables/disables debug info
func (r *TerminalRenderer) SetShowDebugInfo(show bool) {
	r.ShowDebugInfo = show
}

// rasterizeTriangle performs triangle rasterization (Basic, no per-pixel lighting)
func (r *TerminalRenderer) rasterizeTriangle(t *Triangle, camera *Camera) {
	// Re-route to the lighting version for simplicity and correctness
	r.rasterizeTriangleWithLighting(t, camera)
}

// rasterizeTriangleWithLighting performs triangle rasterization with PERSPECTIVE CORRECT lighting
func (r *TerminalRenderer) rasterizeTriangleWithLighting(t *Triangle, camera *Camera) {
	clipped := ClipTriangleToNearPlane(t, camera)
	if len(clipped) == 0 {
		return
	}

	normal := CalculateSurfaceNormal(&t.P0, &t.P1, &t.P2, t.Normal, t.UseSetNormal)
	surfacePoint := Point{
		X: (t.P0.X + t.P1.X + t.P2.X) / 3.0,
		Y: (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0,
		Z: (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0,
	}

	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := dotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)
	if facing < 0 {
		return
	}

	for _, tri := range clipped {
		r.fillTriangleWithPerPixelLighting(tri, camera, normal, t.Material)
	}
}

// fillTriangleWithPerPixelLighting fills a triangle using perspective-correct interpolation
func (r *TerminalRenderer) fillTriangleWithPerPixelLighting(
	t *Triangle,
	camera *Camera,
	normal Point,
	material IMaterial,
) {
	// 1. Project vertices to screen space
	x0, y0, zDepth0 := camera.ProjectPoint(t.P0, r.Height, r.Width)
	x1, y1, zDepth1 := camera.ProjectPoint(t.P1, r.Height, r.Width)
	x2, y2, zDepth2 := camera.ProjectPoint(t.P2, r.Height, r.Width)

	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	// 2. Prepare for Perspective Correct Interpolation
	// We need 1/z (inverse depth) to interpolate linearly in screen space.
	// We use the view-space Z (zDepth) which ProjectPoint returns.
	if zDepth0 <= 0 {
		zDepth0 = 0.001
	}
	if zDepth1 <= 0 {
		zDepth1 = 0.001
	}
	if zDepth2 <= 0 {
		zDepth2 = 0.001
	}

	invZ0 := 1.0 / zDepth0
	invZ1 := 1.0 / zDepth1
	invZ2 := 1.0 / zDepth2

	// Pre-multiply attributes by 1/z
	p0OverZ := Point{X: t.P0.X * invZ0, Y: t.P0.Y * invZ0, Z: t.P0.Z * invZ0}
	p1OverZ := Point{X: t.P1.X * invZ1, Y: t.P1.Y * invZ1, Z: t.P1.Z * invZ1}
	p2OverZ := Point{X: t.P2.X * invZ2, Y: t.P2.Y * invZ2, Z: t.P2.Z * invZ2}

	var uv0OverZ, uv1OverZ, uv2OverZ TextureCoord
	hasUVs := t.HasUVs
	if hasUVs {
		uv0OverZ = TextureCoord{U: t.UV0.U * invZ0, V: t.UV0.V * invZ0}
		uv1OverZ = TextureCoord{U: t.UV1.U * invZ1, V: t.UV1.V * invZ1}
		uv2OverZ = TextureCoord{U: t.UV2.U * invZ2, V: t.UV2.V * invZ2}
	}

	// 3. Sort vertices by Y (Standard Scanline approach)
	if y1 < y0 {
		x0, y0, invZ0, p0OverZ, x1, y1, invZ1, p1OverZ = x1, y1, invZ1, p1OverZ, x0, y0, invZ0, p0OverZ
		if hasUVs {
			uv0OverZ, uv1OverZ = uv1OverZ, uv0OverZ
		}
	}
	if y2 < y0 {
		x0, y0, invZ0, p0OverZ, x2, y2, invZ2, p2OverZ = x2, y2, invZ2, p2OverZ, x0, y0, invZ0, p0OverZ
		if hasUVs {
			uv0OverZ, uv2OverZ = uv2OverZ, uv0OverZ
		}
	}
	if y2 < y1 {
		x1, y1, invZ1, p1OverZ, x2, y2, invZ2, p2OverZ = x2, y2, invZ2, p2OverZ, x1, y1, invZ1, p1OverZ
		if hasUVs {
			uv1OverZ, uv2OverZ = uv2OverZ, uv1OverZ
		}
	}

	totalHeight := y2 - y0
	if totalHeight == 0 {
		return
	}

	// 4. Rasterize scanlines
	// Respect clipping bounds for Y
	startY := y0
	if startY < r.ClipMinY {
		startY = r.ClipMinY
	}
	endY := y2
	if endY >= r.ClipMaxY {
		endY = r.ClipMaxY - 1
	}

	for y := startY; y <= endY; y++ {
		secondHalf := y > y1 || y1 == y0
		alpha := float64(y-y0) / float64(totalHeight)

		// Calculate boundary points for this scanline
		// A is the long edge (P0 -> P2)
		// B is the short edges (P0 -> P1 then P1 -> P2)

		// Interpolate Long Edge (A)
		ax := int(float64(x0) + alpha*float64(x2-x0) + 0.5)
		invZA := invZ0 + alpha*(invZ2-invZ0)
		pOverZA := lerpPoint3D(p0OverZ, p2OverZ, alpha)

		var uvOverZA TextureCoord
		if hasUVs {
			uvOverZA = lerpTextureCoord(uv0OverZ, uv2OverZ, alpha)
		}

		// Interpolate Short Edge (B)
		var bx int
		var invZB float64
		var pOverZB Point
		var uvOverZB TextureCoord

		beta := 0.0
		if secondHalf {
			if y2 != y1 {
				beta = float64(y-y1) / float64(y2-y1)
				bx = int(float64(x1) + beta*float64(x2-x1) + 0.5)
				invZB = invZ1 + beta*(invZ2-invZ1)
				pOverZB = lerpPoint3D(p1OverZ, p2OverZ, beta)
				if hasUVs {
					uvOverZB = lerpTextureCoord(uv1OverZ, uv2OverZ, beta)
				}
			}
		} else {
			if y1 != y0 {
				beta = float64(y-y0) / float64(y1-y0)
				bx = int(float64(x0) + beta*float64(x1-x0) + 0.5)
				invZB = invZ0 + beta*(invZ1-invZ0)
				pOverZB = lerpPoint3D(p0OverZ, p1OverZ, beta)
				if hasUVs {
					uvOverZB = lerpTextureCoord(uv0OverZ, uv1OverZ, beta)
				}
			}
		}

		// Ensure A is left, B is right
		if ax > bx {
			ax, bx = bx, ax
			invZA, invZB = invZB, invZA
			pOverZA, pOverZB = pOverZB, pOverZA
			if hasUVs {
				uvOverZA, uvOverZB = uvOverZB, uvOverZA
			}
		}

		// Respect clipping bounds for X
		startX := ax
		if startX < r.ClipMinX {
			startX = r.ClipMinX
		}
		endX := bx
		if endX >= r.ClipMaxX {
			endX = r.ClipMaxX - 1
		}

		width := bx - ax

		for x := startX; x <= endX; x++ {
			t := 0.0
			if width != 0 {
				t = float64(x-ax) / float64(width)
			}

			// Perspective Correct Interpolation for Pixel
			currentInvZ := invZA + t*(invZB-invZA)

			// Depth Buffer Test (using 1/invZ = Z)
			// Note: We use 1/invZ for depth check.
			// Smaller Z is closer.
			z := 1.0 / currentInvZ

			if z > 0 && z < r.ZBuffer[y][x] {
				currentPOverZ := lerpPoint3D(pOverZA, pOverZB, t)

				// Recover World Position: (P/Z) / (1/Z) = P
				pixelWorldPos := Point{
					X: currentPOverZ.X / currentInvZ,
					Y: currentPOverZ.Y / currentInvZ,
					Z: currentPOverZ.Z / currentInvZ,
				}

				// Recover UV
				var u, v float64
				if hasUVs {
					currentUVOverZ := lerpTextureCoord(uvOverZA, uvOverZB, t)
					u = currentUVOverZ.U / currentInvZ
					v = currentUVOverZ.V / currentInvZ
				}

				var pixelColor Color
				if r.LightingSystem != nil {
					if pbrMat, ok := material.(*PBRMaterial); ok {
						shadowCb := func(l *Light, p Point) float64 {
							if r.ShadowRenderer != nil {
								if sm := r.ShadowRenderer.ShadowMaps[l]; sm != nil {
									return sm.CalculateShadow(p)
								}
							}
							return 1.0
						}

						viewDirX, viewDirY, viewDirZ := camera.GetViewDirection(pixelWorldPos)
						viewDir := Point{X: viewDirX, Y: viewDirY, Z: viewDirZ}

						pixelColor = CalculatePBRLightingWithUV(pixelWorldPos, normal, viewDir, pbrMat, r.LightingSystem.Lights, r.LightingSystem.AmbientLight, r.LightingSystem.AmbientIntensity, u, v, shadowCb)
					} else {
						// Standard Lighting with Shadows & Textures
						shadowFactor := 1.0
						if r.ShadowRenderer != nil && len(r.LightingSystem.Lights) > 0 {
							for _, l := range r.LightingSystem.Lights {
								if l.IsEnabled {
									if sm := r.ShadowRenderer.ShadowMaps[l]; sm != nil {
										shadowFactor = sm.CalculateShadow(pixelWorldPos)
										break
									}
								}
							}
						}

						ao := CalculateSimpleAO(normal)

						if texMat, ok := material.(*TexturedMaterial); ok && hasUVs && texMat.UseTextures {
							litColor := r.LightingSystem.CalculateLighting(pixelWorldPos, normal, material, ao)
							texColor := texMat.SampleDiffuse(u, v)
							pixelColor = Color{
								R: uint8(float64(litColor.R) * float64(texColor.R) / 255.0),
								G: uint8(float64(litColor.G) * float64(texColor.G) / 255.0),
								B: uint8(float64(litColor.B) * float64(texColor.B) / 255.0),
							}
						} else {
							pixelColor = r.LightingSystem.CalculateLighting(pixelWorldPos, normal, material, ao)
						}

						if shadowFactor < 1.0 {
							pixelColor = Color{
								R: uint8(float64(pixelColor.R) * (0.2 + 0.8*shadowFactor)),
								G: uint8(float64(pixelColor.G) * (0.2 + 0.8*shadowFactor)),
								B: uint8(float64(pixelColor.B) * (0.2 + 0.8*shadowFactor)),
							}
						}
					}
				} else {
					pixelColor = r.simpleLighting(normal, material)
				}

				if r.UseColor {
					r.Surface[y][x] = FILLED_CHAR
					r.ColorBuffer[y][x] = pixelColor
				} else {
					// Fallback char
					brightness := (float64(pixelColor.R) + float64(pixelColor.G) + float64(pixelColor.B)) / (3.0 * 255.0)
					idx := int(brightness * float64(len(SHADING_RAMP)-1))
					if idx < 0 {
						idx = 0
					}
					if idx >= len(SHADING_RAMP) {
						idx = len(SHADING_RAMP) - 1
					}
					r.Surface[y][x] = rune(SHADING_RAMP[idx])
				}
				r.ZBuffer[y][x] = z
			}
		}
	}
}

// fillTriangle performs basic scanline rasterization (Kept for fallback, uses new clipping)
func (r *TerminalRenderer) fillTriangle(t *Triangle, camera *Camera, color Color, fillChar rune) {
	// Redirect to the robust lighting function to ensure clipping/perspective consistency
	// This ensures we don't duplicate the complex clipping logic.
	// If specific "solid color" behavior is needed, the material can be adjusted.
	r.fillTriangleWithPerPixelLighting(t, camera, Point{0, 1, 0}, t.Material)
}

// renderTriangleWireframe renders triangle edges
func (r *TerminalRenderer) renderTriangleWireframe(t *Triangle, camera *Camera) {
	line1 := NewLine(t.P0, t.P1)
	line2 := NewLine(t.P1, t.P2)
	line3 := NewLine(t.P2, t.P0)

	clipped1, visible1 := ClipLineToNearPlane(line1, camera)
	if visible1 {
		r.renderLineProjected(clipped1, camera, t.Material.GetWireframeColor())
	}

	clipped2, visible2 := ClipLineToNearPlane(line2, camera)
	if visible2 {
		r.renderLineProjected(clipped2, camera, t.Material.GetWireframeColor())
	}

	clipped3, visible3 := ClipLineToNearPlane(line3, camera)
	if visible3 {
		r.renderLineProjected(clipped3, camera, t.Material.GetWireframeColor())
	}
}

// renderLineProjected projects and renders a line with clipping
func (r *TerminalRenderer) renderLineProjected(line *Line, camera *Camera, color Color) {
	sx0, sy0, z0 := camera.ProjectPoint(line.Start, r.Height, r.Width)
	sx1, sy1, z1 := camera.ProjectPoint(line.End, r.Height, r.Width)

	if sx0 == -1 || sx1 == -1 {
		return
	}
	r.drawLineWithZ(sx0, sy0, sx1, sy1, z0, z1, color)
}

// drawLineWithZ draws a line with z-buffering and clipping
func (r *TerminalRenderer) drawLineWithZ(x0, y0, x1, y1 int, z0, z1 float64, color Color) {
	dx := x1 - x0
	dy := y1 - y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		return
	}

	xStep := float64(dx) / float64(steps)
	yStep := float64(dy) / float64(steps)
	zStep := (z1 - z0) / float64(steps)

	x := float64(x0)
	y := float64(y0)
	z := z0

	for i := 0; i <= steps; i++ {
		xi := int(x + 0.5)
		yi := int(y + 0.5)

		// Check clipping bounds
		if xi >= r.ClipMinX && xi < r.ClipMaxX && yi >= r.ClipMinY && yi < r.ClipMaxY {
			if z < r.ZBuffer[yi][xi] {
				if r.UseColor {
					r.Surface[yi][xi] = FILLED_CHAR
					r.ColorBuffer[yi][xi] = color
				} else {
					// ASCII line drawing logic
					char := '*'
					if abs(dx) > abs(dy)*2 {
						char = '-'
					} else if abs(dy) > abs(dx)*2 {
						char = '|'
					} else if (dx > 0 && dy > 0) || (dx < 0 && dy < 0) {
						char = '\\'
					} else {
						char = '/'
					}
					r.Surface[yi][xi] = char
				}
				r.ZBuffer[yi][xi] = z
			}
		}
		x += xStep
		y += yStep
		z += zStep
	}
}

// renderQuad converts quad to triangles and renders
func (r *TerminalRenderer) renderQuad(quad *Quad, worldMatrix Matrix4x4, camera *Camera) {
	p0 := worldMatrix.TransformPointAffine(quad.P0)
	p1 := worldMatrix.TransformPointAffine(quad.P1)
	p2 := worldMatrix.TransformPointAffine(quad.P2)
	p3 := worldMatrix.TransformPointAffine(quad.P3)

	transformed := &Quad{P0: p0, P1: p1, P2: p2, P3: p3, Material: quad.Material, UseSetNormal: quad.UseSetNormal}

	if quad.UseSetNormal && quad.Normal != nil {
		transformedNormal := worldMatrix.TransformDirection(*quad.Normal)
		transformed.Normal = &transformedNormal
	}

	triangles := ConvertQuadToTriangles(transformed)
	for _, tri := range triangles {
		if r.isTriangleVisible(tri, camera) {
			if tri.Material.IsWireframe() {
				r.renderTriangleWireframe(tri, camera)
			} else {
				r.rasterizeTriangleWithLighting(tri, camera)
			}
		}
	}
}

// renderCircle renders a circle as connected line segments
func (r *TerminalRenderer) renderCircle(circle *Circle, worldMatrix Matrix4x4, camera *Camera) {
	if len(circle.Points) == 0 {
		return
	}
	transformedPoints := make([]Point, len(circle.Points))
	for i, p := range circle.Points {
		transformedPoints[i] = worldMatrix.TransformPointAffine(p)
	}
	for i := 0; i < len(transformedPoints); i++ {
		p1 := transformedPoints[i]
		p2 := transformedPoints[(i+1)%len(transformedPoints)]
		line := NewLine(p1, p2)
		clipped, visible := ClipLineToNearPlane(line, camera)
		if visible {
			r.renderLineProjected(clipped, camera, ColorWhite)
		}
	}
}

func (r *TerminalRenderer) isTriangleVisible(t *Triangle, camera *Camera) bool {
	v0 := camera.TransformToViewSpace(t.P0)
	v1 := camera.TransformToViewSpace(t.P1)
	v2 := camera.TransformToViewSpace(t.P2)
	return !(v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near)
}

func (r *TerminalRenderer) simpleLighting(normal Point, material IMaterial) Color {
	ao := CalculateSimpleAO(normal)
	lx, ly, lz := -1.0, 1.0, -1.0
	lx, ly, lz = normalizeVector(lx, ly, lz)
	intensity := dotProduct(normal.X, normal.Y, normal.Z, lx, ly, lz)
	if intensity < 0 {
		intensity = 0
	}
	intensity *= ao

	diffuseColor := material.GetDiffuseColor(0, 0)
	return Color{
		R: uint8(clamp(float64(diffuseColor.R)*intensity, 0, 255)),
		G: uint8(clamp(float64(diffuseColor.G)*intensity, 0, 255)),
		B: uint8(clamp(float64(diffuseColor.B)*intensity, 0, 255)),
	}
}

func (r *TerminalRenderer) showDebugLine() {
	if r.Camera == nil {
		return
	}
	pos := r.Camera.GetPosition()
	pitch, yaw, roll := r.Camera.GetRotation()
	r.debugBuffer.Reset()
	r.debugBuffer.WriteString(fmt.Sprintf("FPS: %.1f", 60.0))
	camInfo := fmt.Sprintf("Pos:(%.1f,%.1f,%.1f) Rot:(P:%.2f Y:%.2f R:%.2f)", pos.X, pos.Y, pos.Z, pitch*180/3.14159, yaw*180/3.14159, roll*180/3.14159)
	totalLen := r.debugBuffer.Len() + len(camInfo)
	padding := r.Width - totalLen
	if padding < 1 {
		padding = 1
	}
	for i := 0; i < padding; i++ {
		r.debugBuffer.WriteByte(' ')
	}
	r.debugBuffer.WriteString(camInfo)
	debugLine := r.debugBuffer.String()
	if debugLine != r.lastDebugLine {
		fmt.Fprintf(r.Writer, "\033[%d;1H", r.Height+1)
		fmt.Fprintf(r.Writer, "\033[K%s", debugLine)
		r.Writer.Flush()
		r.lastDebugLine = debugLine
	}
}

func lerpPoint3D(a, b Point, t float64) Point {
	return Point{
		X: a.X + t*(b.X-a.X),
		Y: a.Y + t*(b.Y-a.Y),
		Z: a.Z + t*(b.Z-a.Z),
	}
}

func lerpTextureCoord(a, b TextureCoord, t float64) TextureCoord {
	return TextureCoord{
		U: a.U + t*(b.U-a.U),
		V: a.V + t*(b.V-a.V),
	}
}
