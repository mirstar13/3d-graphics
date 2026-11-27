package terminal

import (
	"bytes"
	"fmt"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

type rowSpan struct{ row, first, last int }

type presentationHistory struct {
	spans               []rowSpan
	compatible          bool
	valid, color        bool
	mode                string
	width, height, x, y int
	glyphs              []rune
	colors              []engine.Color
}

// Send one span per touched row. Dense frames use the normal sequential
// encoder, which avoids a cursor command on each row of a full viewport.
func (r *TerminalRenderer) appendChangedRows(out *bytes.Buffer) bool {
	p := &r.previousPresentation
	sparse := p.valid && p.compatible && p.width == r.Width && p.height == r.Height && p.x == r.OffsetX && p.y == r.OffsetY && p.color == r.UseColor && p.mode == r.ColorMode
	n := r.Width * r.Height
	if len(p.glyphs) != n {
		p.glyphs = make([]rune, n)
		p.colors = make([]engine.Color, n)
	}
	p.width, p.height, p.x, p.y = r.Width, r.Height, r.OffsetX, r.OffsetY
	p.color, p.mode, p.valid = r.UseColor, r.ColorMode, true
	covered := 0
	p.spans = p.spans[:0]
	p.compatible = true
	for y, row := range r.Surface {
		start := y * r.Width
		glyphs, colors := p.glyphs[start:start+r.Width], p.colors[start:start+r.Width]
		first, last := r.Width, -1
		for x, glyph := range row {
			if (glyph < 32 || glyph > 126) && glyph != engine.FILLED_CHAR {
				p.compatible = false
			}
			if sparse {
				if glyph != glyphs[x] || (r.UseColor && r.ColorBuffer[y][x] != colors[x]) {
					first = min(first, x)
					last = x
				}
			}
		}
		if sparse {
			if last >= first {
				covered += last - first + 1
				if covered > n/3 {
					sparse = false
				} else {
					p.spans = append(p.spans, rowSpan{y, first, last})
				}
			}
		}
		copy(glyphs, row)
		if r.UseColor {
			copy(colors, r.ColorBuffer[y])
		}
	}
	if !sparse || !p.compatible {
		return false
	}
	for _, span := range p.spans {
		r.appendSpan(out, span.row, span.first, span.last)
	}
	return true
}

func (r *TerminalRenderer) appendSpan(out *bytes.Buffer, y, first, last int) {
	fmt.Fprintf(out, "\x1b[%d;%dH", r.OffsetY+y+1, r.OffsetX+first+1)
	var previous engine.Color
	for x := first; x <= last; x++ {
		if color := r.ColorBuffer[y][x]; r.UseColor && (x == first || color != previous) {
			out.WriteString(Foreground(color, r.ColorMode))
			previous = color
		}
		out.WriteRune(r.Surface[y][x])
	}
	if r.UseColor {
		out.WriteString(engine.ColorReset())
	}
}
