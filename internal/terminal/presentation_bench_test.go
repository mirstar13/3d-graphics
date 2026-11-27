package terminal

import (
	"bufio"
	"io"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func BenchmarkPresentLargeFrame(b *testing.B) {
	r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 80, 240)
	r.ColorMode = "truecolor"
	for y := range r.Surface {
		for x := range r.Surface[y] {
			r.Surface[y][x] = engine.FILLED_CHAR
			r.ColorBuffer[y][x] = engine.Color{R: uint8(x), G: uint8(y), B: 128}
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		r.Present()
	}
}
