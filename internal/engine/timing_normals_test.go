package engine

import (
	"math"
	"testing"
)

func TestCameraOrbitIndependentOfFrameRate(t *testing.T) {
	var positions []Point
	for _, fps := range []int{30, 60, 144} {
		c := NewCameraController(NewCamera())
		for i := 0; i < fps; i++ {
			c.Update(InputState{}, OrientationTerminal, 1/float64(fps))
		}
		positions = append(positions, c.Camera.GetPosition())
	}
	for _, p := range positions[1:] {
		q := positions[0]
		if math.Abs(p.X-q.X)+math.Abs(p.Z-q.Z) > 1e-8 {
			t.Fatal(positions)
		}
	}
}
func TestNormalRemainsPerpendicularAfterScale(t *testing.T) {
	m := IdentityMatrix()
	m.M[0] = 2
	m.M[5] = 3
	n := m.TransformNormal(Point{X: 1, Y: 1})
	tangent := m.TransformDirection(Point{X: 1, Y: -1})
	if math.Abs(n.X*tangent.X+n.Y*tangent.Y) > 1e-9 {
		t.Fatal(n, tangent)
	}
	if math.Abs(n.X*n.X+n.Y*n.Y+n.Z*n.Z-1) > 1e-9 {
		t.Fatal(n)
	}
}
