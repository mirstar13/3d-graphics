package engine

import (
	"math"
)

// Quaternion represents a rotation using quaternions (avoids gimbal lock)
type Quaternion struct {
	W, X, Y, Z float64
}

// IdentityQuaternion returns a quaternion representing no rotation
func IdentityQuaternion() Quaternion {
	return Quaternion{W: 1, X: 0, Y: 0, Z: 0}
}

// QuaternionFromEuler composes yaw about Y, pitch about X, and roll about Z.
// Angles are radians; the resulting matrix is Ry(yaw) * Rx(pitch) * Rz(roll).
func QuaternionFromEuler(pitch, yaw, roll float64) Quaternion {
	p := QuaternionFromAxisAngle(Point{X: 1}, pitch)
	y := QuaternionFromAxisAngle(Point{Y: 1}, yaw)
	r := QuaternionFromAxisAngle(Point{Z: 1}, roll)
	return y.Multiply(p).Multiply(r).Normalize()
}

// ToEuler returns pitch (X), yaw (Y), and roll (Z) in radians.
// At gimbal lock roll is chosen as zero, preserving the combined orientation.
func (q Quaternion) ToEuler() (pitch, yaw, roll float64) {
	matrix := q.Normalize().ToMatrix()
	pitch = math.Asin(Clamp(-matrix.M[6], -1, 1))
	if math.Abs(math.Cos(pitch)) < 1e-10 {
		return pitch, math.Atan2(-matrix.M[8], matrix.M[0]), 0
	}
	yaw = math.Atan2(matrix.M[2], matrix.M[10])
	roll = math.Atan2(matrix.M[4], matrix.M[5])
	return
}

// Normalize normalizes the quaternion
func (q Quaternion) Normalize() Quaternion {
	length := math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if length < 1e-10 {
		return IdentityQuaternion()
	}

	invLen := 1.0 / length
	return Quaternion{
		W: q.W * invLen,
		X: q.X * invLen,
		Y: q.Y * invLen,
		Z: q.Z * invLen,
	}
}

// Multiply multiplies two quaternions (combines rotations)
func (q Quaternion) Multiply(other Quaternion) Quaternion {
	result := Quaternion{
		W: q.W*other.W - q.X*other.X - q.Y*other.Y - q.Z*other.Z,
		X: q.W*other.X + q.X*other.W + q.Y*other.Z - q.Z*other.Y,
		Y: q.W*other.Y - q.X*other.Z + q.Y*other.W + q.Z*other.X,
		Z: q.W*other.Z + q.X*other.Y - q.Y*other.X + q.Z*other.W,
	}

	return result.Normalize()
}

// Slerp performs spherical linear interpolation between two quaternions
func (q Quaternion) Slerp(other Quaternion, t float64) Quaternion {
	// Clamp t
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Compute dot product
	dot := q.W*other.W + q.X*other.X + q.Y*other.Y + q.Z*other.Z

	// If dot < 0, negate one quaternion to take shorter path
	if dot < 0 {
		other = Quaternion{W: -other.W, X: -other.X, Y: -other.Y, Z: -other.Z}
		dot = -dot
	}

	// If quaternions are very close, use linear interpolation
	if dot > 0.9995 {
		return Quaternion{
			W: q.W + t*(other.W-q.W),
			X: q.X + t*(other.X-q.X),
			Y: q.Y + t*(other.Y-q.Y),
			Z: q.Z + t*(other.Z-q.Z),
		}.Normalize()
	}

	// Perform slerp
	theta := math.Acos(dot)
	sinTheta := math.Sin(theta)
	wa := math.Sin((1-t)*theta) / sinTheta
	wb := math.Sin(t*theta) / sinTheta

	return Quaternion{
		W: q.W*wa + other.W*wb,
		X: q.X*wa + other.X*wb,
		Y: q.Y*wa + other.Y*wb,
		Z: q.Z*wa + other.Z*wb,
	}
}

// ToMatrix converts quaternion to rotation matrix
func (q Quaternion) ToMatrix() Matrix4x4 {
	xx := q.X * q.X
	yy := q.Y * q.Y
	zz := q.Z * q.Z
	xy := q.X * q.Y
	xz := q.X * q.Z
	yz := q.Y * q.Z
	wx := q.W * q.X
	wy := q.W * q.Y
	wz := q.W * q.Z

	return Matrix4x4{M: [16]float64{
		1 - 2*(yy+zz), 2 * (xy - wz), 2 * (xz + wy), 0,
		2 * (xy + wz), 1 - 2*(xx+zz), 2 * (yz - wx), 0,
		2 * (xz - wy), 2 * (yz + wx), 1 - 2*(xx+yy), 0,
		0, 0, 0, 1,
	}}
}

// RotateVector rotates a vector by this quaternion
func (q Quaternion) RotateVector(v Point) Point {
	// Convert vector to quaternion
	vecQuat := Quaternion{W: 0, X: v.X, Y: v.Y, Z: v.Z}

	// Compute q * v * q^-1
	qConj := Quaternion{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
	result := q.Multiply(vecQuat).Multiply(qConj)

	return Point{X: result.X, Y: result.Y, Z: result.Z}
}

// QuaternionFromAxisAngle creates a quaternion from axis-angle representation
func QuaternionFromAxisAngle(axis Point, angle float64) Quaternion {
	// Normalize axis
	length := math.Sqrt(axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z)
	if length < 1e-10 {
		return IdentityQuaternion()
	}
	axis.X /= length
	axis.Y /= length
	axis.Z /= length

	halfAngle := angle * 0.5
	s := math.Sin(halfAngle)

	return Quaternion{
		W: math.Cos(halfAngle),
		X: axis.X * s,
		Y: axis.Y * s,
		Z: axis.Z * s,
	}
}
