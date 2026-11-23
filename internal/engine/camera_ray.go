package engine

import (
	"fmt"
	"math"
)

// CellRay returns a world ray through a terminal cell's centre, starting at the
// near plane. Distance ends at the far plane, matching the perspective frustum.
func (cam *Camera) CellRay(column, row, width, height int) (origin, direction Point, distance float64, err error) {
	valid := func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
	if width <= 0 || height <= 0 || column < 0 || column >= width || row < 0 || row >= height || !valid(cam.Near) || !valid(cam.Far) || cam.Near < 0 || cam.Far <= cam.Near || !valid(cam.FOV.X) || !valid(cam.FOV.Y) || cam.FOV.X <= 0 || cam.FOV.X >= 180 || cam.FOV.Y <= 0 || cam.FOV.Y >= 180 {
		err = fmt.Errorf("invalid viewport or camera frustum")
		return
	}
	hx, hy := cam.projectionHalfExtents()
	local := Point{X: (2*(float64(column)+.5)/float64(width) - 1) * hx, Y: (1 - 2*(float64(row)+.5)/float64(height)) * hy, Z: 1}
	size := math.Sqrt(local.X*local.X + local.Y*local.Y + 1)
	direction = cam.Transform.GetWorldRotationQuaternion().RotateVector(Point{X: local.X / size, Y: local.Y / size, Z: 1 / size})
	origin = cam.Transform.TransformPoint(Point{X: local.X * cam.Near, Y: local.Y * cam.Near, Z: cam.Near})
	distance = (cam.Far - cam.Near) * size
	return
}
