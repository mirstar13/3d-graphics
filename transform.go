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
	}
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

// Rotate rotates the transform by delta angles
func (t *Transform) Rotate(dpitch, dyaw, droll float64) {
	deltaQuat := QuaternionFromEuler(dpitch, dyaw, droll)
	// Left-multiply for local-space rotation
	t.Rotation = deltaQuat.Multiply(t.Rotation).Normalize()
	t.MarkDirty()
}

// RotateAxisAngle rotates around an arbitrary axis
func (t *Transform) RotateAxisAngle(axis Point, angle float64) {
	// Normalize axis
	length := math.Sqrt(axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z)
	if length < 1e-10 {
		return // Invalid axis
	}
	normalizedAxis := Point{
		X: axis.X / length,
		Y: axis.Y / length,
		Z: axis.Z / length,
	}

	// Create quaternion from axis-angle
	deltaQuat := QuaternionFromAxisAngle(normalizedAxis, angle)

	// Multiply on the LEFT for local-space rotation
	// q_new = q_delta * q_current
	t.Rotation = deltaQuat.Multiply(t.Rotation).Normalize()

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

	// Build a look-at rotation matrix
	// Forward = direction to target (we want +Z to point at target)
	forward := Point{X: dx, Y: dy, Z: dz}

	// Up vector (world up)
	worldUp := Point{X: 0, Y: 1, Z: 0}

	// Right = forward × up
	rightX, rightY, rightZ := crossProduct(forward.X, forward.Y, forward.Z, worldUp.X, worldUp.Y, worldUp.Z)
	rightLen := math.Sqrt(rightX*rightX + rightY*rightY + rightZ*rightZ)

	if rightLen < 1e-10 {
		// Forward and up are parallel, choose arbitrary right
		rightX, rightY, rightZ = 1, 0, 0
	} else {
		rightX /= rightLen
		rightY /= rightLen
		rightZ /= rightLen
	}

	// Up = right × forward (to ensure orthogonality)
	upX, upY, upZ := crossProduct(rightX, rightY, rightZ, forward.X, forward.Y, forward.Z)

	// Build rotation matrix from basis vectors
	// Note: Our forward is +Z, right is +X, up is +Y
	rotMatrix := Matrix4x4{M: [16]float64{
		rightX, upX, forward.X, 0,
		rightY, upY, forward.Y, 0,
		rightZ, upZ, forward.Z, 0,
		0, 0, 0, 1,
	}}

	// Convert matrix to quaternion
	t.Rotation = MatrixToQuaternion(rotMatrix)
	t.MarkDirty()
}

// MatrixToQuaternion converts a rotation matrix to a quaternion
func MatrixToQuaternion(m Matrix4x4) Quaternion {
	// Extract rotation part
	m00, m01, m02 := m.M[0], m.M[1], m.M[2]
	m10, m11, m12 := m.M[4], m.M[5], m.M[6]
	m20, m21, m22 := m.M[8], m.M[9], m.M[10]

	trace := m00 + m11 + m22

	var q Quaternion

	if trace > 0 {
		s := math.Sqrt(trace+1.0) * 2
		q.W = 0.25 * s
		q.X = (m21 - m12) / s
		q.Y = (m02 - m20) / s
		q.Z = (m10 - m01) / s
	} else if m00 > m11 && m00 > m22 {
		s := math.Sqrt(1.0+m00-m11-m22) * 2
		q.W = (m21 - m12) / s
		q.X = 0.25 * s
		q.Y = (m01 + m10) / s
		q.Z = (m02 + m20) / s
	} else if m11 > m22 {
		s := math.Sqrt(1.0+m11-m00-m22) * 2
		q.W = (m02 - m20) / s
		q.X = (m01 + m10) / s
		q.Y = 0.25 * s
		q.Z = (m12 + m21) / s
	} else {
		s := math.Sqrt(1.0+m22-m00-m11) * 2
		q.W = (m10 - m01) / s
		q.X = (m02 + m20) / s
		q.Y = (m12 + m21) / s
		q.Z = 0.25 * s
	}

	return q.Normalize()
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
