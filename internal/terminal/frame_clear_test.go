package terminal

import (
	"fmt"
	"math"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestBeginFrameClearsEveryPlane(t *testing.T) {
	r := NewTerminalRenderer(nil, 1, 1)
	for _, size := range [][2]int{{1, 17}, {33, 1}, {321, 121}, {7, 4}} {
		r.Resize(size[0], size[1])
		for y := range r.Surface {
			for x := range r.Surface[y] {
				r.Surface[y][x] = 'X'
				r.ColorBuffer[y][x] = engine.Color{R: 255}
				r.ZBuffer[y][x] = 0
			}
		}
		r.BeginFrame()
		for y := range r.Surface {
			for x := range r.Surface[y] {
				if r.Surface[y][x] != ' ' || r.ColorBuffer[y][x] != engine.ColorBlack || !math.IsInf(r.ZBuffer[y][x], 1) {
					t.Fatalf("uncleared cell (%d, %d) in %v", x, y, size)
				}
			}
		}
	}
}

func BenchmarkBeginFrame(b *testing.B) {
	for _, size := range [][2]int{{160, 60}, {320, 120}, {640, 240}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			r := NewTerminalRenderer(nil, size[1], size[0])
			b.ReportAllocs()
			for b.Loop() {
				r.BeginFrame()
			}
		})
	}
}
