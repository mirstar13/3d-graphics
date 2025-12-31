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
	ViewFrustum    *Frustum // For culling
}

// Frustum represents a view frustum for culling
type Frustum struct {
	Planes [6]Plane // Left, Right, Top, Bottom, Near, Far
}

// Plane represents a plane equation: ax + by + cz + d = 0
type Plane struct {
	Normal   Point
	Distance float64
}

// NewFrustumFromCamera creates a frustum from camera parameters
func NewFrustumFromCamera(camera *Camera) *Frustum {
	// TODO: Implement proper frustum extraction
	return &Frustum{}
}

// ContainsSphere checks if a sphere intersects the frustum
func (f *Frustum) ContainsSphere(center Point, radius float64) bool {
	// TODO: Implement sphere-frustum test
	return true
}

// ContainsAABB checks if an AABB intersects the frustum
func (f *Frustum) ContainsAABB(aabb *AABB) bool {
	// TODO: Implement AABB-frustum test
	return true
}
