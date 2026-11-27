package terminal

import "strings"

// CenteredGameView positions a bounded frame below the HUD.
func CenteredGameView(hud string, rows []string, frameWidth, columns, height int) string {
	available := max(0, height-1-strings.Count(hud, "\n"))
	rows = rows[:min(len(rows), available)]
	if len(rows) == 0 {
		return hud
	}
	left := strings.Repeat(" ", max(0, (columns-1-frameWidth)/2))
	var out strings.Builder
	out.WriteString(hud)
	out.WriteByte('\n')
	out.WriteString(strings.Repeat("\n", (available-len(rows))/2))
	for i, row := range rows {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(left)
		out.WriteString(row)
	}
	return out.String()
}

// GlyphRows copies framebuffer rows into reusable string storage for a text UI.
func GlyphRows(dst []string, surface [][]rune) []string {
	if cap(dst) < len(surface) {
		dst = make([]string, len(surface))
	}
	dst = dst[:len(surface)]
	for i, row := range surface {
		dst[i] = string(row)
	}
	return dst
}
