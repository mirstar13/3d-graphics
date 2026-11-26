package app

import (
	"context"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

type buildPanel struct {
	cancel context.CancelFunc
	output string
	scroll int
}
type projectBuildResult struct {
	path string
	err  error
}

func (m projectEditorModel) startBuild() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.build = buildPanel{cancel: cancel, output: "Compiling project…"}
	m.modal = "build"
	p := m.project
	return m, func() tea.Msg { path, err := p.Build(ctx); return projectBuildResult{path, err} }
}
func (m projectEditorModel) buildLines() []string {
	clean := strings.Map(func(r rune) rune {
		if r == '\n' || !unicode.IsControl(r) {
			return r
		}
		return ' '
	}, m.build.output)
	return strings.Split(runewidth.Wrap(clean, max(1, m.width-1)), "\n")
}
func (m projectEditorModel) buildKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "ctrl+c":
		if m.build.cancel != nil {
			m.build.cancel()
			m.build.output = "Canceling build…"
		} else {
			m.modal = ""
		}
	case "up":
		m.build.scroll = max(0, m.build.scroll-1)
	case "down":
		m.build.scroll = min(max(0, len(m.buildLines())-max(1, m.height-2)), m.build.scroll+1)
	case "ctrl+b":
		if m.build.cancel == nil {
			return m.startBuild()
		}
	}
	return m, nil
}
func (m projectEditorModel) buildView() string {
	lines := m.buildLines()
	rows := max(1, m.height-2)
	first := min(m.build.scroll, max(0, len(lines)-rows))
	help := "Up/Down scroll | Ctrl+B rebuild | Esc close"
	if m.build.cancel != nil {
		help = "Esc cancel"
	}
	return fitView("Project build\n"+help+"\n"+strings.Join(lines[first:min(len(lines), first+rows)], "\n"), m.width, m.height)
}
