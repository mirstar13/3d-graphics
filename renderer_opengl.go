package main

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// OpenGLRenderer renders using OpenGL 4.1
type OpenGLRenderer struct {
	window        *glfw.Window
	width         int
	height        int
	renderContext *RenderContext

	// OpenGL resources
	program      uint32
	lineProgram  uint32
	vao          uint32
	vbo          uint32
	lineVAO      uint32
	lineVBO      uint32
	uniformModel int32
	uniformView  int32
	uniformProj  int32

	// Line shader uniforms
	lineUniformModel int32
	lineUniformView  int32
	lineUniformProj  int32

	// Vertex data
	maxVertices     int
	currentVertices []VulkanVertex // Interleaved: pos(3) + color(3)
	lineVertices    []float32

	// Settings
	UseColor       bool
	ShowDebugInfo  bool
	LightingSystem *LightingSystem
	Camera         *Camera

	// Clipping (not used in OpenGL but required by interface)
	clipMinX, clipMinY, clipMaxX, clipMaxY int

	initialized bool
	frameCount  int
}

const (
	vertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aColor;

out vec3 FragColor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;

void main() {
    gl_Position = proj * view * model * vec4(aPos, 1.0);
    FragColor = aColor;
}
` + "\x00"

	fragmentShaderSource = `
#version 410 core
in vec3 FragColor;
out vec4 color;

void main() {
    color = vec4(FragColor, 1.0);
}
` + "\x00"

	lineVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aColor;

out vec3 FragColor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;

void main() {
    gl_Position = proj * view * model * vec4(aPos, 1.0);
    FragColor = aColor;
}
` + "\x00"

	lineFragmentShaderSource = `
#version 410 core
in vec3 FragColor;
out vec4 color;

void main() {
    color = vec4(FragColor, 1.0);
}
` + "\x00"
)

func NewOpenGLRenderer(width, height int) *OpenGLRenderer {
	return &OpenGLRenderer{
		width:  width,
		height: height,
		renderContext: &RenderContext{
			ViewFrustum: &ViewFrustum{},
		},
		UseColor:        true,
		ShowDebugInfo:   false,
		maxVertices:     100000,
		currentVertices: make([]VulkanVertex, 0, 600000), // 100k vertices * 6 floats
		lineVertices:    make([]float32, 0, 60000),       // 10k line vertices
	}
}

func (r *OpenGLRenderer) Initialize() error {
	if r.initialized {
		return nil
	}

	fmt.Println("[OpenGL] Initializing...")

	// Lock to OS thread (required for OpenGL)
	runtime.LockOSThread()

	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize GLFW: %v", err)
	}

	// Set OpenGL version
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	// Create window
	window, err := glfw.CreateWindow(r.width, r.height, "Go 3D Engine (OpenGL)", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}
	r.window = window

	r.window.MakeContextCurrent()

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL: %v", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Printf("[OpenGL] Version: %s\n", version)

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// Create shader programs
	if err := r.createShaderProgram(); err != nil {
		return err
	}

	if err := r.createLineShaderProgram(); err != nil {
		return err
	}

	// Create vertex buffers
	if err := r.createBuffers(); err != nil {
		return err
	}

	// Set viewport
	gl.Viewport(0, 0, int32(r.width), int32(r.height))

	fmt.Println("[OpenGL] Initialization complete")
	r.initialized = true
	return nil
}

func (r *OpenGLRenderer) createShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link program: %v", log)
	}

	r.program = program

	// Get uniform locations
	r.uniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.uniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.uniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))

	return nil
}

func (r *OpenGLRenderer) createLineShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(lineVertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("line vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(lineFragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("line fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link line program: %v", log)
	}

	r.lineProgram = program

	// Get uniform locations
	r.lineUniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.lineUniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.lineUniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))

	return nil
}

func (r *OpenGLRenderer) compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	// Check for compilation errors
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}

	return shader, nil
}

func (r *OpenGLRenderer) createBuffers() error {
	// Generate VAO for triangles
	gl.GenVertexArrays(1, &r.vao)
	gl.BindVertexArray(r.vao)

	// Generate VBO
	gl.GenBuffers(1, &r.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)

	// Allocate buffer (dynamic)
	bufferSize := r.maxVertices * 6 * 4 // 6 floats per vertex, 4 bytes per float
	gl.BufferData(gl.ARRAY_BUFFER, bufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Color attribute (location 1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	// Generate VAO for lines
	gl.GenVertexArrays(1, &r.lineVAO)
	gl.BindVertexArray(r.lineVAO)

	// Generate line VBO
	gl.GenBuffers(1, &r.lineVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.lineVBO)

	// Allocate buffer for lines
	lineBufferSize := 10000 * 6 * 4 // 10k vertices * 6 floats * 4 bytes
	gl.BufferData(gl.ARRAY_BUFFER, lineBufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Color attribute (location 1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	return nil
}

func (r *OpenGLRenderer) Shutdown() {
	if !r.initialized {
		return
	}

	fmt.Println("[OpenGL] Shutting down...")

	// Delete OpenGL resources
	gl.DeleteBuffers(1, &r.vbo)
	gl.DeleteBuffers(1, &r.lineVBO)
	gl.DeleteVertexArrays(1, &r.vao)
	gl.DeleteVertexArrays(1, &r.lineVAO)
	gl.DeleteProgram(r.program)
	gl.DeleteProgram(r.lineProgram)

	r.window.Destroy()
	glfw.Terminate()
	r.initialized = false
}

func (r *OpenGLRenderer) BeginFrame() {
	if !r.initialized {
		return
	}

	glfw.PollEvents()

	// Clear buffers
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (r *OpenGLRenderer) EndFrame() {
	// No-op for OpenGL (rendering happens in Present)
}

func (r *OpenGLRenderer) Present() {
	if !r.initialized {
		return
	}

	// Upload vertex data
	if len(r.currentVertices) > 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
		dataSize := len(r.currentVertices) * 24 // 6 floats * 4 bytes per float
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, dataSize, gl.Ptr(r.currentVertices))
	}

	// Use shader program
	gl.UseProgram(r.program)

	// Draw
	if len(r.currentVertices) > 0 {
		gl.BindVertexArray(r.vao)
		vertexCount := int32(len(r.currentVertices))
		gl.DrawArrays(gl.TRIANGLES, 0, vertexCount)
		gl.BindVertexArray(0)
	}

	r.window.SwapBuffers()
	r.frameCount++

	// Show FPS in title
	if r.frameCount%60 == 0 && r.ShowDebugInfo {
		r.window.SetTitle(fmt.Sprintf("Go 3D Engine (OpenGL) - Frame %d - Vertices: %d", r.frameCount, len(r.currentVertices)))
	}
}

func (r *OpenGLRenderer) updateMatrices(modelUniform, viewUniform, projUniform int32) {
	// Model matrix (identity - transforms are baked into vertices)
	modelMatrix := IdentityMatrix()
	r.uploadMatrix(modelUniform, modelMatrix)

	// View matrix (inverse of camera transform)
	viewMatrix := r.Camera.Transform.GetInverseMatrix()
	r.uploadMatrix(viewUniform, viewMatrix)

	// Projection matrix
	projMatrix := r.buildProjectionMatrix()
	r.uploadMatrix(projUniform, projMatrix)
}

func (r *OpenGLRenderer) buildProjectionMatrix() Matrix4x4 {
	if r.Camera == nil {
		return IdentityMatrix()
	}

	fovY := r.Camera.FOV.Y * math.Pi / 180.0
	aspect := float64(r.width) / float64(r.height)
	near := r.Camera.Near
	far := r.Camera.Far

	f := 1.0 / math.Tan(fovY/2.0)

	return Matrix4x4{M: [16]float64{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}}
}

func (r *OpenGLRenderer) uploadMatrix(uniform int32, matrix Matrix4x4) {
	// Convert to float32 array
	var m [16]float32
	for i := 0; i < 16; i++ {
		m[i] = float32(matrix.M[i])
	}
	gl.UniformMatrix4fv(uniform, 1, false, &m[0])
}

func (r *OpenGLRenderer) RenderScene(scene *Scene) {
	if !r.initialized {
		return
	}

	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
	}

	// Clear vertex buffer at start of scene rendering
	r.currentVertices = r.currentVertices[:0]

	// Collect all geometry
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNode(node, worldMatrix, scene.Camera)
	}

	// Update matrices now that we have a camera
	if r.Camera != nil {
		gl.UseProgram(r.program)
		r.updateMatrices(r.uniformModel, r.uniformView, r.uniformProj)
	}
}

func (r *OpenGLRenderer) renderNode(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Triangle:
		r.RenderTriangle(obj, worldMatrix, camera)
	case *Quad:
		r.renderQuad(obj, worldMatrix, camera)
	case *Mesh:
		r.RenderMesh(obj, worldMatrix, camera)
	case *Line:
		r.RenderLine(obj, worldMatrix, camera)
	case *Point:
		r.RenderPoint(obj, worldMatrix, camera)
	case *LODGroup:
		// Render current LOD mesh
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil {
			r.RenderMesh(currentMesh, worldMatrix, camera)
		}
	case *LODGroupWithTransitions:
		// Render current LOD mesh
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil {
			r.RenderMesh(currentMesh, worldMatrix, camera)
		}
	}
}

func (r *OpenGLRenderer) RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices to world space
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Get color
	color := tri.Material.DiffuseColor

	// Apply simple lighting if available
	if r.LightingSystem != nil {
		// Calculate normal
		normal := CalculateSurfaceNormal(&tri.P0, &tri.P1, &tri.P2, tri.Normal, tri.UseSetNormal)
		worldNormal := worldMatrix.TransformDirection(normal)

		// Simple diffuse lighting
		intensity := 0.3 // Ambient
		for _, light := range r.LightingSystem.Lights {
			if !light.IsEnabled {
				continue
			}

			centerX := (p0.X + p1.X + p2.X) / 3.0
			centerY := (p0.Y + p1.Y + p2.Y) / 3.0
			centerZ := (p0.Z + p1.Z + p2.Z) / 3.0

			lx := light.Position.X - centerX
			ly := light.Position.Y - centerY
			lz := light.Position.Z - centerZ
			lx, ly, lz = normalizeVector(lx, ly, lz)

			diff := dotProduct(worldNormal.X, worldNormal.Y, worldNormal.Z, lx, ly, lz)
			if diff > 0 {
				intensity += diff * light.Intensity * 0.7
			}
		}

		if intensity > 1.0 {
			intensity = 1.0
		}

		color = Color{
			R: uint8(float64(color.R) * intensity),
			G: uint8(float64(color.G) * intensity),
			B: uint8(float64(color.B) * intensity),
		}
	}

	rf := float32(color.R) / 255.0
	gf := float32(color.G) / 255.0
	bf := float32(color.B) / 255.0

	// Add vertices (interleaved: position + color)
	r.addVertex(p0, rf, gf, bf)
	r.addVertex(p1, rf, gf, bf)
	r.addVertex(p2, rf, gf, bf)
}

func (r *OpenGLRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// Render all triangles
	for _, tri := range mesh.Triangles {
		// Offset by mesh position
		offsetTri := &Triangle{
			P0:           Point{X: tri.P0.X + mesh.Position.X, Y: tri.P0.Y + mesh.Position.Y, Z: tri.P0.Z + mesh.Position.Z},
			P1:           Point{X: tri.P1.X + mesh.Position.X, Y: tri.P1.Y + mesh.Position.Y, Z: tri.P1.Z + mesh.Position.Z},
			P2:           Point{X: tri.P2.X + mesh.Position.X, Y: tri.P2.Y + mesh.Position.Y, Z: tri.P2.Z + mesh.Position.Z},
			Material:     tri.Material,
			Normal:       tri.Normal,
			UseSetNormal: tri.UseSetNormal,
		}
		r.RenderTriangle(offsetTri, worldMatrix, camera)
	}

	// Render all quads
	for _, quad := range mesh.Quads {
		offsetQuad := &Quad{
			P0:           Point{X: quad.P0.X + mesh.Position.X, Y: quad.P0.Y + mesh.Position.Y, Z: quad.P0.Z + mesh.Position.Z},
			P1:           Point{X: quad.P1.X + mesh.Position.X, Y: quad.P1.Y + mesh.Position.Y, Z: quad.P1.Z + mesh.Position.Z},
			P2:           Point{X: quad.P2.X + mesh.Position.X, Y: quad.P2.Y + mesh.Position.Y, Z: quad.P2.Z + mesh.Position.Z},
			P3:           Point{X: quad.P3.X + mesh.Position.X, Y: quad.P3.Y + mesh.Position.Y, Z: quad.P3.Z + mesh.Position.Z},
			Material:     quad.Material,
			Normal:       quad.Normal,
			UseSetNormal: quad.UseSetNormal,
		}
		r.renderQuad(offsetQuad, worldMatrix, camera)
	}
}

func (r *OpenGLRenderer) renderQuad(quad *Quad, worldMatrix Matrix4x4, camera *Camera) {
	triangles := ConvertQuadToTriangles(quad)
	for _, tri := range triangles {
		r.RenderTriangle(tri, worldMatrix, camera)
	}
}

func (r *OpenGLRenderer) RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices to world space
	p0 := worldMatrix.TransformPoint(line.Start)
	p1 := worldMatrix.TransformPoint(line.End)

	// Use white color for lines
	rf := float32(1.0)
	gf := float32(1.0)
	bf := float32(1.0)

	// Add line vertices
	r.addLineVertex(p0, rf, gf, bf)
	r.addLineVertex(p1, rf, gf, bf)
}

func (r *OpenGLRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera) {
	// Transform point to world space
	p := worldMatrix.TransformPoint(*point)

	// Render as small sphere approximation (octahedron)
	size := 0.5
	color := Color{255, 255, 255}
	rf := float32(color.R) / 255.0
	gf := float32(color.G) / 255.0
	bf := float32(color.B) / 255.0

	// Create 8 triangles forming an octahedron
	top := Point{X: p.X, Y: p.Y + size, Z: p.Z}
	bottom := Point{X: p.X, Y: p.Y - size, Z: p.Z}
	front := Point{X: p.X, Y: p.Y, Z: p.Z + size}
	back := Point{X: p.X, Y: p.Y, Z: p.Z - size}
	left := Point{X: p.X - size, Y: p.Y, Z: p.Z}
	right := Point{X: p.X + size, Y: p.Y, Z: p.Z}

	// Top pyramid
	r.addVertex(top, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)

	// Bottom pyramid
	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
}

func (r *OpenGLRenderer) addVertex(p Point, red, green, blue float32) {
	r.currentVertices = append(r.currentVertices,
		VulkanVertex{
			Pos:   [3]float32{float32(p.X), float32(p.Y), float32(p.Z)},
			Color: [3]float32{red, green, blue},
		},
	)
}

func (r *OpenGLRenderer) addLineVertex(p Point, red, green, blue float32) {
	r.lineVertices = append(r.lineVertices,
		float32(p.X), float32(p.Y), float32(p.Z), // Position
		red, green, blue, // Color
	)
}

func (r *OpenGLRenderer) SetLightingSystem(ls *LightingSystem) {
	r.LightingSystem = ls
	r.renderContext.LightingSystem = ls
}

func (r *OpenGLRenderer) SetCamera(camera *Camera) {
	r.Camera = camera
	r.renderContext.Camera = camera
}

func (r *OpenGLRenderer) GetDimensions() (int, int) {
	return r.width, r.height
}

func (r *OpenGLRenderer) SetUseColor(useColor bool) {
	r.UseColor = useColor
}

func (r *OpenGLRenderer) SetShowDebugInfo(show bool) {
	r.ShowDebugInfo = show
}

func (r *OpenGLRenderer) SetClipBounds(minX, minY, maxX, maxY int) {
	// Store for interface compliance, but not used in OpenGL
	r.clipMinX = minX
	r.clipMinY = minY
	r.clipMaxX = maxX
	r.clipMaxY = maxY
}

func (r *OpenGLRenderer) GetRenderContext() *RenderContext {
	return r.renderContext
}

// ShouldClose checks if window should close
func (r *OpenGLRenderer) ShouldClose() bool {
	if r.window == nil {
		return true
	}
	return r.window.ShouldClose()
}

// GetWindow returns the GLFW window (for input handling)
func (r *OpenGLRenderer) GetWindow() *glfw.Window {
	return r.window
}
