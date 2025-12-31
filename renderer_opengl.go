package main

import (
	"fmt"
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
	vao          uint32
	vbo          uint32
	uniformModel int32
	uniformView  int32
	uniformProj  int32

	// Vertex data
	maxVertices     int
	currentVertices []float32 // Interleaved: pos(3) + color(3)

	// Settings
	UseColor       bool
	ShowDebugInfo  bool
	LightingSystem *LightingSystem
	Camera         *Camera

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
		currentVertices: make([]float32, 0, 60000), // 10k vertices * 6 floats
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

	// Create shader program
	if err := r.createShaderProgram(); err != nil {
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
	// Generate VAO
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

	return nil
}

func (r *OpenGLRenderer) Shutdown() {
	if !r.initialized {
		return
	}

	fmt.Println("[OpenGL] Shutting down...")

	// Delete OpenGL resources
	gl.DeleteBuffers(1, &r.vbo)
	gl.DeleteVertexArrays(1, &r.vao)
	gl.DeleteProgram(r.program)

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

	// Clear vertex data
	r.currentVertices = r.currentVertices[:0]
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
		dataSize := len(r.currentVertices) * 4 // 4 bytes per float
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, dataSize, gl.Ptr(r.currentVertices))
	}

	// Use shader program
	gl.UseProgram(r.program)

	// Set uniforms (MVP matrices)
	if r.Camera != nil {
		r.updateMatrices()
	}

	// Draw
	if len(r.currentVertices) > 0 {
		gl.BindVertexArray(r.vao)
		vertexCount := int32(len(r.currentVertices) / 6)
		gl.DrawArrays(gl.TRIANGLES, 0, vertexCount)
		gl.BindVertexArray(0)
	}

	r.window.SwapBuffers()
	r.frameCount++

	// Show FPS in title
	if r.frameCount%60 == 0 && r.ShowDebugInfo {
		r.window.SetTitle(fmt.Sprintf("Go 3D Engine (OpenGL) - Frame %d", r.frameCount))
	}
}

func (r *OpenGLRenderer) updateMatrices() {
	// Model matrix (identity for now - transforms are baked into vertices)
	modelMatrix := IdentityMatrix()
	r.uploadMatrix(r.uniformModel, modelMatrix)

	// View matrix (inverse of camera transform)
	viewMatrix := r.Camera.Transform.GetInverseMatrix()
	r.uploadMatrix(r.uniformView, viewMatrix)

	// Projection matrix
	projMatrix := r.buildProjectionMatrix()
	r.uploadMatrix(r.uniformProj, projMatrix)
}

func (r *OpenGLRenderer) buildProjectionMatrix() Matrix4x4 {
	if r.Camera == nil {
		return IdentityMatrix()
	}

	fovY := r.Camera.FOV.Y * 3.14159 / 180.0
	aspect := float64(r.width) / float64(r.height)
	near := r.Camera.Near
	far := r.Camera.Far

	f := 1.0 / tan(fovY/2.0)

	return Matrix4x4{M: [16]float64{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}}
}

func tan(x float64) float64 {
	// Simple tan implementation
	sin := 0.0
	cos := 1.0
	term := x

	for i := 1; i < 10; i++ {
		if i%2 == 1 {
			sin += term
			term *= -x * x / float64((2*i)*(2*i+1))
		}
		if i%2 == 0 {
			cos += term
			term *= -x * x / float64((2*i)*(2*i+1))
		}
	}

	if cos < 0.001 {
		cos = 0.001
	}
	return sin / cos
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

	r.BeginFrame()

	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
	}

	// Collect all geometry
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNode(node, worldMatrix, scene.Camera)
	}

	r.EndFrame()
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
	}
}

func (re *OpenGLRenderer) RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices to world space
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Get color
	color := tri.Material.DiffuseColor
	r := float32(color.R) / 255.0
	g := float32(color.G) / 255.0
	b := float32(color.B) / 255.0

	// Add vertices (interleaved: position + color)
	re.addVertex(p0, r, g, b)
	re.addVertex(p1, r, g, b)
	re.addVertex(p2, r, g, b)
}

func (r *OpenGLRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// Render all triangles
	for _, tri := range mesh.Triangles {
		// Offset by mesh position
		offsetTri := &Triangle{
			P0:       Point{X: tri.P0.X + mesh.Position.X, Y: tri.P0.Y + mesh.Position.Y, Z: tri.P0.Z + mesh.Position.Z},
			P1:       Point{X: tri.P1.X + mesh.Position.X, Y: tri.P1.Y + mesh.Position.Y, Z: tri.P1.Z + mesh.Position.Z},
			P2:       Point{X: tri.P2.X + mesh.Position.X, Y: tri.P2.Y + mesh.Position.Y, Z: tri.P2.Z + mesh.Position.Z},
			Material: tri.Material,
		}
		r.RenderTriangle(offsetTri, worldMatrix, camera)
	}

	// Render all quads
	for _, quad := range mesh.Quads {
		offsetQuad := &Quad{
			P0:       Point{X: quad.P0.X + mesh.Position.X, Y: quad.P0.Y + mesh.Position.Y, Z: quad.P0.Z + mesh.Position.Z},
			P1:       Point{X: quad.P1.X + mesh.Position.X, Y: quad.P1.Y + mesh.Position.Y, Z: quad.P1.Z + mesh.Position.Z},
			P2:       Point{X: quad.P2.X + mesh.Position.X, Y: quad.P2.Y + mesh.Position.Y, Z: quad.P2.Z + mesh.Position.Z},
			P3:       Point{X: quad.P3.X + mesh.Position.X, Y: quad.P3.Y + mesh.Position.Y, Z: quad.P3.Z + mesh.Position.Z},
			Material: quad.Material,
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
	// Lines would require a separate shader or GL_LINES mode
	// For simplicity, render as thin triangles or skip
}

func (r *OpenGLRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera) {
	// Points would require GL_POINTS mode
	// For simplicity, render as small triangles or skip
}

func (re *OpenGLRenderer) addVertex(p Point, r, g, b float32) {
	re.currentVertices = append(re.currentVertices,
		float32(p.X), float32(p.Y), float32(p.Z), // Position
		r, g, b, // Color
	)
}

func (r *OpenGLRenderer) SetLightingSystem(ls *LightingSystem) {
	r.LightingSystem = ls
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
	// Not applicable for OpenGL (uses viewport)
}

func (r *OpenGLRenderer) GetRenderContext() *RenderContext {
	return r.renderContext
}

// Helper to check if window should close
func (r *OpenGLRenderer) ShouldClose() bool {
	if r.window == nil {
		return true
	}
	return r.window.ShouldClose()
}
