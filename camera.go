package main

// Camera represents the viewing frustum and projection parameters
type Camera struct {
	Transform *Transform // Unified transform system
	FOV       Point      // Field of view (X, Y angles)
	Near      float64    // Near clipping plane
	Far       float64    // Far clipping plane
	DZ        float64    // Z offset for projection (for backward compatibility)
}

// NewCamera creates a new camera with default settings
func NewCamera() *Camera {
	transform := NewTransformAt(0, 0, DEFAULT_CAMERA_Z)
	return &Camera{
		Transform: transform,
		FOV:       Point{X: FOV_X, Y: FOV_Y, Z: 0},
		Near:      0.1,
		Far:       100000.0,
		DZ:        DEFAULT_DZ,
	}
}

// NewCameraAt creates a camera at a specific position
func NewCameraAt(x, y, z float64) *Camera {
	cam := NewCamera()
	cam.Transform.SetPosition(x, y, z)
	return cam
}

// Legacy accessors for backward compatibility
func (cam *Camera) GetPosition() Point {
	return cam.Transform.GetWorldPosition()
}

func (cam *Camera) SetPosition(x, y, z float64) {
	cam.Transform.SetPosition(x, y, z)
}

func (cam *Camera) GetRotation() (pitch, yaw, roll float64) {
	rot := cam.Transform.GetWorldRotation()
	return rot.X, rot.Y, rot.Z
}

func (cam *Camera) SetRotation(yaw, pitch float64) {
	cam.Transform.SetRotation(pitch, yaw, 0)
}

func (cam *Camera) MoveBy(dx, dy, dz float64) {
	cam.Transform.Translate(dx, dy, dz)
}

// TransformToViewSpace transforms a world-space point to camera/view space
func (cam *Camera) TransformToViewSpace(p Point) Point {
	// Use cached inverse matrix
	return cam.Transform.InverseTransformPoint(p)
}

// ProjectPoint projects a 3D point to 2D screen coordinates
// Returns screen X, Y, and the Z depth (for z-buffering)
// Returns (-1, -1, 0) if point is behind camera
func (cam *Camera) ProjectPoint(p Point, canvasHeight, canvasWidth int) (int, int, float64) {
	// Transform to view space first
	viewPoint := cam.TransformToViewSpace(p)

	// In view space, Z is the depth (distance in front of camera)
	// Positive Z = in front of camera
	zDepth := viewPoint.Z

	// Check near plane clipping
	if zDepth <= cam.Near {
		return -1, -1, 0
	}

	// Perspective projection (simple pinhole camera model)
	projX := (viewPoint.X * cam.FOV.X) / zDepth
	projY := (viewPoint.Y * cam.FOV.Y) / zDepth

	// Convert to screen coordinates
	screenX, screenY := normalize(canvasHeight, canvasWidth, int(projX), int(projY))

	return screenX, screenY, zDepth
}

// GetViewDirection returns the normalized direction vector from a point to the camera
func (cam *Camera) GetViewDirection(point Point) (float64, float64, float64) {
	camPos := cam.GetPosition()
	dirX := camPos.X - point.X
	dirY := camPos.Y - point.Y
	dirZ := camPos.Z - point.Z

	return normalizeVector(dirX, dirY, dirZ)
}

// GetForwardVector returns the direction the camera is looking
func (cam *Camera) GetForwardVector() (float64, float64, float64) {
	forward := cam.Transform.GetForwardVector()
	return forward.X, forward.Y, forward.Z
}

// GetRightVector returns the right direction of the camera
func (cam *Camera) GetRightVector() (float64, float64, float64) {
	right := cam.Transform.GetRightVector()
	return right.X, right.Y, right.Z
}

// GetUpVector returns the up direction of the camera
func (cam *Camera) GetUpVector() (float64, float64, float64) {
	up := cam.Transform.GetUpVector()
	return up.X, up.Y, up.Z
}

// IsPointVisible checks if a point is within the camera's view frustum
func (cam *Camera) IsPointVisible(p Point) bool {
	viewPoint := cam.TransformToViewSpace(p)
	zDepth := viewPoint.Z + cam.DZ
	return zDepth > cam.Near && zDepth < cam.Far
}

// IsSphereVisible performs frustum culling on a bounding sphere
// Returns true if the sphere is potentially visible
func (cam *Camera) IsSphereVisible(center Point, radius float64) bool {
	// Transform center to view space
	viewCenter := cam.TransformToViewSpace(center)
	zDepth := viewCenter.Z + cam.DZ

	// Early rejection: behind near plane or beyond far plane (with margin)
	if zDepth+radius < cam.Near || zDepth-radius > cam.Far {
		return false
	}

	// Check if sphere is within view frustum
	// Calculate the half-angles of the FOV
	halfFOVX := cam.FOV.X / 2.0
	halfFOVY := cam.FOV.Y / 2.0

	// Project the sphere center
	if zDepth > 0 {
		projX := (viewCenter.X * cam.FOV.X) / zDepth
		projY := (viewCenter.Y * cam.FOV.Y) / zDepth

		// Calculate projected radius (conservative estimate)
		projRadius := (radius * cam.FOV.X) / (zDepth - radius)
		if projRadius < 0 {
			projRadius = 0
		}

		// Check if sphere is within frustum bounds (with margin)
		if projX+projRadius < -halfFOVX || projX-projRadius > halfFOVX {
			return false
		}
		if projY+projRadius < -halfFOVY || projY-projRadius > halfFOVY {
			return false
		}
	}

	return true
}

// GetCameraDirection returns the normalized direction from surface to camera
// Used for backface culling and lighting calculations
func (cam *Camera) GetCameraDirection(surfacePoint Point) (float64, float64, float64) {
	camPos := cam.GetPosition()
	cameraDirX := camPos.X - surfacePoint.X
	cameraDirY := camPos.Y - surfacePoint.Y
	cameraDirZ := camPos.Z - surfacePoint.Z

	return normalizeVector(cameraDirX, cameraDirY, cameraDirZ)
}

// SetFOV sets the field of view
func (cam *Camera) SetFOV(fovX, fovY float64) {
	cam.FOV.X = fovX
	cam.FOV.Y = fovY
}

// LookAt makes the camera look at a target position
func (cam *Camera) LookAt(target Point) {
	cam.Transform.LookAt(target)
}

// MoveForward moves the camera forward in its local space
func (cam *Camera) MoveForward(distance float64) {
	forward := cam.Transform.GetForwardVector()
	cam.Transform.Translate(forward.X*distance, forward.Y*distance, forward.Z*distance)
}

// MoveRight moves the camera right in its local space
func (cam *Camera) MoveRight(distance float64) {
	right := cam.Transform.GetRightVector()
	cam.Transform.Translate(right.X*distance, right.Y*distance, right.Z*distance)
}

// MoveUp moves the camera up in its local space (or world space if preferred)
func (cam *Camera) MoveUp(distance float64) {
	// World-space up (usually preferred for cameras)
	cam.Transform.Translate(0, distance, 0)
}

// MoveUpLocal moves the camera up in its local space
func (cam *Camera) MoveUpLocal(distance float64) {
	up := cam.Transform.GetUpVector()
	cam.Transform.Translate(up.X*distance, up.Y*distance, up.Z*distance)
}

// RotateYaw rotates the camera around the Y axis (left/right look)
func (cam *Camera) RotateYaw(angle float64) {
	cam.Transform.Rotate(0, angle, 0)
}

// RotatePitch rotates the camera around the X axis (up/down look)
func (cam *Camera) RotatePitch(angle float64) {
	cam.Transform.Rotate(angle, 0, 0)
}

// RotateRoll rotates the camera around the Z axis (tilt)
func (cam *Camera) RotateRoll(angle float64) {
	cam.Transform.Rotate(0, 0, angle)
}
