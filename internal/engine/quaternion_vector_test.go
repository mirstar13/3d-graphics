package engine

import (
	"math"
	"testing"
)

func TestQuaternionRotationPreservesVectorLength(t *testing.T) {
	q := QuaternionFromEuler(.2, .4, .7)
	v := Point{X: 3, Y: -4, Z: 8}
	got := q.RotateVector(v)
	m := q.ToMatrix()
	want := m.TransformDirection(v)
	if math.Abs(got.X-want.X)+math.Abs(got.Y-want.Y)+math.Abs(got.Z-want.Z) > 1e-10 {
		t.Fatal(got, want)
	}
	if q.RotateVector(Point{}) != (Point{}) {
		t.Fatal("zero vector changed")
	}
}
