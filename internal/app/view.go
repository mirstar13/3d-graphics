package app

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// fitView keeps plain-text screens inside the terminal, reserving the last
// column to avoid automatic wrapping, including wide project/property names.
func fitView(view string, width, height int) string {
	lines := strings.Split(view, "\n")
	width, height = max(1, width-1), max(1, height)
	for i, line := range lines {
		lines[i] = runewidth.Truncate(line, width, "")
	}
	return strings.Join(lines[:min(len(lines), height)], "\n")
}
