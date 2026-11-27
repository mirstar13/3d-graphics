package terminal

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"golang.org/x/term"
	"os"
)

// SyncSize fits the viewport to the current console between completed frames.
// Failed size queries retain the last usable buffers.
func (r *TerminalRenderer) SyncSize() {
	columns, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || columns < 2 || rows < 2 {
		return
	}
	width, height := terminalDimensions(r.RequestedWidth, r.RequestedHeight, columns, rows)
	r.fitTerminal(width, height, columns, rows)
}

func (r *TerminalRenderer) fitTerminal(width, height, columns, rows int) {
	x, y := max(0, (columns-1-width)/2), max(0, (rows-1-height)/2)
	moved := x != r.OffsetX || y != r.OffsetY
	r.OffsetX, r.OffsetY = x, y
	if r.Resize(width, height) || moved {
		r.lastDebugLine = ""
		fmt.Fprint(r.Writer, "\033[2J\033[H")
	}
}

// Resize replaces framebuffer storage. Call only when render workers are idle.
// Filters and tile scheduling read the new dimensions on their next frame.
func (r *TerminalRenderer) Resize(width, height int) bool {
	if width < 1 || height < 1 || (width == r.Width && height == r.Height) {
		return false
	}
	r.Width, r.Height = width, height
	r.Surface = make([][]rune, height)
	r.ColorBuffer = make([][]engine.Color, height)
	r.ZBuffer = make([][]float64, height)
	for y := range r.Surface {
		r.Surface[y] = make([]rune, width)
		r.ColorBuffer[y] = make([]engine.Color, width)
		r.ZBuffer[y] = make([]float64, width)
	}
	r.lastDebugLine = ""
	r.BeginFrame()
	return true
}
