package main

import (
	"bufio"
	"fmt"
	"math"
	"strings"
	"time"
)

// Renderer handles all rendering operations
type Renderer struct {
	Writer         *bufio.Writer
	Height         int
	Width          int
	Surface        [][]rune
	ColorBuffer    [][]Color
	ZBuffer        [][]float64
	Charset        [9]rune
	UseColor       bool
	LightingSystem *LightingSystem
	ShowDebugInfo  bool
	debugBuffer    strings.Builder
	lastDebugLine  string
}

// NewRenderer creates a new renderer
func NewRenderer(writer *bufio.Writer, height, width int) *Renderer {
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

	return &Renderer{
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

// SetUseColor enables or disables color rendering
func (r *Renderer) SetUseColor(useColor bool) {
	r.UseColor = useColor
}

// SetLightingSystem sets the lighting system for this renderer
func (r *Renderer) SetLightingSystem(ls *LightingSystem) {
	r.LightingSystem = ls
}

// ClearBuffers clears all rendering buffers
func (r *Renderer) ClearBuffers() {
	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			r.Surface[y][x] = ' '
			r.ColorBuffer[y][x] = ColorBlack
			r.ZBuffer[y][x] = math.Inf(1)
		}
	}
}

// RenderScene renders a complete scene
func (r *Renderer) RenderScene(scene *Scene) {
	r.ClearBuffers()

	// Update lighting system camera reference
	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
	}

	// Get all renderable nodes
	nodes := scene.GetRenderableNodes()

	// Render each node - transform on-the-fly
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.RenderNodeWithMatrix(node, worldMatrix, scene.Camera)
	}
}

// RenderMesh renders a mesh with per-primitive culling
func (r *Renderer) RenderMesh(mesh *Mesh, camera *Camera) {
	// Render each quad by converting to triangles
	for _, quad := range mesh.Quads {
		triangles := ConvertQuadToTriangles(quad)
		for _, tri := range triangles {
			if r.IsTriangleVisible(tri, camera) {
				if tri.Material.Wireframe {
					r.RenderTriangleWireframe(tri, camera)
				} else {
					tri.DrawFilled(r, camera)
				}
			}
		}
	}

	// Render each triangle
	for _, tri := range mesh.Triangles {
		if r.IsTriangleVisible(tri, camera) {
			if tri.Material.Wireframe {
				r.RenderTriangleWireframe(tri, camera)
			} else {
				tri.DrawFilled(r, camera)
			}
		}
	}
}

// RenderTriangle renders a triangle (solid or wireframe)
func (r *Renderer) RenderTriangle(t *Triangle, camera *Camera) {
	if t.Material.Wireframe {
		r.RenderTriangleWireframe(t, camera)
	} else {
		t.DrawFilled(r, camera)
	}
}

// IsTriangleVisible checks if triangle is potentially visible
func (r *Renderer) IsTriangleVisible(t *Triangle, camera *Camera) bool {
	v0 := camera.TransformToViewSpace(t.P0)
	v1 := camera.TransformToViewSpace(t.P1)
	v2 := camera.TransformToViewSpace(t.P2)

	// If all vertices behind near plane, cull
	if v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near {
		return false
	}

	return true
}

// RenderTriangleWireframe renders triangle as wireframe
func (r *Renderer) RenderTriangleWireframe(t *Triangle, camera *Camera) {
	line1 := NewLine(t.P0, t.P1)
	line2 := NewLine(t.P1, t.P2)
	line3 := NewLine(t.P2, t.P0)

	clipped1, visible1 := ClipLineToNearPlane(line1, camera)
	if visible1 {
		r.RenderLineWithColor(clipped1, camera, t.Material.WireframeColor)
	}

	clipped2, visible2 := ClipLineToNearPlane(line2, camera)
	if visible2 {
		r.RenderLineWithColor(clipped2, camera, t.Material.WireframeColor)
	}

	clipped3, visible3 := ClipLineToNearPlane(line3, camera)
	if visible3 {
		r.RenderLineWithColor(clipped3, camera, t.Material.WireframeColor)
	}
}

// RenderLineWithColor renders a line with specific color
func (r *Renderer) RenderLineWithColor(line *Line, camera *Camera, color Color) {
	sx0, sy0, zStart := camera.ProjectPoint(line.Start, r.Height, r.Width)
	sx1, sy1, zEnd := camera.ProjectPoint(line.End, r.Height, r.Width)

	if sx0 == -1 || sx1 == -1 {
		return
	}

	r.drawLineWithColorAndZ(sx0, sy0, sx1, sy1, zStart, zEnd, color)
}

// RenderNodeWithMatrix renders a node with a given world matrix (NO COPYING!)
func (r *Renderer) RenderNodeWithMatrix(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Triangle:
		r.RenderTriangleWithMatrix(obj, worldMatrix, camera)

	case *Quad:
		r.RenderQuadWithMatrix(obj, worldMatrix, camera)

	case *Mesh:
		r.RenderMeshWithMatrix(obj, worldMatrix, camera)

	case *Line:
		r.RenderLineWithMatrix(obj, worldMatrix, camera)

	case *Circle:
		r.RenderCircleWithMatrix(obj, worldMatrix, camera)
	}
}

// RenderTriangleWithMatrix renders a triangle with world matrix (transforms on-the-fly)
func (r *Renderer) RenderTriangleWithMatrix(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices on-the-fly
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Check visibility in world space
	v0 := camera.TransformToViewSpace(p0)
	v1 := camera.TransformToViewSpace(p1)
	v2 := camera.TransformToViewSpace(p2)

	if v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near {
		return // All behind near plane
	}

	// Create temporary triangle for rendering (only this one allocation)
	transformed := &Triangle{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		char:         tri.char,
		Material:     tri.Material,
		UseSetNormal: tri.UseSetNormal,
	}

	// Transform normal if set
	if tri.UseSetNormal && tri.Normal != nil {
		transformedNormal := worldMatrix.TransformDirection(*tri.Normal)
		transformed.Normal = &transformedNormal
	}

	// Render
	if tri.Material.Wireframe {
		r.RenderTriangleWireframe(transformed, camera)
	} else {
		transformed.DrawFilled(r, camera)
	}
}

// RenderQuadWithMatrix renders a quad with world matrix
func (r *Renderer) RenderQuadWithMatrix(quad *Quad, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices on-the-fly
	p0 := worldMatrix.TransformPoint(quad.P0)
	p1 := worldMatrix.TransformPoint(quad.P1)
	p2 := worldMatrix.TransformPoint(quad.P2)
	p3 := worldMatrix.TransformPoint(quad.P3)

	// Create temporary quad
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

	// Convert to triangles and render
	triangles := ConvertQuadToTriangles(transformed)
	for _, tri := range triangles {
		if r.IsTriangleVisible(tri, camera) {
			if tri.Material.Wireframe {
				r.RenderTriangleWireframe(tri, camera)
			} else {
				tri.DrawFilled(r, camera)
			}
		}
	}
}

// RenderMeshWithMatrix renders a mesh with world matrix (transforms on-the-fly)
func (r *Renderer) RenderMeshWithMatrix(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// Transform mesh position
	meshPos := worldMatrix.TransformPoint(mesh.Position)

	// Render quads
	for _, quad := range mesh.Quads {
		// Apply mesh position offset
		offsetQuad := &Quad{
			P0:           Point{X: quad.P0.X + meshPos.X, Y: quad.P0.Y + meshPos.Y, Z: quad.P0.Z + meshPos.Z},
			P1:           Point{X: quad.P1.X + meshPos.X, Y: quad.P1.Y + meshPos.Y, Z: quad.P1.Z + meshPos.Z},
			P2:           Point{X: quad.P2.X + meshPos.X, Y: quad.P2.Y + meshPos.Y, Z: quad.P2.Z + meshPos.Z},
			P3:           Point{X: quad.P3.X + meshPos.X, Y: quad.P3.Y + meshPos.Y, Z: quad.P3.Z + meshPos.Z},
			Material:     quad.Material,
			UseSetNormal: quad.UseSetNormal,
			Normal:       quad.Normal,
		}
		r.RenderQuadWithMatrix(offsetQuad, IdentityMatrix(), camera)
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

		if r.IsTriangleVisible(offsetTri, camera) {
			if tri.Material.Wireframe {
				r.RenderTriangleWireframe(offsetTri, camera)
			} else {
				offsetTri.DrawFilled(r, camera)
			}
		}
	}
}

// RenderLineWithMatrix renders a line with world matrix
func (r *Renderer) RenderLineWithMatrix(line *Line, worldMatrix Matrix4x4, camera *Camera) {
	start := worldMatrix.TransformPoint(line.Start)
	end := worldMatrix.TransformPoint(line.End)

	transformedLine := &Line{Start: start, End: end}
	r.RenderLine(transformedLine, camera)
}

// RenderCircleWithMatrix renders a circle with world matrix
func (r *Renderer) RenderCircleWithMatrix(circle *Circle, worldMatrix Matrix4x4, camera *Camera) {
	center := worldMatrix.TransformPoint(circle.Center)
	points := make([]Point, len(circle.Points))
	for i, p := range circle.Points {
		points[i] = worldMatrix.TransformPoint(p)
	}

	transformedCircle := &Circle{
		Center: center,
		Radius: circle.Radius,
		Points: points,
	}
	r.RenderCircle(transformedCircle, camera)
}

// drawLineWithColorAndZ draws colored line with z-buffer
func (r *Renderer) drawLineWithColorAndZ(x0, y0, x1, y1 int, z0, z1 float64, color Color) {
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

// RenderLine renders a line
func (r *Renderer) RenderLine(line *Line, camera *Camera) {
	line.Project(r, camera)
}

// RenderCircle renders a circle
func (r *Renderer) RenderCircle(circle *Circle, camera *Camera) {
	circle.Project(r, camera)
}

// RenderFilled renders a filled object
func (r *Renderer) RenderFilled(obj Drawable, camera *Camera) {
	obj.DrawFilled(r, camera)
}

// Present writes the rendered frame to screen
func (r *Renderer) Present() {
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

			// Clear to end of line
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

	// Write everything at once
	r.Writer.WriteString(builder.String())
	r.Writer.Flush()
}

// RenderLoop starts the main render loop
type UpdateFunc func(scene *Scene, dt float64)

// RenderLoop starts the main render loop
func (r *Renderer) RenderLoop(scene *Scene, fps float64, updateFunc UpdateFunc) {
	if r.Writer == nil {
		panic("Renderer writer is nil")
	}

	// Enter alternate screen buffer
	r.Writer.WriteString("\033[?1049h")
	// Hide cursor
	r.Writer.WriteString("\033[?25l")
	// Clear screen
	r.Writer.WriteString("\033[2J\033[H")
	r.Writer.Flush()

	// Ensure cleanup on exit
	defer func() {
		r.Writer.WriteString("\033[?25h")
		r.Writer.WriteString("\033[?1049l")
		r.Writer.Flush()
	}()

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

		r.ShowDebugLine(scene.Camera, currentFPS)
	}
}

// SetShowDebugInfo enables or disables debug info display
func (r *Renderer) SetShowDebugInfo(show bool) {
	r.ShowDebugInfo = show
}

// ShowDebugLine displays FPS and camera info at the bottom
func (r *Renderer) ShowDebugLine(camera *Camera, fps float64) {
	if !r.ShowDebugInfo || camera == nil {
		return
	}

	pos := camera.GetPosition()
	pitch, yaw, roll := camera.GetRotation()

	// Build debug string
	r.debugBuffer.Reset()
	r.debugBuffer.WriteString(fmt.Sprintf("FPS: %.1f", fps))

	// Calculate padding
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

	// Only update if changed
	if debugLine != r.lastDebugLine {
		fmt.Fprintf(r.Writer, "\033[%d;1H", r.Height+1)
		fmt.Fprintf(r.Writer, "\033[K%s", debugLine)
		r.Writer.Flush()
		r.lastDebugLine = debugLine
	}
}
