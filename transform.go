package main

import "math"

// Transform represents position, rotation, and scale in 3D space
type Transform struct {
	Position Point // Local position
	Rotation Point // Euler angles (X=pitch, Y=yaw, Z=roll) in radians
	Scale    Point // Scale factors (usually 1, 1, 1)
	Parent   *Transform
}

// NewTransform creates a new transform at the origin
func NewTransform() *Transform {
	return &Transform{
		Position: Point{X: 0, Y: 0, Z: 0},
		Rotation: Point{X: 0, Y: 0, Z: 0},
		Scale:    Point{X: 1, Y: 1, Z: 1},
		Parent:   nil,
	}
}

// NewTransformAt creates a transform at a specific position
func NewTransformAt(x, y, z float64) *Transform {
	return &Transform{
		Position: Point{X: x, Y: y, Z: z},
		Rotation: Point{X: 0, Y: 0, Z: 0},
		Scale:    Point{X: 1, Y: 1, Z: 1},
		Parent:   nil,
	}
}

// SetPosition sets the local position
func (t *Transform) SetPosition(x, y, z float64) {
	t.Position.X = x
	t.Position.Y = y
	t.Position.Z = z
}

// SetRotation sets the rotation (pitch, yaw, roll)
func (t *Transform) SetRotation(pitch, yaw, roll float64) {
	t.Rotation.X = pitch
	t.Rotation.Y = yaw
	t.Rotation.Z = roll
}

// SetScale sets the scale
func (t *Transform) SetScale(x, y, z float64) {
	t.Scale.X = x
	t.Scale.Y = y
	t.Scale.Z = z
}

// Translate moves the transform by a delta in local space
func (t *Transform) Translate(dx, dy, dz float64) {
	t.Position.X += dx
	t.Position.Y += dy
	t.Position.Z += dz
}

// Rotate rotates the transform by delta angles
func (t *Transform) Rotate(dpitch, dyaw, droll float64) {
	t.Rotation.X += dpitch
	t.Rotation.Y += dyaw
	t.Rotation.Z += droll
}

// GetWorldPosition returns the world-space position
func (t *Transform) GetWorldPosition() Point {
	if t.Parent == nil {
		return t.Position
	}

	// Transform local position by parent's transform
	return t.Parent.TransformPoint(t.Position)
}

// GetWorldRotation returns the world-space rotation
func (t *Transform) GetWorldRotation() Point {
	if t.Parent == nil {
		return t.Rotation
	}

	// Combine with parent rotation
	parentRot := t.Parent.GetWorldRotation()
	return Point{
		X: t.Rotation.X + parentRot.X,
		Y: t.Rotation.Y + parentRot.Y,
		Z: t.Rotation.Z + parentRot.Z,
	}
}

// TransformPoint transforms a point from local space to world space
func (t *Transform) TransformPoint(p Point) Point {
	// Apply scale
	x := p.X * t.Scale.X
	y := p.Y * t.Scale.Y
	z := p.Z * t.Scale.Z

	// Apply rotation (yaw -> pitch -> roll order)
	// Yaw (Y-axis rotation)
	cosYaw := math.Cos(t.Rotation.Y)
	sinYaw := math.Sin(t.Rotation.Y)
	xRot := x*cosYaw - z*sinYaw
	zRot := x*sinYaw + z*cosYaw
	x = xRot
	z = zRot

	// Pitch (X-axis rotation)
	cosPitch := math.Cos(t.Rotation.X)
	sinPitch := math.Sin(t.Rotation.X)
	yRot := y*cosPitch - z*sinPitch
	zRot = y*sinPitch + z*cosPitch
	y = yRot
	z = zRot

	// Roll (Z-axis rotation)
	cosRoll := math.Cos(t.Rotation.Z)
	sinRoll := math.Sin(t.Rotation.Z)
	xRot = x*cosRoll - y*sinRoll
	yRot = x*sinRoll + y*cosRoll
	x = xRot
	y = yRot

	// Apply translation
	x += t.Position.X
	y += t.Position.Y
	z += t.Position.Z

	// If we have a parent, transform by parent as well
	if t.Parent != nil {
		return t.Parent.TransformPoint(Point{X: x, Y: y, Z: z})
	}

	return Point{X: x, Y: y, Z: z}
}

// TransformDirection transforms a direction vector (ignores position and scale)
func (t *Transform) TransformDirection(dir Point) Point {
	x := dir.X
	y := dir.Y
	z := dir.Z

	// Apply rotation only
	// Yaw (Y-axis rotation)
	cosYaw := math.Cos(t.Rotation.Y)
	sinYaw := math.Sin(t.Rotation.Y)
	xRot := x*cosYaw - z*sinYaw
	zRot := x*sinYaw + z*cosYaw
	x = xRot
	z = zRot

	// Pitch (X-axis rotation)
	cosPitch := math.Cos(t.Rotation.X)
	sinPitch := math.Sin(t.Rotation.X)
	yRot := y*cosPitch - z*sinPitch
	zRot = y*sinPitch + z*cosPitch
	y = yRot
	z = zRot

	// Roll (Z-axis rotation)
	cosRoll := math.Cos(t.Rotation.Z)
	sinRoll := math.Sin(t.Rotation.Z)
	xRot = x*cosRoll - y*sinRoll
	yRot = x*sinRoll + y*cosRoll
	x = xRot
	y = yRot

	// If we have a parent, transform by parent's rotation as well
	if t.Parent != nil {
		return t.Parent.TransformDirection(Point{X: x, Y: y, Z: z})
	}

	return Point{X: x, Y: y, Z: z}
}

// GetForwardVector returns the forward direction in world space
func (t *Transform) GetForwardVector() Point {
	// Forward is (0, 0, 1) in local space
	return t.TransformDirection(Point{X: 0, Y: 0, Z: 1})
}

// GetRightVector returns the right direction in world space
func (t *Transform) GetRightVector() Point {
	// Right is (1, 0, 0) in local space
	return t.TransformDirection(Point{X: 1, Y: 0, Z: 0})
}

// GetUpVector returns the up direction in world space
func (t *Transform) GetUpVector() Point {
	// Up is (0, 1, 0) in local space
	return t.TransformDirection(Point{X: 0, Y: 1, Z: 0})
}

// LookAt makes the transform look at a target position
func (t *Transform) LookAt(target Point) {
	worldPos := t.GetWorldPosition()

	// Direction to target
	dx := target.X - worldPos.X
	dy := target.Y - worldPos.Y
	dz := target.Z - worldPos.Z

	// Calculate yaw (rotation around Y)
	t.Rotation.Y = math.Atan2(dx, dz)

	// Calculate pitch (rotation around X)
	distXZ := math.Sqrt(dx*dx + dz*dz)
	t.Rotation.X = -math.Atan2(dy, distXZ)
}

// SetParent sets the parent transform
func (t *Transform) SetParent(parent *Transform) {
	t.Parent = parent
}

// InverseTransformPoint transforms a world-space point to local space
func (t *Transform) InverseTransformPoint(worldPoint Point) Point {
	// If we have a parent, first transform to parent's local space
	p := worldPoint
	if t.Parent != nil {
		p = t.Parent.InverseTransformPoint(worldPoint)
	}

	// Remove translation
	x := p.X - t.Position.X
	y := p.Y - t.Position.Y
	z := p.Z - t.Position.Z

	// Remove rotation (in reverse order: roll -> pitch -> yaw)
	// Roll (Z-axis rotation) - inverse
	cosRoll := math.Cos(-t.Rotation.Z)
	sinRoll := math.Sin(-t.Rotation.Z)
	xRot := x*cosRoll - y*sinRoll
	yRot := x*sinRoll + y*cosRoll
	x = xRot
	y = yRot

	// Pitch (X-axis rotation) - inverse
	cosPitch := math.Cos(-t.Rotation.X)
	sinPitch := math.Sin(-t.Rotation.X)
	yRot = y*cosPitch - z*sinPitch
	zRot := y*sinPitch + z*cosPitch
	y = yRot
	z = zRot

	// Yaw (Y-axis rotation) - inverse
	cosYaw := math.Cos(-t.Rotation.Y)
	sinYaw := math.Sin(-t.Rotation.Y)
	xRot = x*cosYaw - z*sinYaw
	zRot = x*sinYaw + z*cosYaw
	x = xRot
	z = zRot

	// Remove scale
	x /= t.Scale.X
	y /= t.Scale.Y
	z /= t.Scale.Z

	return Point{X: x, Y: y, Z: z}
}
