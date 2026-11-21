package terminal

import (
	"bufio"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"io"
	"math"
	"testing"
)

func TestPhysicsOverlayClipsAndPreservesDepth(t *testing.T) {
	r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 10, 20)
	r.BeginFrame()
	camera := engine.NewCameraAt(0, 0, 0)
	camera.SetFOV(90, 90)
	r.RenderPhysicsDebug([]engine.DebugLine{{Start: engine.Point{X: -100000, Z: 5}, End: engine.Point{X: 100000, Z: 5}, Color: engine.ColorRed}}, camera)
	visible := 0
	for y, row := range r.Surface {
		for x, c := range row {
			if c == '+' {
				visible++
			}
			if !math.IsInf(r.ZBuffer[y][x], 1) {
				t.Fatal("overlay changed depth")
			}
		}
	}
	if visible < 18 {
		t.Fatal("screen clipping dropped line", visible)
	}
}
