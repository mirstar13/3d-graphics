package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCenteredPresentationAndMovedMargins(t *testing.T) {
	for _, color := range []bool{false, true} {
		var output bytes.Buffer
		r := NewTerminalRenderer(bufio.NewWriter(&output), 60, 160)
		r.UseColor = color
		r.FitTerminal(160, 60, 321, 121)
		if r.OffsetX != 80 || r.OffsetY != 30 {
			t.Fatalf("wrong offsets: %d,%d", r.OffsetX, r.OffsetY)
		}
		r.Present()
		got := output.String()
		for row := 31; row <= 90; row++ {
			if !strings.Contains(got, fmt.Sprintf("\033[%d;81H", row)) {
				t.Fatalf("missing centred row %d", row)
			}
		}
		if !strings.Contains(got, "\033[91;81H") {
			t.Fatal("status line did not follow the viewport")
		}
		if strings.Contains(got, "\r\n") {
			t.Fatal("newline would lose the left margin")
		}
		output.Reset()
		// Both terminal sizes use the same framebuffer dimensions. Movement
		// alone must erase the old location and invalidate the cached status.
		r.FitTerminal(160, 60, 401, 151)
		r.Present()
		if !strings.Contains(output.String(), "\033[2J") || !strings.Contains(output.String(), "\033[106;121H") {
			t.Fatal("old margins or status survived viewport movement")
		}
	}
}
