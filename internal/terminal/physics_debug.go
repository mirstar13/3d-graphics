package terminal

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
)

// RenderPhysicsDebug draws x-ray wireframes after AA without changing depth or
// scene objects. Screen clipping bounds work even for distant offscreen lines.
func (r *TerminalRenderer) RenderPhysicsDebug(lines []engine.DebugLine, camera *engine.Camera) {
	for _, line := range lines {
		clipped, visible := engine.ClipLineToNearPlane(engine.NewLine(line.Start, line.End), camera)
		if !visible {
			continue
		}
		x0, y0, z0 := camera.ProjectPointFloat(clipped.Start, r.Height, r.Width)
		x1, y1, z1 := camera.ProjectPointFloat(clipped.End, r.Height, r.Width)
		if z0 <= 0 || z1 <= 0 || z0 > camera.Far && z1 > camera.Far {
			continue
		}
		dx, dy := x1-x0, y1-y0
		t0, t1 := 0.0, 1.0
		accept := true
		for _, pair := range [][2]float64{{-dx, x0}, {dx, float64(r.Width-1) - x0}, {-dy, y0}, {dy, float64(r.Height-1) - y0}} {
			p, q := pair[0], pair[1]
			if math.Abs(p) < 1e-12 {
				if q < 0 {
					accept = false
					break
				}
				continue
			}
			t := q / p
			if p < 0 {
				t0 = math.Max(t0, t)
			} else {
				t1 = math.Min(t1, t)
			}
		}
		if !accept || t0 > t1 {
			continue
		}
		ax, ay := x0+t0*dx, y0+t0*dy
		bx, by := x0+t1*dx, y0+t1*dy
		steps := max(1, int(math.Ceil(math.Max(math.Abs(bx-ax), math.Abs(by-ay)))))
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			x, y := int(math.Round(ax+(bx-ax)*t)), int(math.Round(ay+(by-ay)*t))
			if x >= 0 && x < r.Width && y >= 0 && y < r.Height {
				r.Surface[y][x] = '+'
				r.ColorBuffer[y][x] = line.Color
			}
		}
	}
}
