package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// TerminalRenderer renders to a terminal using ANSI escape codes
type TerminalRenderer struct {
	simulationStatus    engine.SimulationStatus
	hasSimulationStatus bool
	sampleCount         int
	sampleDepth         []float64
	sampleColor         []engine.Color
	// LOD masks partition screen cells between outgoing and incoming geometry.
	lodMask     bool
	lodIncoming bool
	lodAlpha    float64

	RequestedWidth, RequestedHeight int
	Writer                          *bufio.Writer
	Height                          int
	Width                           int
	OffsetX, OffsetY                int
	Surface                         [][]rune
	ColorBuffer                     [][]engine.Color
	ZBuffer                         [][]float64
	Charset                         [9]rune
	UseColor                        bool
	ColorMode                       string
	restoreConsole                  func()
	lastPresent                     time.Time
	measuredFPS                     float64
	LightingSystem                  *engine.LightingSystem
	ShadowRenderer                  *engine.SimpleShadowRenderer
	Camera                          *engine.Camera
	ShowDebugInfo                   bool
	debugBuffer                     strings.Builder
	lastDebugLine                   string
	presentBuffer                   bytes.Buffer
	presentColors                   map[engine.Color]string

	frameBytes                          int
	measureWrite                        bool
	writeTime                           time.Duration
	timingHUD                           bool
	updateTime, rasterTime, presentTime time.Duration

	// Clipping bounds (inclusive min, exclusive max)
	ClipMinX, ClipMinY int
	ClipMaxX, ClipMaxY int
}

// NewTerminalRenderer creates a new terminal renderer
func NewTerminalRenderer(writer *bufio.Writer, height, width int) *TerminalRenderer {
	surface := make([][]rune, height)
	colorBuffer := make([][]engine.Color, height)
	zBuffer := make([][]float64, height)

	for i := range surface {
		surface[i] = make([]rune, width)
		colorBuffer[i] = make([]engine.Color, width)
		zBuffer[i] = make([]float64, width)
		for j := range surface[i] {
			surface[i][j] = engine.DefaultCharset[0]
			colorBuffer[i][j] = engine.ColorBlack
			zBuffer[i][j] = math.Inf(1)
		}
	}

	return &TerminalRenderer{
		Writer:         writer,
		Height:         height,
		Width:          width,
		Surface:        surface,
		ColorBuffer:    colorBuffer,
		ZBuffer:        zBuffer,
		Charset:        engine.DefaultCharset,
		UseColor:       true,
		ShowDebugInfo:  true,
		ShadowRenderer: engine.NewSimpleShadowRenderer(512), // Moderate resolution for CPU rendering
		ClipMinX:       0,
		ClipMinY:       0,
		ClipMaxX:       width,
		ClipMaxY:       height,
	}
}

// SetClipBounds sets the clipping region for subsequent draw calls
func (r *TerminalRenderer) SetClipBounds(minX, minY, maxX, maxY int) {
	r.ClipMinX = engine.ClampInt(minX, 0, r.Width)
	r.ClipMinY = engine.ClampInt(minY, 0, r.Height)
	r.ClipMaxX = engine.ClampInt(maxX, 0, r.Width)
	r.ClipMaxY = engine.ClampInt(maxY, 0, r.Height)
}

// Initialize sets up the terminal renderer
func (r *TerminalRenderer) Initialize() error {
	if r.Writer == nil {
		return fmt.Errorf("writer is nil")
	}
	restore, err := enableTerminalOutput()
	if err != nil {
		return err
	}
	r.restoreConsole = restore
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
		r.Writer.WriteString("\033[0m\033[?25h") // Show cursor
		r.Writer.WriteString("\033[?1049l")      // Exit alternate screen
		r.Writer.Flush()
	}
	if r.restoreConsole != nil {
		r.restoreConsole()
		r.restoreConsole = nil
	}
}

// BeginFrame clears all buffers
func (r *TerminalRenderer) BeginFrame() {
	r.clearSamples()
	// Reset clipping to full screen
	r.ClipMinX = 0
	r.ClipMinY = 0
	r.ClipMaxX = r.Width
	r.ClipMaxY = r.Height

	for y := 0; y < r.Height; y++ {
		for x := 0; x < r.Width; x++ {
			r.Surface[y][x] = ' '
			r.ColorBuffer[y][x] = engine.ColorBlack
			r.ZBuffer[y][x] = math.Inf(1)
		}
	}
}

// EndFrame does nothing for terminal renderer
func (r *TerminalRenderer) EndFrame() {
	// No-op for terminal
}

func (r *TerminalRenderer) GetRenderContext() *engine.RenderContext {
	return &engine.RenderContext{
		Camera:         r.Camera,
		LightingSystem: r.LightingSystem,
		ViewFrustum:    nil,
	}
}

// Present writes the frame to the terminal
func (r *TerminalRenderer) Present() {
	// Keep the large output allocation between frames, including after resize.
	builder := &r.presentBuffer
	builder.Reset()
	builder.Grow(r.Height * r.Width * 25)

	// Move cursor to home position
	builder.WriteString("\033[H")

	if r.UseColor {
		currentColor := engine.ColorBlack
		firstColor := true
		trueColor := r.ColorMode == "truecolor" || r.ColorMode == ""
		if !trueColor {
			if r.presentColors == nil {
				r.presentColors = make(map[engine.Color]string)
			}
			clear(r.presentColors)
		}
		colorCodes := r.presentColors
		for i := 0; i < r.Height; i++ {
			if r.OffsetX != 0 || r.OffsetY != 0 {
				fmt.Fprintf(builder, "\033[%d;%dH", r.OffsetY+i+1, r.OffsetX+1)
			}
			for j := 0; j < r.Width; j++ {
				char := r.Surface[i][j]
				color := r.ColorBuffer[i][j]

				if firstColor || color != currentColor {
					firstColor = false
					if trueColor {
						var scratch [20]byte
						code := strconv.AppendUint(append(scratch[:0], "\033[38;2;"...), uint64(color.R), 10)
						code = strconv.AppendUint(append(code, ';'), uint64(color.G), 10)
						code = strconv.AppendUint(append(code, ';'), uint64(color.B), 10)
						builder.Write(append(code, 'm'))
					} else {
						code, ok := colorCodes[color]
						if !ok {
							code = Foreground(color, r.ColorMode)
							colorCodes[color] = code
						}
						builder.WriteString(code)
					}
					currentColor = color
				}
				builder.WriteRune(char)
			}

			builder.WriteString("\033[K")
			if i < r.Height-1 && r.OffsetX == 0 && r.OffsetY == 0 {
				builder.WriteString("\r\n")
			}
		}
		builder.WriteString(engine.ColorReset())
	} else {
		for i := 0; i < r.Height; i++ {
			if r.OffsetX != 0 || r.OffsetY != 0 {
				fmt.Fprintf(builder, "\033[%d;%dH", r.OffsetY+i+1, r.OffsetX+1)
			}
			for j := 0; j < r.Width; j++ {
				builder.WriteRune(r.Surface[i][j])
			}
			builder.WriteString("\033[K")
			if i < r.Height-1 && r.OffsetX == 0 && r.OffsetY == 0 {
				builder.WriteString("\r\n")
			}
		}
	}

	r.frameBytes = builder.Len()
	var writeStart time.Time
	if r.measureWrite {
		writeStart = time.Now()
	}
	r.Writer.Write(builder.Bytes())
	r.Writer.Flush()
	if r.measureWrite {
		r.writeTime = time.Since(writeStart)
	}

	now := time.Now()
	if !r.lastPresent.IsZero() {
		r.measuredFPS = 1 / now.Sub(r.lastPresent).Seconds()
	}
	r.lastPresent = now
	r.showDebugLine()
}

// FrameOutput reports encoded bytes and, when enabled, the framebuffer write duration.
func (r *TerminalRenderer) FrameOutput() (int, time.Duration) { return r.frameBytes, r.writeTime }

func (r *TerminalRenderer) MeasureFrameWrite(enabled bool) {
	r.measureWrite = enabled
}

// prepareScene builds frame state on the caller's goroutine before workers read it.
func (r *TerminalRenderer) prepareScene(scene *engine.Scene) []*engine.SceneNode {
	r.BeginFrame()
	scene.Camera.Transform.GetInverseMatrix()
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		node.Transform.GetWorldMatrix()
	}
	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
		if r.ShadowRenderer != nil {
			for _, light := range r.LightingSystem.Lights {
				if light.IsEnabled {
					r.ShadowRenderer.RenderShadowMap(light, scene)
				}
			}
		}
	}
	return nodes
}

// RenderScene rasterizes all enabled scene objects in traversal order.
func (r *TerminalRenderer) RenderScene(scene *engine.Scene) {
	for _, node := range r.prepareScene(scene) {
		r.renderNode(node, node.Transform.GetWorldMatrix(), scene.Camera)
	}
	r.EndFrame()
}

func (r *TerminalRenderer) RenderSceneWithSpatialCulling(scene *engine.Scene) {
	// Compatibility entry point; scene traversal performs raster clipping.
	r.RenderScene(scene)
}

// renderNode renders a single scene node
func (r *TerminalRenderer) renderNode(node *engine.SceneNode, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	switch obj := node.Object.(type) {
	case *engine.Triangle:
		r.RenderTriangle(obj, worldMatrix, camera)
	case *engine.Quad:
		r.RenderQuad(obj, worldMatrix, camera)
	case *engine.Mesh:
		r.RenderMesh(obj, worldMatrix, camera)
	case *engine.LODGroupWithTransitions:
		r.renderLODTransition(obj, worldMatrix, camera)
	case *engine.LODGroup:
		if mesh := obj.GetCurrentMesh(); mesh != nil {
			r.RenderMesh(mesh, worldMatrix, camera)
		}
	case *engine.InstancedMesh:
		r.RenderInstancedMesh(obj, worldMatrix, camera)
	case *engine.Line:
		r.RenderLine(obj, worldMatrix, camera)
	case *engine.Circle:
		r.renderCircle(obj, worldMatrix, camera)
	case *engine.Point:
		r.RenderPoint(obj, worldMatrix, camera)
	}
}

// RenderTriangle renders a single triangle
func (r *TerminalRenderer) RenderTriangle(tri *engine.Triangle, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Transform normal if needed
	var transformedNormal *engine.Point
	if tri.UseSetNormal && tri.Normal != nil {
		tn := worldMatrix.TransformNormal(*tri.Normal)
		transformedNormal = &tn
	}

	r.renderTriangleInternal(p0, p1, p2, tri, transformedNormal, camera)
}

// renderTriangleInternal handles the core rendering logic for pre-transformed vertices
func (r *TerminalRenderer) renderTriangleInternal(p0, p1, p2 engine.Point, originalTri *engine.Triangle, transformedNormal *engine.Point, camera *engine.Camera) {
	v0 := camera.TransformToViewSpace(p0)
	v1 := camera.TransformToViewSpace(p1)
	v2 := camera.TransformToViewSpace(p2)

	if v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near {
		return
	}

	// Use object pool to reduce allocations
	transformed := engine.AcquireTriangle()
	defer engine.ReleaseTriangle(transformed)

	*transformed = *originalTri
	transformed.P0, transformed.P1, transformed.P2 = p0, p1, p2

	if originalTri.UseSetNormal && transformedNormal != nil {
		transformed.Normal = transformedNormal
	}

	if originalTri.Material.IsWireframe() {
		r.renderTriangleWireframe(transformed, camera)
	} else {
		r.rasterizeTriangleWithLighting(transformed, camera, [3]float64{v0.Z, v1.Z, v2.Z})
	}
}

// RenderLine renders a single line
func (r *TerminalRenderer) RenderLine(line *engine.Line, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	start := worldMatrix.TransformPoint(line.Start)
	end := worldMatrix.TransformPoint(line.End)

	transformedLine := &engine.Line{Start: start, End: end}

	clipped, visible := engine.ClipLineToNearPlane(transformedLine, camera)
	if !visible {
		return
	}

	r.renderLineProjected(clipped, camera, engine.ColorWhite)
}

// RenderPoint renders a single point
func (r *TerminalRenderer) RenderPoint(point *engine.Point, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	p := worldMatrix.TransformPoint(*point)

	x, y, z := camera.ProjectPoint(p, r.Height, r.Width)
	if z <= 0 || z > camera.Far {
		return
	}

	if x >= r.ClipMinX && x < r.ClipMaxX && y >= r.ClipMinY && y < r.ClipMaxY {
		r.writeCell(x, y, z, engine.ColorWhite, r.Charset[7])
	}
}

// RenderMesh renders a complete mesh
func (r *TerminalRenderer) RenderMesh(mesh *engine.Mesh, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	transformedVertices := make([]engine.Point, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		vertex.X += mesh.Position.X
		vertex.Y += mesh.Position.Y
		vertex.Z += mesh.Position.Z
		transformedVertices[i] = worldMatrix.TransformPoint(vertex)
	}

	// Reusable triangle struct for metadata passing to internal renderer
	// We only set metadata (Material, UVs) on this, not vertices
	tempTri := engine.AcquireTriangle()
	defer engine.ReleaseTriangle(tempTri)
	tempTri.Char = 'o'
	tempTri.Material = mesh.Material

	hasUVs := len(mesh.UVs) > 0

	// Render triangles from indexed geometry
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 >= 0 && idx1 >= 0 && idx2 >= 0 && idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				// Use pre-transformed vertices
				p0 := transformedVertices[idx0]
				p1 := transformedVertices[idx1]
				p2 := transformedVertices[idx2]

				tempTri.Material = mesh.MaterialForFace(i / 3)
				tempTri.HasUVs = false
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
func (r *TerminalRenderer) RenderInstancedMesh(instMesh *engine.InstancedMesh, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	if !instMesh.Enabled || instMesh.BaseMesh == nil {
		return
	}

	// Render each instance with its own transform
	for _, instance := range instMesh.Instances {
		// Combine world matrix with instance transform
		finalMatrix := worldMatrix.Multiply(instance.Transform)

		// Per-instance materials must not mutate the shared mesh during tiled rendering.
		mesh := *instMesh.BaseMesh
		if instance.Color != (engine.Color{}) {
			material := engine.NewMaterial()
			material.DiffuseColor = instance.Color
			mesh.Material = &material
			mesh.FaceMaterials = nil
		}
		r.RenderMesh(&mesh, finalMatrix, camera)
	}
}

// SetLightingSystem sets the lighting system
func (r *TerminalRenderer) SetLightingSystem(ls *engine.LightingSystem) {
	r.LightingSystem = ls
}

// SetCamera sets the camera
func (r *TerminalRenderer) SetCamera(camera *engine.Camera) {
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

// rasterizeTriangleWithLighting performs triangle rasterization with PERSPECTIVE CORRECT lighting
func (r *TerminalRenderer) rasterizeTriangleWithLighting(t *engine.Triangle, camera *engine.Camera, depths [3]float64) {
	normal := engine.CalculateSurfaceNormal(&t.P0, &t.P1, &t.P2, t.Normal, t.UseSetNormal)
	surfacePoint := engine.Point{
		X: (t.P0.X + t.P1.X + t.P2.X) / 3.0,
		Y: (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0,
		Z: (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0,
	}

	cameraDirX, cameraDirY, cameraDirZ := camera.GetCameraDirection(surfacePoint)
	facing := engine.DotProduct(normal.X, normal.Y, normal.Z, cameraDirX, cameraDirY, cameraDirZ)
	if facing < 0 {
		return
	}

	if depths[0] >= camera.Near && depths[1] >= camera.Near && depths[2] >= camera.Near {
		r.fillTriangleWithPerPixelLighting(t, camera, normal, t.Material)
		return
	}
	var clipped [2]engine.Triangle
	count := engine.ClipTriangleAtDepths(t, depths, camera.Near, &clipped)
	for i := range count {
		r.fillTriangleWithPerPixelLighting(&clipped[i], camera, normal, t.Material)
	}
}

// renderTriangleWireframe renders triangle edges
func (r *TerminalRenderer) renderTriangleWireframe(t *engine.Triangle, camera *engine.Camera) {
	line1 := engine.NewLine(t.P0, t.P1)
	line2 := engine.NewLine(t.P1, t.P2)
	line3 := engine.NewLine(t.P2, t.P0)

	clipped1, visible1 := engine.ClipLineToNearPlane(line1, camera)
	if visible1 {
		r.renderLineProjected(clipped1, camera, t.Material.GetWireframeColor())
	}

	clipped2, visible2 := engine.ClipLineToNearPlane(line2, camera)
	if visible2 {
		r.renderLineProjected(clipped2, camera, t.Material.GetWireframeColor())
	}

	clipped3, visible3 := engine.ClipLineToNearPlane(line3, camera)
	if visible3 {
		r.renderLineProjected(clipped3, camera, t.Material.GetWireframeColor())
	}
}

// renderLineProjected projects and renders a line with clipping
func (r *TerminalRenderer) renderLineProjected(line *engine.Line, camera *engine.Camera, color engine.Color) {
	sx0, sy0, z0 := camera.ProjectPoint(line.Start, r.Height, r.Width)
	sx1, sy1, z1 := camera.ProjectPoint(line.End, r.Height, r.Width)

	if z0 <= 0 || z1 <= 0 {
		return
	}
	r.drawLineWithZ(sx0, sy0, sx1, sy1, z0, z1, camera.Far, color)
}

// drawLineWithZ draws a line with z-buffering and clipping
func (r *TerminalRenderer) drawLineWithZ(x0, y0, x1, y1 int, z0, z1, farDepth float64, color engine.Color) {
	dx := x1 - x0
	dy := y1 - y0
	steps := engine.Abs(dx)
	if engine.Abs(dy) > steps {
		steps = engine.Abs(dy)
	}
	steps = max(1, steps)

	xStep := float64(dx) / float64(steps)
	yStep := float64(dy) / float64(steps)
	inverseZStep := (1/z1 - 1/z0) / float64(steps)

	x := float64(x0)
	y := float64(y0)
	inverseZ := 1 / z0

	for i := 0; i <= steps; i++ {
		xi := int(x + 0.5)
		yi := int(y + 0.5)

		z := 1 / inverseZ
		if xi >= r.ClipMinX && xi < r.ClipMaxX && yi >= r.ClipMinY && yi < r.ClipMaxY {
			if z <= farDepth {
				char := '*'
				if engine.Abs(dx) > engine.Abs(dy)*2 {
					char = '-'
				} else if engine.Abs(dy) > engine.Abs(dx)*2 {
					char = '|'
				}
				r.writeCell(xi, yi, z, color, char)
			}
		}
		x += xStep
		y += yStep
		inverseZ += inverseZStep
	}
}

// RenderQuad triangulates a quad while preserving material and transformed normals.
func (r *TerminalRenderer) RenderQuad(quad *engine.Quad, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	p0 := worldMatrix.TransformPoint(quad.P0)
	p1 := worldMatrix.TransformPoint(quad.P1)
	p2 := worldMatrix.TransformPoint(quad.P2)
	p3 := worldMatrix.TransformPoint(quad.P3)

	transformed := &engine.Quad{P0: p0, P1: p1, P2: p2, P3: p3, Material: quad.Material, UseSetNormal: quad.UseSetNormal}

	if quad.UseSetNormal && quad.Normal != nil {
		transformedNormal := worldMatrix.TransformNormal(*quad.Normal)
		transformed.Normal = &transformedNormal
	}

	triangles := engine.ConvertQuadToTriangles(transformed)
	for _, tri := range triangles {
		if r.isTriangleVisible(tri, camera) {
			if tri.Material.IsWireframe() {
				r.renderTriangleWireframe(tri, camera)
			} else {
				r.rasterizeTriangleWithLighting(tri, camera, [3]float64{camera.TransformToViewSpace(tri.P0).Z, camera.TransformToViewSpace(tri.P1).Z, camera.TransformToViewSpace(tri.P2).Z})
			}
		}
	}
}

// renderCircle renders a circle as connected line segments
func (r *TerminalRenderer) renderCircle(circle *engine.Circle, worldMatrix engine.Matrix4x4, camera *engine.Camera) {
	if len(circle.Points) == 0 {
		return
	}
	transformedPoints := make([]engine.Point, len(circle.Points))
	for i, p := range circle.Points {
		transformedPoints[i] = worldMatrix.TransformPoint(p)
	}
	for i := 0; i < len(transformedPoints); i++ {
		p1 := transformedPoints[i]
		p2 := transformedPoints[(i+1)%len(transformedPoints)]
		line := engine.NewLine(p1, p2)
		clipped, visible := engine.ClipLineToNearPlane(line, camera)
		if visible {
			r.renderLineProjected(clipped, camera, engine.ColorWhite)
		}
	}
}

func (r *TerminalRenderer) isTriangleVisible(t *engine.Triangle, camera *engine.Camera) bool {
	v0 := camera.TransformToViewSpace(t.P0)
	v1 := camera.TransformToViewSpace(t.P1)
	v2 := camera.TransformToViewSpace(t.P2)
	return !(v0.Z <= camera.Near && v1.Z <= camera.Near && v2.Z <= camera.Near)
}

func (r *TerminalRenderer) simpleLighting(normal engine.Point, material engine.IMaterial) engine.Color {
	ao := engine.CalculateSimpleAO(normal)
	lx, ly, lz := -1.0, 1.0, -1.0
	lx, ly, lz = engine.NormalizeVector(lx, ly, lz)
	intensity := engine.DotProduct(normal.X, normal.Y, normal.Z, lx, ly, lz)
	if intensity < 0 {
		intensity = 0
	}
	intensity *= ao

	diffuseColor := material.GetDiffuseColor(0, 0)
	return engine.Color{
		R: uint8(engine.Clamp(float64(diffuseColor.R)*intensity, 0, 255)),
		G: uint8(engine.Clamp(float64(diffuseColor.G)*intensity, 0, 255)),
		B: uint8(engine.Clamp(float64(diffuseColor.B)*intensity, 0, 255)),
	}
}

// SetSimulationStatus updates the status snapshot before presenting a frame.
func (r *TerminalRenderer) SetSimulationStatus(status engine.SimulationStatus) {
	r.simulationStatus = status
	r.hasSimulationStatus = true
}

func (r *TerminalRenderer) showDebugLine() {
	line := r.statusLine()
	if len(line) > r.Width {
		line = line[:r.Width]
	}
	if line != r.lastDebugLine {
		fmt.Fprintf(r.Writer, "\033[%d;%dH\033[K%s", r.OffsetY+r.Height+1, r.OffsetX+1, line)
		r.Writer.Flush()
		r.lastDebugLine = line
	}
}

func (r *TerminalRenderer) SetFrameTiming(enabled bool, update, raster, present time.Duration) {
	r.timingHUD = enabled
	r.updateTime, r.rasterTime, r.presentTime = update, raster, present
}

func (r *TerminalRenderer) statusLine() string {
	if r.timingHUD {
		ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
		return fmt.Sprintf("Update %.1fms | Raster %.1fms | Present %.1fms", ms(r.updateTime), ms(r.rasterTime), ms(r.presentTime))
	}
	status := r.simulationStatus
	if !r.ShowDebugInfo && (!r.hasSimulationStatus || !status.Paused) {
		return ""
	}
	r.debugBuffer.Reset()
	if r.hasSimulationStatus {
		state := "RUNNING"
		if status.Paused {
			state = "PAUSED"
		}
		fmt.Fprintf(&r.debugBuffer, "%s %.0fHz Tick:%d", state, status.Hz, status.Tick)
		if status.Paused {
			r.debugBuffer.WriteString(" | N step | Enter resume")
		}
	}
	if r.ShowDebugInfo {
		if r.debugBuffer.Len() > 0 {
			r.debugBuffer.WriteString(" | ")
		}
		fmt.Fprintf(&r.debugBuffer, "FPS: %.1f", r.measuredFPS)
		if status.DroppedSeconds > 0 || status.DroppedCommands > 0 {
			fmt.Fprintf(&r.debugBuffer, " | Drop:%.3fs Cmd:%d", status.DroppedSeconds, status.DroppedCommands)
		}
		if status.PhysicsBodies > 0 {
			if status.PhysicsOverlay {
				r.debugBuffer.WriteString(" | C:ON")
			} else {
				r.debugBuffer.WriteString(" | C:OFF")
			}
			fmt.Fprintf(&r.debugBuffer, " | Bodies:%d Sleep:%d Pairs:%d Contacts:%d Phys:%.2fms", status.PhysicsBodies, status.PhysicsSleeping, status.PhysicsPairs, status.PhysicsContacts, status.PhysicsMilliseconds)
		}
		if r.Camera != nil {
			pos := r.Camera.GetPosition()
			fmt.Fprintf(&r.debugBuffer, " | Pos:(%.1f,%.1f,%.1f)", pos.X, pos.Y, pos.Z)
		}
		if r.hasSimulationStatus && !status.Paused {
			r.debugBuffer.WriteString(" | Space pause")
		}
	}
	return r.debugBuffer.String()
}
