package main

// Renderer is the core interface all renderer implementations must satisfy
type Renderer interface {
	// Lifecycle management
	Initialize() error
	Shutdown()

	// Frame management
	BeginFrame()
	EndFrame()
	Present()

	// Primitive rendering (transforms are applied externally)
	RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera)
	RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera)
	RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera)
	RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera)

	// Scene rendering
	RenderScene(scene *Scene)

	// Configuration
	SetLightingSystem(ls *LightingSystem)
	SetCamera(camera *Camera)
	GetDimensions() (width, height int)

	// Settings
	SetUseColor(useColor bool)
	SetShowDebugInfo(show bool)

	// Clipping (New for Parallel Rendering Safety)
	SetClipBounds(minX, minY, maxX, maxY int)

	// Getters for shared context
	GetRenderContext() *RenderContext
}

// RenderContext contains shared rendering state
type RenderContext struct {
	Camera         *Camera
	LightingSystem *LightingSystem
	ViewFrustum    *ViewFrustum // For culling
}
