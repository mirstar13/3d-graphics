package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"golang.org/x/term"
)

func terminalColorMode(requested, terminal, colorTerm string, noColor bool) (string, error) {
	switch requested {
	case "none", "16", "256", "truecolor":
		return requested, nil
	case "auto":
		if noColor || terminal == "dumb" {
			return "none", nil
		}
		if colorTerm == "truecolor" || colorTerm == "24bit" {
			return "truecolor", nil
		}
		if strings.Contains(terminal, "256color") {
			return "256", nil
		}
		return "16", nil
	default:
		return "", fmt.Errorf("invalid color mode %q (use auto, none, 16, 256, or truecolor)", requested)
	}
}

func terminalDimensions(width, height, columns, rows int) (int, int) {
	if columns < 2 || rows < 2 {
		columns, rows = 80, 24
	}
	// Leave a column to avoid automatic wrapping, and a row for status.
	if width == 0 || width > columns-1 {
		width = columns - 1
	}
	if height == 0 || height > rows-1 {
		height = rows - 1
	}
	return width, height
}

func ConfigureTerminal(config *engine.EngineConfig, colorMode string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("terminal renderer requires interactive stdin and stdout")
	}
	if os.Getenv("TERM") == "dumb" {
		return fmt.Errorf("terminal renderer requires cursor-addressing support; TERM=dumb is unsupported")
	}
	mode, err := terminalColorMode(colorMode, os.Getenv("TERM"), os.Getenv("COLORTERM"), os.Getenv("NO_COLOR") != "")
	if err != nil {
		return err
	}
	columns, rows, _ := term.GetSize(int(os.Stdout.Fd()))
	config.RequestedWidth, config.RequestedHeight = config.Width, config.Height
	config.Width, config.Height = terminalDimensions(config.Width, config.Height, columns, rows)
	config.ColorMode = mode
	config.UseColor = mode != "none"
	return nil
}

// terminalForeground maps RGB colors into the selected terminal palette.
func terminalForeground(c engine.Color, mode string) string {
	if mode == "truecolor" || mode == "" {
		return c.ToANSI()
	}
	palette := []engine.Color{
		{R: 0, G: 0, B: 0}, {R: 128, G: 0, B: 0}, {R: 0, G: 128, B: 0}, {R: 128, G: 128, B: 0},
		{R: 0, G: 0, B: 128}, {R: 128, G: 0, B: 128}, {R: 0, G: 128, B: 128}, {R: 192, G: 192, B: 192},
		{R: 128, G: 128, B: 128}, {R: 255, G: 0, B: 0}, {R: 0, G: 255, B: 0}, {R: 255, G: 255, B: 0},
		{R: 0, G: 0, B: 255}, {R: 255, G: 0, B: 255}, {R: 0, G: 255, B: 255}, {R: 255, G: 255, B: 255},
	}
	if mode == "256" {
		levels := []uint8{0, 95, 135, 175, 215, 255}
		for _, r := range levels {
			for _, g := range levels {
				for _, b := range levels {
					palette = append(palette, engine.Color{R: r, G: g, B: b})
				}
			}
		}
		for i := 0; i < 24; i++ {
			v := uint8(8 + i*10)
			palette = append(palette, engine.Color{R: v, G: v, B: v})
		}
	}
	best, distance := 0, 3*256*256
	for i, color := range palette {
		r, g, b := int(c.R)-int(color.R), int(c.G)-int(color.G), int(c.B)-int(color.B)
		d := r*r + g*g + b*b
		if d < distance {
			best, distance = i, d
		}
	}
	if mode == "256" {
		return fmt.Sprintf("\033[38;5;%dm", best)
	}
	if best < 8 {
		return fmt.Sprintf("\033[%dm", 30+best)
	}
	return fmt.Sprintf("\033[%dm", 90+best-8)
}
