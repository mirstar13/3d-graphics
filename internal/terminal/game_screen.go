package terminal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

type ScreenAction uint8

const (
	ScreenNone ScreenAction = iota
	ScreenStart
	ScreenMenu
	ScreenQuit
)

// GameScreen shares layout and input between the standalone project hosts.
// Coordinates are terminal cells, including the margins around the panel.
type GameScreen struct {
	Title, Detail, Help string
	Result              string
}

type screenButton struct {
	label  string
	action ScreenAction
}

func (s GameScreen) buttons() []screenButton {
	if s.Result != "" {
		return []screenButton{{"Play again", ScreenStart}, {"Main menu", ScreenMenu}, {"Quit", ScreenQuit}}
	}
	return []screenButton{{"Play", ScreenStart}, {"Quit", ScreenQuit}}
}

func (s GameScreen) layout(width, height int) ([]string, int, int) {
	width = max(1, width-1)
	buttons := s.buttons()
	var lines []string
	title := s.Title
	if s.Result != "" {
		title = s.Result + " · " + title
	}
	lines = append(lines, title)
	if s.Result == "" {
		s.Help += "\nEnter play · Q quit"
	}
	for _, paragraph := range []string{s.Detail, s.Help} {
		if paragraph != "" {
			lines = append(lines, strings.Split(runewidth.Wrap(paragraph, min(width, 64)), "\n")...)
		}
	}
	if len(lines)+len(buttons)+1 <= height {
		lines = append(lines, "")
	}
	start := len(lines)
	for _, button := range buttons {
		lines = append(lines, "["+button.label+"]")
	}
	if len(lines) > height {
		skip := min(len(lines)-height, max(0, start-1))
		lines = append(lines[:1], lines[1+skip:]...)
		start -= skip
	}
	top := max(0, (height-len(lines))/2)
	return lines, start, top
}

func (s GameScreen) View(width, height int) string {
	lines, _, top := s.layout(width, height)
	var out strings.Builder
	out.WriteString(strings.Repeat("\n", top))
	width = max(1, width-1)
	for i, line := range lines[:min(len(lines), max(0, height-top))] {
		if i > 0 {
			out.WriteByte('\n')
		}
		line = runewidth.Truncate(line, width, "")
		out.WriteString(strings.Repeat(" ", (width-runewidth.StringWidth(line))/2))
		out.WriteString(line)
	}
	return out.String()
}

func (s GameScreen) Input(msg tea.Msg, width, height int) ScreenAction {
	switch v := msg.(type) {
	case tea.KeyMsg:
		if v.Paste {
			return ScreenNone
		}
		switch v.String() {
		case "enter", "r", " ":
			return ScreenStart
		case "esc":
			if s.Result != "" {
				return ScreenMenu
			}
			return ScreenQuit
		case "q", "ctrl+c":
			return ScreenQuit
		}
	case tea.MouseMsg:
		if v.Action == tea.MouseActionPress && v.Button == tea.MouseButtonLeft && v.Y >= 0 && v.Y < height {
			lines, start, top := s.layout(width, height)
			row := v.Y - top
			buttons := s.buttons()
			if row >= start && row < len(lines) {
				w := min(max(1, width-1), runewidth.StringWidth(lines[row]))
				left := (max(1, width-1) - w) / 2
				if v.X >= left && v.X < left+w {
					return buttons[row-start].action
				}
			}
		}
	}
	return ScreenNone
}
