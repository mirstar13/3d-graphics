package main

import "math"

// Transform represents position, rotation, and scale in 3D space with caching
type Transform struct {
	Position Point      // Local position
	Rotation Quaternion // Rotation as quaternion (no gimbal lock!)
	Scale    Point      // Scale factors (usually 1, 1, 1)
	Parent   *Transform

	// Cache for performance
	worldMatrix        Matrix4x4
	localMatrix        Matrix4x4
	inverseMatrix      Matrix4x4
	worldMatrixDirty   bool
	localMatrixDirty   bool
	inverseMatrixDirty bool
}

// NewTransform creates a new transform at the origin
func NewTransform() *Transform {
	return &Transform{
		Position:           Point{X: 0, Y: 0, Z: 0},
		Rotation:           IdentityQuaternion(),
		Scale:              Point{X: 1, Y: 1, Z: 1},
		Parent:             nil,
		worldMatrix:        IdentityMatrix(),
		localMatrix:        IdentityMatrix(),
		inverseMatrix:      IdentityMatrix(),
		worldMatrixDirty:   true,
		localMatrixDirty:   true,
		inverseMatrixDirty: true,
	}
}

// NewTransformAt creates a transform at a specific position
func NewTransformAt(x, y, z float64) *Transform {
	t := NewTransform()
	t.SetPosition(x, y, z)
	return t
}

// MarkDirty marks all cached matrices as needing recalculation
// and propagates to children
func (t *Transform) MarkDirty() {
	if !t.worldMatrixDirty { // Only propagate if we weren't already dirty
		t.worldMatrixDirty = true
		t.localMatrixDirty = true
		t.inverseMatrixDirty = true

		// Propagate to all children recursively
		t.markChildrenDirty()
	}
}

func (t *Transform) markChildrenDirty() {
	// This will be called by SceneNode to mark children
	// We can't do it here because Transform doesn't know about SceneNode
}

// SetPosition sets the local position
func (t *Transform) SetPosition(x, y, z float64) {
	t.Position.X = x
	t.Position.Y = y
	t.Position.Z = z
	t.MarkDirty()
}

// SetRotation sets the rotation (pitch, yaw, roll) - converts to quaternion
func (t *Transform) SetRotation(pitch, yaw, roll float64) {
	t.Rotation = QuaternionFromEuler(pitch, yaw, roll)
	t.MarkDirty()
}

// SetRotationQuaternion sets rotation directly from quaternion
func (t *Transform) SetRotationQuaternion(q Quaternion) {
	t.Rotation = q.Normalize()
	t.MarkDirty()
}

// SetScale sets the scale
func (t *Transform) SetScale(x, y, z float64) {
	t.Scale.X = x
	t.Scale.Y = y
	t.Scale.Z = z
	t.MarkDirty()
}

// Translate moves the transform by a delta in local space
func (t *Transform) Translate(dx, dy, dz float64) {
	t.Position.X += dx
	t.Position.Y += dy
	t.Position.Z += dz
	t.MarkDirty()
}

// Rotate rotates the transform by delta angles (converts to quaternion internally)
func (t *Transform) Rotate(dpitch, dyaw, droll float64) {
	deltaQuat := QuaternionFromEuler(dpitch, dyaw, droll)
	t.Rotation = t.Rotation.Multiply(deltaQuat).Normalize()
	t.MarkDirty()
}

// RotateAxisAngle rotates around an arbitrary axis
func (t *Transform) RotateAxisAngle(axis Point, angle float64) {
	deltaQuat := QuaternionFromAxisAngle(axis, angle)
	t.Rotation = t.Rotation.Multiply(deltaQuat).Normalize()
	t.MarkDirty()
}

// GetWorldMatrix returns the cached world transformation matrix
func (t *Transform) GetWorldMatrix() Matrix4x4 {
	if t.worldMatrixDirty {
		localMat := t.GetLocalMatrix()
		if t.Parent != nil {
			// Check if parent is dirty and force update
			parentMat := t.Parent.GetWorldMatrix()
			t.worldMatrix = parentMat.Multiply(localMat)
		} else {
			t.worldMatrix = localMat
		}
		t.worldMatrixDirty = false
		// Don't clear inverse here - let it be lazy
	}
	return t.worldMatrix
}

// GetLocalMatrix returns the cached local transformation matrix
func (t *Transform) GetLocalMatrix() Matrix4x4 {
	if t.localMatrixDirty {
		t.localMatrix = ComposeMatrix(t.Position, t.Rotation, t.Scale)
		t.localMatrixDirty = false
	}
	return t.localMatrix
}

// GetInverseMatrix returns the cached inverse world matrix
func (t *Transform) GetInverseMatrix() Matrix4x4 {
	if t.inverseMatrixDirty {
		t.inverseMatrix = t.GetWorldMatrix().Invert()
		t.inverseMatrixDirty = false
	}
	return t.inverseMatrix
}

// TransformPoint transforms a point from local space to world space (CACHED)
func (t *Transform) TransformPoint(p Point) Point {
	return t.GetWorldMatrix().TransformPoint(p)
}

// TransformDirection transforms a direction vector (CACHED)
func (t *Transform) TransformDirection(d Point) Point {
	return t.GetWorldMatrix().TransformDirection(d)
}

// InverseTransformPoint transforms a world-space point to local space (CACHED)
func (t *Transform) InverseTransformPoint(worldPoint Point) Point {
	return t.GetInverseMatrix().TransformPoint(worldPoint)
}

// GetWorldPosition returns the world-space position (CACHED)
func (t *Transform) GetWorldPosition() Point {
	mat := t.GetWorldMatrix()
	return Point{X: mat.M[3], Y: mat.M[7], Z: mat.M[11]}
}

// GetWorldRotation returns the world-space rotation as Euler angles
func (t *Transform) GetWorldRotation() Point {
	// Get world rotation quaternion
	worldQuat := t.Rotation
	if t.Parent != nil {
		// Combine with parent rotation
		parentRot := t.Parent.GetWorldRotationQuaternion()
		worldQuat = parentRot.Multiply(t.Rotation).Normalize()
	}

	pitch, yaw, roll := worldQuat.ToEuler()
	return Point{X: pitch, Y: yaw, Z: roll}
}

// GetWorldRotationQuaternion returns world rotation as quaternion
func (t *Transform) GetWorldRotationQuaternion() Quaternion {
	if t.Parent != nil {
		return t.Parent.GetWorldRotationQuaternion().Multiply(t.Rotation).Normalize()
	}
	return t.Rotation
}

// GetForwardVector returns the forward direction in world space (CACHED)
func (t *Transform) GetForwardVector() Point {
	return t.TransformDirection(Point{X: 0, Y: 0, Z: 1})
}

// GetRightVector returns the right direction in world space (CACHED)
func (t *Transform) GetRightVector() Point {
	return t.TransformDirection(Point{X: 1, Y: 0, Z: 0})
}

// GetUpVector returns the up direction in world space (CACHED)
func (t *Transform) GetUpVector() Point {
	return t.TransformDirection(Point{X: 0, Y: 1, Z: 0})
}

// LookAt makes the transform look at a target position
func (t *Transform) LookAt(target Point) {
	worldPos := t.GetWorldPosition()

	// Direction to target
	dx := target.X - worldPos.X
	dy := target.Y - worldPos.Y
	dz := target.Z - worldPos.Z

	// Normalize
	length := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if length < 1e-10 {
		return
	}
	dx /= length
	dy /= length
	dz /= length

	// Calculate yaw and pitch
	yaw := math.Atan2(dx, dz)
	distXZ := math.Sqrt(dx*dx + dz*dz)
	pitch := -math.Atan2(dy, distXZ)

	t.SetRotation(pitch, yaw, 0)
}

// SetParent sets the parent transform and marks dirty
func (t *Transform) SetParent(parent *Transform) {
	t.Parent = parent
	t.MarkDirty()
}

// Legacy compatibility methods (for backward compatibility)
func (t *Transform) GetRotation() (pitch, yaw, roll float64) {
	return t.Rotation.ToEuler()
}
