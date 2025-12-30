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

func (cam *Camera) SetRotation(pitch, yaw, roll float64) {
	cam.Transform.SetRotation(pitch, yaw, roll)
}

func (cam *Camera) MoveBy(dx, dy, dz float64) {
	cam.Transform.Translate(dx, dy, dz)
}

// TransformToViewSpace transforms a world-space point to camera/view space
func (cam *Camera) TransformToViewSpace(p Point) Point {
	return cam.Transform.InverseTransformPoint(p)
}

// ProjectPoint projects a 3D point to 2D screen coordinates
func (cam *Camera) ProjectPoint(p Point, canvasHeight, canvasWidth int) (int, int, float64) {
	viewPoint := cam.TransformToViewSpace(p)
	zDepth := viewPoint.Z

	if zDepth <= cam.Near {
		return -1, -1, 0
	}

	projX := (viewPoint.X * cam.FOV.X) / zDepth
	projY := (viewPoint.Y * cam.FOV.Y) / zDepth

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
func (cam *Camera) IsSphereVisible(center Point, radius float64) bool {
	viewCenter := cam.TransformToViewSpace(center)
	zDepth := viewCenter.Z + cam.DZ

	if zDepth+radius < cam.Near || zDepth-radius > cam.Far {
		return false
	}

	halfFOVX := cam.FOV.X / 2.0
	halfFOVY := cam.FOV.Y / 2.0

	if zDepth > 0 {
		projX := (viewCenter.X * cam.FOV.X) / zDepth
		projY := (viewCenter.Y * cam.FOV.Y) / zDepth

		projRadius := (radius * cam.FOV.X) / (zDepth - radius)
		if projRadius < 0 {
			projRadius = 0
		}

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

// MoveUp moves the camera up in WORLD space (not local)
func (cam *Camera) MoveUp(distance float64) {
	// World-space up (Y axis)
	cam.Transform.Translate(0, distance, 0)
}

// MoveUpLocal moves the camera up in its LOCAL space
func (cam *Camera) MoveUpLocal(distance float64) {
	up := cam.Transform.GetUpVector()
	cam.Transform.Translate(up.X*distance, up.Y*distance, up.Z*distance)
}

// RotateYaw rotates the camera around the WORLD Y axis (left/right turn)
func (cam *Camera) RotateYaw(angle float64) {
	// Rotate around world Y axis
	worldYAxis := Point{X: 0, Y: 1, Z: 0}
	cam.Transform.RotateAxisAngle(worldYAxis, angle)
}

// RotatePitch rotates the camera around its LOCAL X axis (up/down look)
func (cam *Camera) RotatePitch(angle float64) {
	// Rotate around local right vector
	right := cam.Transform.GetRightVector()
	cam.Transform.RotateAxisAngle(right, angle)
}

// RotateRoll rotates the camera around its LOCAL Z axis (tilt)
func (cam *Camera) RotateRoll(angle float64) {
	// Rotate around local forward vector
	forward := cam.Transform.GetForwardVector()
	cam.Transform.RotateAxisAngle(forward, angle)
}
