package main

import (
	"bufio"
	"fmt"
	"math"
	"strings"
	"time"
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
	Camera         *Camera
	ShowDebugInfo  bool
	debugBuffer    strings.Builder
	lastDebugLine  string
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
		ShowDebugInfo: true,
	}
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
	}

	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNode(node, worldMatrix, scene.Camera)
	}

	r.EndFrame()
}

func (r *TerminalRenderer) RenderSceneWithSpatialCulling(scene *Scene) {
	r.BeginFrame()

	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
	}

	// Build or update BVH
	bvh := scene.BuildBVH()
	if bvh == nil {
		// Fallback to all nodes
		nodes := scene.GetRenderableNodes()
		for _, node := range nodes {
			worldMatrix := node.Transform.GetWorldMatrix()
			r.renderNode(node, worldMatrix, scene.Camera)
		}
		return
	}

	// Query BVH with camera frustum
	frustum := BuildFrustum(scene.Camera)

	// Convert frustum to AABB query volume (conservative)
	camPos := scene.Camera.GetPosition()
	querySize := scene.Camera.Far
	queryBounds := NewAABB(
		Point{
			X: camPos.X - querySize,
			Y: camPos.Y - querySize,
			Z: camPos.Z - querySize,
		},
		Point{
			X: camPos.X + querySize,
			Y: camPos.Y + querySize,
			Z: camPos.Z + querySize,
		},
	)

	// Get potentially visible objects from BVH
	candidates := bvh.Query(queryBounds)

	// Test candidates against frustum
	for _, node := range candidates {
		bounds := ComputeNodeBounds(node)
		if bounds != nil && frustum.TestAABB(bounds) {
			worldMatrix := node.Transform.GetWorldMatrix()
			r.renderNode(node, worldMatrix, scene.Camera)
		}
	}

	r.EndFrame()
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
	// Transform vertices
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Check visibility
	v0 := camera.TransformToViewSpace(p0)
	v1 := camera.TransformToViewSpace(p1)
	v2 := camera.TransformToViewSpace(p2)

	if v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near {
		return
	}

	// Create transformed triangle
	transformed := &Triangle{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		char:         tri.char,
		Material:     tri.Material,
		UseSetNormal: tri.UseSetNormal,
	}

	if tri.UseSetNormal && tri.Normal != nil {
		transformedNormal := worldMatrix.TransformDirection(*tri.Normal)
		transformed.Normal = &transformedNormal
	}

	if tri.Material.Wireframe {
		r.renderTriangleWireframe(transformed, camera)
	} else {
		r.rasterizeTriangle(transformed, camera)
	}
}

// RenderLine renders a single line
func (r *TerminalRenderer) RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera) {
	start := worldMatrix.TransformPoint(line.Start)
	end := worldMatrix.TransformPoint(line.End)

	transformedLine := &Line{Start: start, End: end}

	clipped, visible := ClipLineToNearPlane(transformedLine, camera)
	if !visible {
		return
	}

	r.renderLineProjected(clipped, camera, ColorWhite)
}

// RenderPoint renders a single point
func (r *TerminalRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera) {
	p := worldMatrix.TransformPoint(*point)

	x, y, z := camera.ProjectPoint(p, r.Height, r.Width)
	if x == -1 {
		return
	}

	if x >= 0 && x < r.Width && y >= 0 && y < r.Height {
		if z < r.ZBuffer[y][x] {
			r.Surface[y][x] = r.Charset[7]
			r.ZBuffer[y][x] = z
		}
	}
}

// RenderMesh renders a complete mesh
func (r *TerminalRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// Transform mesh position
	meshPos := worldMatrix.TransformPoint(mesh.Position)

	// Render quads
	for _, quad := range mesh.Quads {
		offsetQuad := &Quad{
			P0:           Point{X: quad.P0.X + meshPos.X, Y: quad.P0.Y + meshPos.Y, Z: quad.P0.Z + meshPos.Z},
			P1:           Point{X: quad.P1.X + meshPos.X, Y: quad.P1.Y + meshPos.Y, Z: quad.P1.Z + meshPos.Z},
			P2:           Point{X: quad.P2.X + meshPos.X, Y: quad.P2.Y + meshPos.Y, Z: quad.P2.Z + meshPos.Z},
			P3:           Point{X: quad.P3.X + meshPos.X, Y: quad.P3.Y + meshPos.Y, Z: quad.P3.Z + meshPos.Z},
			Material:     quad.Material,
			UseSetNormal: quad.UseSetNormal,
			Normal:       quad.Normal,
		}
		r.renderQuad(offsetQuad, IdentityMatrix(), camera)
	}

	// Render triangles
	for _, tri := range mesh.Triangles {
		offsetTri := &Triangle{
			P0:           Point{X: tri.P0.X + meshPos.X, Y: tri.P0.Y + meshPos.Y, Z: tri.P0.Z + meshPos.Z},
			P1:           Point{X: tri.P1.X + meshPos.X, Y: tri.P1.Y + meshPos.Y, Z: tri.P1.Z + meshPos.Z},
			P2:           Point{X: tri.P2.X + meshPos.X, Y: tri.P2.Y + meshPos.Y, Z: tri.P2.Z + meshPos.Z},
			char:         tri.char,
			Material:     tri.Material,
			UseSetNormal: tri.UseSetNormal,
			Normal:       tri.Normal,
		}

		if r.isTriangleVisible(offsetTri, camera) {
			if tri.Material.Wireframe {
				r.renderTriangleWireframe(offsetTri, camera)
			} else {
				r.rasterizeTriangleWithLighting(offsetTri, camera)
			}
		}
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

// rasterizeTriangle performs triangle rasterization with lighting
func (r *TerminalRenderer) rasterizeTriangle(t *Triangle, camera *Camera) {
	clipped := ClipTriangleToNearPlane(t, camera)
	if len(clipped) == 0 {
		return
	}

	// Calculate lighting once
	normal := CalculateSurfaceNormal(&t.P0, &t.P1, &t.P2, t.Normal, t.UseSetNormal)
	surfacePoint := Point{
		X: (t.P0.X + t.P1.X + t.P2.X) / 3.0,
		Y: (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0,
		Z: (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0,
	}

	// Backface culling
	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := dotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)
	if facing < 0 {
		return
	}

	// Calculate lighting
	var pixelColor Color
	var fillChar rune

	if r.LightingSystem != nil {
		ao := CalculateSimpleAO(normal)
		pixelColor = r.LightingSystem.CalculateLighting(surfacePoint, normal, t.Material, ao)
	} else {
		pixelColor = r.simpleLighting(normal, t.Material)
	}

	brightness := (float64(pixelColor.R) + float64(pixelColor.G) + float64(pixelColor.B)) / (3.0 * 255.0)
	index := int(brightness * float64(len(SHADING_RAMP)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(SHADING_RAMP) {
		index = len(SHADING_RAMP) - 1
	}
	fillChar = rune(SHADING_RAMP[index])

	// Rasterize clipped triangles
	for _, tri := range clipped {
		r.fillTriangle(tri, camera, pixelColor, fillChar)
	}
}

func (r *TerminalRenderer) rasterizeTriangleWithLighting(t *Triangle, camera *Camera) {
	clipped := ClipTriangleToNearPlane(t, camera)
	if len(clipped) == 0 {
		return
	}

	// Calculate surface normal once (for backface culling)
	normal := CalculateSurfaceNormal(&t.P0, &t.P1, &t.P2, t.Normal, t.UseSetNormal)
	surfacePoint := Point{
		X: (t.P0.X + t.P1.X + t.P2.X) / 3.0,
		Y: (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0,
		Z: (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0,
	}

	// Backface culling
	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := dotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)
	if facing < 0 {
		return
	}

	// Rasterize clipped triangles with PER-PIXEL lighting
	for _, tri := range clipped {
		r.fillTriangleWithPerPixelLighting(tri, camera, normal, t.Material)
	}
}

// fillTriangle performs scanline rasterization
func (r *TerminalRenderer) fillTriangle(t *Triangle, camera *Camera, color Color, fillChar rune) {
	x0, y0, z0 := camera.ProjectPoint(t.P0, r.Height, r.Width)
	x1, y1, z1 := camera.ProjectPoint(t.P1, r.Height, r.Width)
	x2, y2, z2 := camera.ProjectPoint(t.P2, r.Height, r.Width)

	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	if z0 <= 0 {
		z0 = 0.001
	}
	if z1 <= 0 {
		z1 = 0.001
	}
	if z2 <= 0 {
		z2 = 0.001
	}

	// Sort by Y
	if y1 < y0 {
		x0, y0, z0, x1, y1, z1 = x1, y1, z1, x0, y0, z0
	}
	if y2 < y0 {
		x0, y0, z0, x2, y2, z2 = x2, y2, z2, x0, y0, z0
	}
	if y2 < y1 {
		x1, y1, z1, x2, y2, z2 = x2, y2, z2, x1, y1, z1
	}

	totalHeight := y2 - y0
	if totalHeight == 0 {
		return
	}

	for y := y0; y <= y2; y++ {
		if y < 0 || y >= r.Height {
			continue
		}

		secondHalf := y > y1 || y1 == y0
		alpha := float64(y-y0) / float64(totalHeight)

		beta := 0.0
		if secondHalf {
			if y2 != y1 {
				beta = float64(y-y1) / float64(y2-y1)
			}
		} else {
			if y1 != y0 {
				beta = float64(y-y0) / float64(y1-y0)
			}
		}

		ax := int(float64(x0) + alpha*float64(x2-x0) + 0.5)
		az := z0 + alpha*(z2-z0)

		var bx int
		var bz float64
		if secondHalf {
			bx = int(float64(x1) + beta*float64(x2-x1) + 0.5)
			bz = z1 + beta*(z2-z1)
		} else {
			bx = int(float64(x0) + beta*float64(x1-x0) + 0.5)
			bz = z0 + beta*(z1-z0)
		}

		if ax > bx {
			ax, bx = bx, ax
			az, bz = bz, az
		}

		for x := ax; x <= bx; x++ {
			if x < 0 || x >= r.Width {
				continue
			}

			t := 0.0
			if bx != ax {
				t = float64(x-ax) / float64(bx-ax)
			}
			z := az + t*(bz-az)

			if z > 0 && z < r.ZBuffer[y][x] {
				if r.UseColor {
					r.Surface[y][x] = FILLED_CHAR
					r.ColorBuffer[y][x] = color
				} else {
					r.Surface[y][x] = fillChar
				}
				r.ZBuffer[y][x] = z
			}
		}
	}
}

// fillTriangleWithPerPixelLighting - Per-pixel lighting computation
func (r *TerminalRenderer) fillTriangleWithPerPixelLighting(
	t *Triangle,
	camera *Camera,
	normal Point,
	material Material,
) {
	x0, y0, z0 := camera.ProjectPoint(t.P0, r.Height, r.Width)
	x1, y1, z1 := camera.ProjectPoint(t.P1, r.Height, r.Width)
	x2, y2, z2 := camera.ProjectPoint(t.P2, r.Height, r.Width)

	if x0 == -1 || x1 == -1 || x2 == -1 {
		return
	}

	// Guard against zero/negative depths
	if z0 <= 0 {
		z0 = 0.001
	}
	if z1 <= 0 {
		z1 = 0.001
	}
	if z2 <= 0 {
		z2 = 0.001
	}

	// Sort vertices by Y
	if y1 < y0 {
		x0, y0, z0, x1, y1, z1 = x1, y1, z1, x0, y0, z0
	}
	if y2 < y0 {
		x0, y0, z0, x2, y2, z2 = x2, y2, z2, x0, y0, z0
	}
	if y2 < y1 {
		x1, y1, z1, x2, y2, z2 = x2, y2, z2, x1, y1, z1
	}

	totalHeight := y2 - y0
	if totalHeight == 0 {
		return
	}

	// Calculate world-space positions for lighting interpolation
	p0World := t.P0
	p1World := t.P1
	p2World := t.P2

	for y := y0; y <= y2; y++ {
		if y < 0 || y >= r.Height {
			continue
		}

		secondHalf := y > y1 || y1 == y0
		alpha := float64(y-y0) / float64(totalHeight)

		beta := 0.0
		if secondHalf {
			if y2 != y1 {
				beta = float64(y-y1) / float64(y2-y1)
			}
		} else {
			if y1 != y0 {
				beta = float64(y-y0) / float64(y1-y0)
			}
		}

		ax := int(float64(x0) + alpha*float64(x2-x0) + 0.5)
		az := z0 + alpha*(z2-z0)

		var bx int
		var bz float64
		if secondHalf {
			bx = int(float64(x1) + beta*float64(x2-x1) + 0.5)
			bz = z1 + beta*(z2-z1)
		} else {
			bx = int(float64(x0) + beta*float64(x1-x0) + 0.5)
			bz = z0 + beta*(z1-z0)
		}

		if ax > bx {
			ax, bx = bx, ax
			az, bz = bz, az
		}

		// Interpolate world positions for this scanline
		var aWorldPos, bWorldPos Point
		if secondHalf {
			aWorldPos = lerpPoint3D(p0World, p2World, alpha)
			bWorldPos = lerpPoint3D(p1World, p2World, beta)
		} else {
			aWorldPos = lerpPoint3D(p0World, p2World, alpha)
			bWorldPos = lerpPoint3D(p0World, p1World, beta)
		}

		// Rasterize scanline with per-pixel lighting
		for x := ax; x <= bx; x++ {
			if x < 0 || x >= r.Width {
				continue
			}

			t := 0.0
			if bx != ax {
				t = float64(x-ax) / float64(bx-ax)
			}
			z := az + t*(bz-az)

			if z > 0 && z < r.ZBuffer[y][x] {
				// Interpolate world position for this pixel
				pixelWorldPos := lerpPoint3D(aWorldPos, bWorldPos, t)

				// Calculate lighting for THIS pixel
				var pixelColor Color
				if r.LightingSystem != nil {
					ao := CalculateSimpleAO(normal)
					pixelColor = r.LightingSystem.CalculateLighting(
						pixelWorldPos,
						normal,
						material,
						ao,
					)
				} else {
					pixelColor = r.simpleLighting(normal, material)
				}

				// Calculate character based on brightness
				brightness := (float64(pixelColor.R) + float64(pixelColor.G) + float64(pixelColor.B)) / (3.0 * 255.0)
				index := int(brightness * float64(len(SHADING_RAMP)-1))
				if index < 0 {
					index = 0
				}
				if index >= len(SHADING_RAMP) {
					index = len(SHADING_RAMP) - 1
				}
				fillChar := rune(SHADING_RAMP[index])

				// Write pixel with correct lighting
				if r.UseColor {
					r.Surface[y][x] = FILLED_CHAR
					r.ColorBuffer[y][x] = pixelColor
				} else {
					r.Surface[y][x] = fillChar
				}
				r.ZBuffer[y][x] = z
			}
		}
	}
}

// renderTriangleWireframe renders triangle edges
func (r *TerminalRenderer) renderTriangleWireframe(t *Triangle, camera *Camera) {
	line1 := NewLine(t.P0, t.P1)
	line2 := NewLine(t.P1, t.P2)
	line3 := NewLine(t.P2, t.P0)

	clipped1, visible1 := ClipLineToNearPlane(line1, camera)
	if visible1 {
		r.renderLineProjected(clipped1, camera, t.Material.WireframeColor)
	}

	clipped2, visible2 := ClipLineToNearPlane(line2, camera)
	if visible2 {
		r.renderLineProjected(clipped2, camera, t.Material.WireframeColor)
	}

	clipped3, visible3 := ClipLineToNearPlane(line3, camera)
	if visible3 {
		r.renderLineProjected(clipped3, camera, t.Material.WireframeColor)
	}
}

// renderLineProjected projects and renders a line
func (r *TerminalRenderer) renderLineProjected(line *Line, camera *Camera, color Color) {
	sx0, sy0, z0 := camera.ProjectPoint(line.Start, r.Height, r.Width)
	sx1, sy1, z1 := camera.ProjectPoint(line.End, r.Height, r.Width)

	if sx0 == -1 || sx1 == -1 {
		return
	}

	r.drawLineWithZ(sx0, sy0, sx1, sy1, z0, z1, color)
}

// drawLineWithZ draws a line with z-buffering
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

		if xi >= 0 && xi < r.Width && yi >= 0 && yi < r.Height {
			if z < r.ZBuffer[yi][xi] {
				if r.UseColor {
					r.Surface[yi][xi] = FILLED_CHAR
					r.ColorBuffer[yi][xi] = color
				} else {
					if abs(dx) > abs(dy)*2 {
						r.Surface[yi][xi] = '-'
					} else if abs(dy) > abs(dx)*2 {
						r.Surface[yi][xi] = '|'
					} else if (dx > 0 && dy > 0) || (dx < 0 && dy < 0) {
						r.Surface[yi][xi] = '\\'
					} else {
						r.Surface[yi][xi] = '/'
					}
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
	p0 := worldMatrix.TransformPoint(quad.P0)
	p1 := worldMatrix.TransformPoint(quad.P1)
	p2 := worldMatrix.TransformPoint(quad.P2)
	p3 := worldMatrix.TransformPoint(quad.P3)

	transformed := &Quad{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		P3:           p3,
		Material:     quad.Material,
		UseSetNormal: quad.UseSetNormal,
	}

	if quad.UseSetNormal && quad.Normal != nil {
		transformedNormal := worldMatrix.TransformDirection(*quad.Normal)
		transformed.Normal = &transformedNormal
	}

	triangles := ConvertQuadToTriangles(transformed)
	for _, tri := range triangles {
		if r.isTriangleVisible(tri, camera) {
			if tri.Material.Wireframe {
				r.renderTriangleWireframe(tri, camera)
			} else {
				r.rasterizeTriangle(tri, camera)
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
		transformedPoints[i] = worldMatrix.TransformPoint(p)
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

// isTriangleVisible checks basic visibility
func (r *TerminalRenderer) isTriangleVisible(t *Triangle, camera *Camera) bool {
	v0 := camera.TransformToViewSpace(t.P0)
	v1 := camera.TransformToViewSpace(t.P1)
	v2 := camera.TransformToViewSpace(t.P2)

	return !(v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near)
}

// simpleLighting provides basic lighting when no lighting system is set
func (r *TerminalRenderer) simpleLighting(normal Point, material Material) Color {
	ao := CalculateSimpleAO(normal)
	lx, ly, lz := -1.0, 1.0, -1.0
	lx, ly, lz = normalizeVector(lx, ly, lz)
	intensity := dotProduct(normal.X, normal.Y, normal.Z, lx, ly, lz)
	if intensity < 0 {
		intensity = 0
	}
	intensity *= ao

	red := float64(material.DiffuseColor.R) * intensity
	green := float64(material.DiffuseColor.G) * intensity
	blue := float64(material.DiffuseColor.B) * intensity

	return Color{
		R: uint8(clamp(red, 0, 255)),
		G: uint8(clamp(green, 0, 255)),
		B: uint8(clamp(blue, 0, 255)),
	}
}

// showDebugLine displays debug info
func (r *TerminalRenderer) showDebugLine() {
	if r.Camera == nil {
		return
	}

	pos := r.Camera.GetPosition()
	pitch, yaw, roll := r.Camera.GetRotation()

	r.debugBuffer.Reset()
	r.debugBuffer.WriteString(fmt.Sprintf("FPS: %.1f", 60.0))

	camInfo := fmt.Sprintf("Pos:(%.1f,%.1f,%.1f) Rot:(P:%.2f Y:%.2f R:%.2f)",
		pos.X, pos.Y, pos.Z, pitch*180/3.14159, yaw*180/3.14159, roll*180/3.14159)

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

// RenderLoop starts the render loop (convenience method)
func (r *TerminalRenderer) RenderLoop(scene *Scene, fps float64, updateFunc func(*Scene, float64)) {
	dt := 1.0 / fps
	ticker := time.NewTicker(time.Duration(dt*1000) * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()
	currentFPS := fps

	for {
		<-ticker.C

		if updateFunc != nil {
			updateFunc(scene, dt)
		}

		scene.Update(dt)
		r.RenderScene(scene)
		r.Present()

		frameCount++
		if frameCount%10 == 0 {
			elapsed := time.Since(startTime).Seconds()
			currentFPS = float64(frameCount) / elapsed
		}

		_ = currentFPS // Use for debug display if needed
	}
}

func (r *TerminalRenderer) RenderLODGroupWithTransition(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	lodGroup, ok := node.Object.(*LODGroupWithTransitions)
	if !ok {
		return
	}

	if !lodGroup.TransitionState.IsTransitioning {
		// Not transitioning - render current LOD
		mesh := lodGroup.GetCurrentMesh()
		if mesh != nil {
			r.RenderMesh(mesh, worldMatrix, camera)
		}
		return
	}

	// Transitioning - blend two LODs
	fromMesh, toMesh, alpha := lodGroup.GetBlendedMesh()

	if fromMesh != nil && toMesh != nil {
		// Render both meshes with alpha blending
		// First, render lower LOD (will be occluded by higher LOD where appropriate)
		r.RenderMeshWithAlpha(fromMesh, worldMatrix, camera, 1.0-alpha)
		r.RenderMeshWithAlpha(toMesh, worldMatrix, camera, alpha)
	}
}

func (r *TerminalRenderer) RenderMeshWithAlpha(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera, alpha float64) {
	// Store original material colors
	originalColors := make([]Color, len(mesh.Triangles))
	for i, tri := range mesh.Triangles {
		originalColors[i] = tri.Material.DiffuseColor

		// Blend material color with alpha
		tri.Material.DiffuseColor = Color{
			R: uint8(float64(tri.Material.DiffuseColor.R) * alpha),
			G: uint8(float64(tri.Material.DiffuseColor.G) * alpha),
			B: uint8(float64(tri.Material.DiffuseColor.B) * alpha),
		}
	}

	// Render with modified colors
	r.RenderMesh(mesh, worldMatrix, camera)

	// Restore original colors
	for i, tri := range mesh.Triangles {
		tri.Material.DiffuseColor = originalColors[i]
	}
}

// RasterizeScanline fills a horizontal line with perspective-correct depth interpolation
// Uses 1/z for proper perspective interpolation
func (r *TerminalRenderer) RasterizeScanline(y, xStart, xEnd int, zStart, zEnd float64, pixelColor Color, fillChar rune) {

	// Bounds check for y
	if y < 0 || y >= r.Height {
		return
	}

	// Early rejection if completely off-screen
	if (xStart < 0 && xEnd < 0) || (xStart >= r.Width && xEnd >= r.Width) {
		return
	}

	originalXStart := xStart
	originalXEnd := xEnd

	// Guard against zero/negative depths
	if zStart <= 0 {
		zStart = 0.001
	}
	if zEnd <= 0 {
		zEnd = 0.001
	}

	// Perspective-correct interpolation: use 1/z
	invZStart := 1.0 / zStart
	invZEnd := 1.0 / zEnd

	// Clamp to screen bounds
	if xStart < 0 {
		xStart = 0
	}
	if xEnd >= r.Width {
		xEnd = r.Width - 1
	}

	// Handle degenerate case
	if xStart > xEnd {
		return
	}

	// Draw pixels including the end pixel (<=) to avoid gaps
	for x := xStart; x <= xEnd; x++ {
		// Calculate interpolation parameter accounting for clamping
		t := 0.0
		if originalXEnd != originalXStart {
			t = float64(x-originalXStart) / float64(originalXEnd-originalXStart)
		}

		// Perspective-correct depth interpolation
		invZ := invZStart + t*(invZEnd-invZStart)
		z := 1.0 / invZ

		// Additional safety check
		if z <= 0 || math.IsNaN(z) || math.IsInf(z, 0) {
			continue
		}

		// Get render context for buffer access

		// Z-buffer test
		if z < r.ZBuffer[y][x] {
			if r.UseColor {
				r.Surface[y][x] = FILLED_CHAR
				r.ColorBuffer[y][x] = pixelColor
			} else {
				r.Surface[y][x] = fillChar
			}
			r.ZBuffer[y][x] = z
		}
	}
}

// lerpPoint3D interpolates between two 3D points
func lerpPoint3D(a, b Point, t float64) Point {
	return Point{
		X: a.X + t*(b.X-a.X),
		Y: a.Y + t*(b.Y-a.Y),
		Z: a.Z + t*(b.Z-a.Z),
	}
}
