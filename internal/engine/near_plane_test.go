package engine

import (
	"math"
	"testing"
)

func TestClippedVerticesProjectAfterCameraRotation(t *testing.T) {
	camera := NewCameraAt(2.3, 1.7, -4.2)
	camera.Near = .1
	for step := 0; step < 100; step++ {
		camera.SetRotation(.17, float64(step)*.07, 0)
		world := camera.Transform.GetWorldMatrix()
		tri := NewTriangle(
			world.TransformPoint(Point{X: -2, Y: -1, Z: -.5}),
			world.TransformPoint(Point{Y: 2, Z: 3}),
			world.TransformPoint(Point{X: 2, Y: -1, Z: 3}), 'x')
		for _, clipped := range ClipTriangleToNearPlane(tri, camera) {
			for _, p := range []Point{clipped.P0, clipped.P1, clipped.P2} {
				x, y, z := camera.ProjectPointFloat(p, 60, 160)
				if z < camera.Near || math.IsNaN(x) || math.IsNaN(y) {
					t.Fatalf("rotation %d: clipped vertex rejected at depth %.17g", step, camera.TransformToViewSpace(p).Z)
				}
			}
		}
	}
	// Tolerance must not admit geometry meaningfully behind the near plane.
	world := camera.Transform.GetWorldMatrix()
	p := world.TransformPoint(Point{Z: camera.Near - .0001})
	if _, _, z := camera.ProjectPointFloat(p, 60, 160); z != 0 {
		t.Fatal("projected a vertex behind the clipping plane")
	}
}
