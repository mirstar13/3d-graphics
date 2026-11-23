package engine

import (
	"math"
	"testing"
)

func TestCellRayMatchesProjection(t *testing.T) {
	camera := NewCameraAt(4, 7, -9)
	camera.LookAt(Point{Y: 1})
	camera.Near = .2
	camera.Far = 80
	for _, size := range [][2]int{{1, 1}, {40, 14}, {71, 27}} {
		for _, cell := range [][2]int{{0, 0}, {size[0] / 2, size[1] / 2}, {size[0] - 1, size[1] - 1}} {
			origin, direction, distance, err := camera.CellRay(cell[0], cell[1], size[0], size[1])
			if err != nil {
				t.Fatal(err)
			}
			point := Point{X: origin.X + direction.X*distance*.5, Y: origin.Y + direction.Y*distance*.5, Z: origin.Z + direction.Z*distance*.5}
			x, y, _ := camera.ProjectPointFloat(point, size[1], size[0])
			if math.Abs(x-float64(cell[0])-.5) > 1e-9 || math.Abs(y-float64(cell[1])-.5) > 1e-9 {
				t.Fatal("cell ray mismatch", x, y, cell)
			}
			end := Point{X: origin.X + direction.X*distance, Y: origin.Y + direction.Y*distance, Z: origin.Z + direction.Z*distance}
			if math.Abs(camera.TransformToViewSpace(origin).Z-camera.Near) > 1e-9 || math.Abs(camera.TransformToViewSpace(end).Z-camera.Far) > 1e-9 {
				t.Fatal("ray clip planes")
			}
		}
	}
	if _, _, _, err := camera.CellRay(-1, 0, 10, 10); err == nil {
		t.Fatal("invalid cell accepted")
	}
	camera.FOV.X = 180
	if _, _, _, err := camera.CellRay(0, 0, 10, 10); err == nil {
		t.Fatal("invalid frustum accepted")
	}
}
