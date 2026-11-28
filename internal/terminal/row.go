package terminal

import (
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func EncodeRow(glyphs []rune, colors []engine.Color, mode string, reticle int) string {
	var row strings.Builder
	previous := ""
	var last engine.Color
	for x, glyph := range glyphs {
		color := colors[x]
		if x == reticle {
			glyph, color = '+', engine.ColorWhite
		}
		if mode != "none" && glyph != ' ' && (previous == "" || color != last) {
			code := Foreground(color, mode)
			if code == "\033[30m" {
				code = "\033[90m"
			}
			if code != previous {
				row.WriteString(code)
			}
			previous, last = code, color
		}
		row.WriteRune(glyph)
	}
	if previous != "" {
		row.WriteString("\033[0m")
	}
	return row.String()
}
