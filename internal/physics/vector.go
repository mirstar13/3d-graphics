package physics

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

type vec = engine.Point
type quat = engine.Quaternion

func add(a, b vec) vec         { return vec{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
func sub(a, b vec) vec         { return vec{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }
func mul(a vec, s float64) vec { return vec{X: a.X * s, Y: a.Y * s, Z: a.Z * s} }
func dot(a, b vec) float64     { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func cross(a, b vec) vec {
	return vec{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
func length(a vec) float64 { return math.Sqrt(dot(a, a)) }
func unit(a vec) vec {
	if n := length(a); n > 1e-12 {
		return mul(a, 1/n)
	}
	return vec{X: 1}
}
func component(a vec, i int) float64 {
	switch i {
	case 0:
		return a.X
	case 1:
		return a.Y
	default:
		return a.Z
	}
}
func axis(i int) vec {
	switch i {
	case 0:
		return vec{X: 1}
	case 1:
		return vec{Y: 1}
	default:
		return vec{Z: 1}
	}
}
func conjugate(q quat) quat           { return quat{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z} }
func inverseRotate(q quat, v vec) vec { return conjugate(q).RotateVector(v) }
func finite(v float64) bool           { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func validRotation(q quat) bool {
	norm := q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
	return finite(norm) && norm > 1e-20
}
func rotateBy(q quat, angular vec, dt float64) quat {
	angle := length(angular) * dt
	if angle < 1e-14 {
		return q
	}
	return engine.QuaternionFromAxisAngle(unit(angular), angle).Multiply(q)
}
func angularBetween(from, to quat, dt float64) vec {
	q := to.Multiply(conjugate(from))
	if q.W < 0 {
		q = quat{W: -q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
	}
	v := vec{X: q.X, Y: q.Y, Z: q.Z}
	s := length(v)
	if s < 1e-12 {
		return vec{}
	}
	return mul(v, 2*math.Atan2(s, q.W)/(s*dt))
}
