package terminal

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func TestTerminalColorDetection(t *testing.T) {
	for _, tc := range []struct {
		request, terminal, colorTerm string
		noColor                      bool
		want                         string
	}{
		{"auto", "xterm", "truecolor", false, "truecolor"},
		{"auto", "xterm-256color", "", false, "256"},
		{"auto", "vt100", "", false, "16"},
		{"auto", "xterm", "truecolor", true, "none"},
		{"auto", "dumb", "", false, "none"},
		{"16", "xterm", "truecolor", true, "16"},
	} {
		got, err := terminalColorMode(tc.request, tc.terminal, tc.colorTerm, tc.noColor)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: got %q, %v", tc, got, err)
		}
	}
	if _, err := terminalColorMode("invalid", "", "", false); err == nil {
		t.Fatal("invalid color mode accepted")
	}
}

func TestTerminalDimensions(t *testing.T) {
	for _, tc := range [][6]int{
		{0, 0, 80, 24, 79, 23},
		{223, 51, 40, 12, 39, 11},
		{20, 10, 80, 24, 20, 10},
		{0, 0, 0, 0, 79, 23},
	} {
		w, h := terminalDimensions(tc[0], tc[1], tc[2], tc[3])
		if w != tc[4] || h != tc[5] {
			t.Fatalf("%v: got %dx%d", tc, w, h)
		}
	}
}

func BenchmarkTerminalForeground(b *testing.B) {
	for _, mode := range []string{"16", "256", "truecolor"} {
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Foreground(engine.Color{R: 113, G: 157, B: 193}, mode)
			}
		})
	}
}

func TestTerminalOutputColorModes(t *testing.T) {
	for _, mode := range []string{"none", "16", "256", "truecolor"} {
		var output bytes.Buffer
		renderer := NewTerminalRenderer(bufio.NewWriter(&output), 1, 2)
		renderer.ColorMode = mode
		renderer.UseColor = mode != "none"
		renderer.Surface[0][0] = 'X'
		renderer.ColorBuffer[0][0] = engine.ColorRed
		renderer.Present()
		got := output.String()
		if !strings.Contains(got, "X") {
			t.Fatal("missing geometry")
		}
		switch mode {
		case "none":
			if strings.Contains(got, "m") {
				t.Fatalf("color sequence in monochrome output: %q", got)
			}
		case "16":
			if !strings.Contains(got, "\033[91m") {
				t.Fatalf("missing ANSI red: %q", got)
			}
		case "256":
			if !strings.Contains(got, "\033[38;5;9m") {
				t.Fatalf("missing indexed red: %q", got)
			}
		case "truecolor":
			if !strings.Contains(got, "\033[38;2;255;0;0m") {
				t.Fatalf("missing RGB red: %q", got)
			}
		}
	}
}

func TestInputCancellationAndStickyQuit(t *testing.T) {
	for _, event := range []keyboard.KeyEvent{{Rune: 'x'}, {Key: keyboard.KeyEsc}, {Key: keyboard.KeyCtrlC}} {
		manager := NewSilentInputManager()
		manager.stopChan = make(chan struct{})
		manager.done = make(chan struct{})
		events := make(chan keyboard.KeyEvent)
		go manager.readEvents(events)
		events <- event
		deadline := time.After(time.Second)
		for !manager.GetInputState().Quit {
			select {
			case <-deadline:
				t.Fatal("quit was not registered")
			default:
				time.Sleep(time.Millisecond)
			}
		}
		manager.ClearKeys()
		if !manager.GetInputState().Quit {
			t.Fatal("frame cleared quit request")
		}
		close(manager.stopChan)
		select {
		case <-manager.done:
		case <-time.After(time.Second):
			t.Fatal("input cancellation waited for another key")
		}
	}
}
