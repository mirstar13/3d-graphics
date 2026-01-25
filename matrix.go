package main

import "math"

// Matrix4x4 represents a 4x4 transformation matrix
type Matrix4x4 struct {
	M [16]float64 // Column-major order
}

// Identity returns an identity matrix
func IdentityMatrix() Matrix4x4 {
	return Matrix4x4{M: [16]float64{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}}
}

// Multiply multiplies two matrices
func (m *Matrix4x4) Multiply(other Matrix4x4) Matrix4x4 {
	var result Matrix4x4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			sum := 0.0
			for k := 0; k < 4; k++ {
				sum += m.M[i*4+k] * other.M[k*4+j]
			}
			// Round to prevent floating point drift
			if math.Abs(sum) < 1e-10 {
				sum = 0.0
			}
			result.M[i*4+j] = sum
		}
	}
	return result
}

// TransformPoint transforms a point by this matrix
func (m *Matrix4x4) TransformPoint(p Point) Point {
	x := m.M[0]*p.X + m.M[1]*p.Y + m.M[2]*p.Z + m.M[3]
	y := m.M[4]*p.X + m.M[5]*p.Y + m.M[6]*p.Z + m.M[7]
	z := m.M[8]*p.X + m.M[9]*p.Y + m.M[10]*p.Z + m.M[11]
	w := m.M[12]*p.X + m.M[13]*p.Y + m.M[14]*p.Z + m.M[15]

	if math.Abs(w) > 1e-10 {
		return Point{X: x / w, Y: y / w, Z: z / w}
	}
	return Point{X: x, Y: y, Z: z}
}

// TransformPointAffine transforms a point by this matrix, assuming it is an affine matrix (bottom row is 0,0,0,1).
// This avoids calculating W and division, making it faster for model-to-world transformations.
func (m *Matrix4x4) TransformPointAffine(p Point) Point {
	return Point{
		X: m.M[0]*p.X + m.M[1]*p.Y + m.M[2]*p.Z + m.M[3],
		Y: m.M[4]*p.X + m.M[5]*p.Y + m.M[6]*p.Z + m.M[7],
		Z: m.M[8]*p.X + m.M[9]*p.Y + m.M[10]*p.Z + m.M[11],
	}
}

// MultiplyPoint is an alias for TransformPoint for compatibility
func (m *Matrix4x4) MultiplyPoint(p Point) Point {
	return m.TransformPoint(p)
}

// TransformDirection transforms a direction vector (ignores translation)
func (m *Matrix4x4) TransformDirection(d Point) Point {
	x := m.M[0]*d.X + m.M[1]*d.Y + m.M[2]*d.Z
	y := m.M[4]*d.X + m.M[5]*d.Y + m.M[6]*d.Z
	z := m.M[8]*d.X + m.M[9]*d.Y + m.M[10]*d.Z
	return Point{X: x, Y: y, Z: z}
}

// ComposeMatrix creates a transformation matrix from position, rotation, scale
func ComposeMatrix(pos Point, rot Quaternion, scale Point) Matrix4x4 {
	// Convert quaternion to rotation matrix
	rotMatrix := rot.ToMatrix()

	// Apply scale
	var result Matrix4x4
	result.M[0] = rotMatrix.M[0] * scale.X
	result.M[1] = rotMatrix.M[1] * scale.X
	result.M[2] = rotMatrix.M[2] * scale.X
	result.M[3] = pos.X

	result.M[4] = rotMatrix.M[4] * scale.Y
	result.M[5] = rotMatrix.M[5] * scale.Y
	result.M[6] = rotMatrix.M[6] * scale.Y
	result.M[7] = pos.Y

	result.M[8] = rotMatrix.M[8] * scale.Z
	result.M[9] = rotMatrix.M[9] * scale.Z
	result.M[10] = rotMatrix.M[10] * scale.Z
	result.M[11] = pos.Z

	result.M[12] = 0
	result.M[13] = 0
	result.M[14] = 0
	result.M[15] = 1

	return result
}

// Invert returns the inverse matrix
func (m *Matrix4x4) Invert() Matrix4x4 {
	// Using adjugate method - full implementation
	var inv Matrix4x4
	inv.M[0] = m.M[5]*m.M[10]*m.M[15] - m.M[5]*m.M[11]*m.M[14] - m.M[9]*m.M[6]*m.M[15] +
		m.M[9]*m.M[7]*m.M[14] + m.M[13]*m.M[6]*m.M[11] - m.M[13]*m.M[7]*m.M[10]

	inv.M[4] = -m.M[4]*m.M[10]*m.M[15] + m.M[4]*m.M[11]*m.M[14] + m.M[8]*m.M[6]*m.M[15] -
		m.M[8]*m.M[7]*m.M[14] - m.M[12]*m.M[6]*m.M[11] + m.M[12]*m.M[7]*m.M[10]

	inv.M[8] = m.M[4]*m.M[9]*m.M[15] - m.M[4]*m.M[11]*m.M[13] - m.M[8]*m.M[5]*m.M[15] +
		m.M[8]*m.M[7]*m.M[13] + m.M[12]*m.M[5]*m.M[11] - m.M[12]*m.M[7]*m.M[9]

	inv.M[12] = -m.M[4]*m.M[9]*m.M[14] + m.M[4]*m.M[10]*m.M[13] + m.M[8]*m.M[5]*m.M[14] -
		m.M[8]*m.M[6]*m.M[13] - m.M[12]*m.M[5]*m.M[10] + m.M[12]*m.M[6]*m.M[9]

	inv.M[1] = -m.M[1]*m.M[10]*m.M[15] + m.M[1]*m.M[11]*m.M[14] + m.M[9]*m.M[2]*m.M[15] -
		m.M[9]*m.M[3]*m.M[14] - m.M[13]*m.M[2]*m.M[11] + m.M[13]*m.M[3]*m.M[10]

	inv.M[5] = m.M[0]*m.M[10]*m.M[15] - m.M[0]*m.M[11]*m.M[14] - m.M[8]*m.M[2]*m.M[15] +
		m.M[8]*m.M[3]*m.M[14] + m.M[12]*m.M[2]*m.M[11] - m.M[12]*m.M[3]*m.M[10]

	inv.M[9] = -m.M[0]*m.M[9]*m.M[15] + m.M[0]*m.M[11]*m.M[13] + m.M[8]*m.M[1]*m.M[15] -
		m.M[8]*m.M[3]*m.M[13] - m.M[12]*m.M[1]*m.M[11] + m.M[12]*m.M[3]*m.M[9]

	inv.M[13] = m.M[0]*m.M[9]*m.M[14] - m.M[0]*m.M[10]*m.M[13] - m.M[8]*m.M[1]*m.M[14] +
		m.M[8]*m.M[2]*m.M[13] + m.M[12]*m.M[1]*m.M[10] - m.M[12]*m.M[2]*m.M[9]

	inv.M[2] = m.M[1]*m.M[6]*m.M[15] - m.M[1]*m.M[7]*m.M[14] - m.M[5]*m.M[2]*m.M[15] +
		m.M[5]*m.M[3]*m.M[14] + m.M[13]*m.M[2]*m.M[7] - m.M[13]*m.M[3]*m.M[6]

	inv.M[6] = -m.M[0]*m.M[6]*m.M[15] + m.M[0]*m.M[7]*m.M[14] + m.M[4]*m.M[2]*m.M[15] -
		m.M[4]*m.M[3]*m.M[14] - m.M[12]*m.M[2]*m.M[7] + m.M[12]*m.M[3]*m.M[6]

	inv.M[10] = m.M[0]*m.M[5]*m.M[15] - m.M[0]*m.M[7]*m.M[13] - m.M[4]*m.M[1]*m.M[15] +
		m.M[4]*m.M[3]*m.M[13] + m.M[12]*m.M[1]*m.M[7] - m.M[12]*m.M[3]*m.M[5]

	inv.M[14] = -m.M[0]*m.M[5]*m.M[14] + m.M[0]*m.M[6]*m.M[13] + m.M[4]*m.M[1]*m.M[14] -
		m.M[4]*m.M[2]*m.M[13] - m.M[12]*m.M[1]*m.M[6] + m.M[12]*m.M[2]*m.M[5]

	inv.M[3] = -m.M[1]*m.M[6]*m.M[11] + m.M[1]*m.M[7]*m.M[10] + m.M[5]*m.M[2]*m.M[11] -
		m.M[5]*m.M[3]*m.M[10] - m.M[9]*m.M[2]*m.M[7] + m.M[9]*m.M[3]*m.M[6]

	inv.M[7] = m.M[0]*m.M[6]*m.M[11] - m.M[0]*m.M[7]*m.M[10] - m.M[4]*m.M[2]*m.M[11] +
		m.M[4]*m.M[3]*m.M[10] + m.M[8]*m.M[2]*m.M[7] - m.M[8]*m.M[3]*m.M[6]

	inv.M[11] = -m.M[0]*m.M[5]*m.M[11] + m.M[0]*m.M[7]*m.M[9] + m.M[4]*m.M[1]*m.M[11] -
		m.M[4]*m.M[3]*m.M[9] - m.M[8]*m.M[1]*m.M[7] + m.M[8]*m.M[3]*m.M[5]

	inv.M[15] = m.M[0]*m.M[5]*m.M[10] - m.M[0]*m.M[6]*m.M[9] - m.M[4]*m.M[1]*m.M[10] +
		m.M[4]*m.M[2]*m.M[9] + m.M[8]*m.M[1]*m.M[6] - m.M[8]*m.M[2]*m.M[5]

	det := m.M[0]*inv.M[0] + m.M[1]*inv.M[4] + m.M[2]*inv.M[8] + m.M[3]*inv.M[12]

	if math.Abs(det) < 1e-10 {
		return IdentityMatrix()
	}

	invDet := 1.0 / det
	for i := 0; i < 16; i++ {
		inv.M[i] *= invDet
	}

	return inv
}

// CreateOrthographicMatrix creates an orthographic projection matrix
func CreateOrthographicMatrix(left, right, bottom, top, near, far float64) Matrix4x4 {
	mat := Matrix4x4{}

	// Scale
	mat.M[0] = 2.0 / (right - left)
	mat.M[5] = 2.0 / (top - bottom)
	mat.M[10] = -2.0 / (far - near)

	// Translation
	mat.M[3] = -(right + left) / (right - left)
	mat.M[7] = -(top + bottom) / (top - bottom)
	mat.M[11] = -(far + near) / (far - near)

	mat.M[15] = 1.0

	return mat
}
