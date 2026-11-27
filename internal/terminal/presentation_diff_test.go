package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestSparsePresentAndInvalidation(t *testing.T) {
	for _, mode := range []string{"none", "16", "256", "truecolor"} {
		t.Run(mode, func(t *testing.T) {
			var output bytes.Buffer
			r := NewTerminalRenderer(bufio.NewWriter(&output), 6, 20)
			r.DiffPresentation = true
			r.ShowDebugInfo = false
			r.ColorMode, r.UseColor = mode, mode != "none"
			r.OffsetX, r.OffsetY = 3, 4
			r.Present()
			output.Reset()
			r.Present()
			if output.Len() != 0 {
				t.Fatalf("static viewport emitted %q", output.String())
			}
			r.Surface[2][7] = 'X'
			r.ColorBuffer[2][7] = engine.ColorRed
			r.Present()
			want := "\x1b[7;11H"
			if r.UseColor {
				want += Foreground(engine.ColorRed, mode)
			}
			want += "X"
			if r.UseColor {
				want += engine.ColorReset()
			}
			if output.String() != want {
				t.Fatalf("sparse output %q, want %q", output.String(), want)
			}
			r.Resize(20, 7)
			output.Reset()
			r.Present()
			if !strings.HasPrefix(output.String(), "\x1b[H") {
				t.Fatal("resize requires full frame")
			}
		})
	}
}

func BenchmarkFrameUpdates(b *testing.B) {
	for _, color := range []string{"none", "16", "256", "truecolor"} {
		for _, motion := range []string{"static", "object", "dense"} {
			for _, diff := range []bool{false, true} {
				b.Run(fmt.Sprintf("%s/%s/diff_%t", color, motion, diff), func(b *testing.B) {
					r := NewTerminalRenderer(bufio.NewWriter(io.Discard), 60, 160)
					r.ShowDebugInfo = false
					r.DiffPresentation = diff
					r.ColorMode, r.UseColor = color, color != "none"
					r.Present()
					var total int64
					for i := 0; b.Loop(); i++ {
						glyph := rune('A' + i%2)
						switch motion {
						case "object":
							for y := 25; y < 30; y++ {
								for x := 65; x < 75; x++ {
									r.Surface[y][x] = glyph
									r.ColorBuffer[y][x] = engine.ColorBlue
								}
							}
						case "dense":
							for y, row := range r.Surface {
								for x := range row {
									row[x] = glyph
									r.ColorBuffer[y][x] = engine.ColorBlue
								}
							}
						}
						r.Present()
						n, _ := r.FrameOutput()
						total += int64(n)
					}
					b.ReportMetric(float64(total)/float64(b.N), "bytes/frame")
				})
			}
		}
	}
}
