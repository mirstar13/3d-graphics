package app

import "strings"

// fitView keeps plain-text screens inside the terminal, reserving the last
// column to avoid automatic wrapping. All current menu labels are ASCII.
func fitView(view string, width, height int) string {
	lines := strings.Split(view, "\n")
	width, height = max(1, width-1), max(1, height)
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) > width {
			lines[i] = string(runes[:width])
		}
	}
	return strings.Join(lines[:min(len(lines), height)], "\n")
}
