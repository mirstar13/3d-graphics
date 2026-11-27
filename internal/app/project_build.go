package app

import (
	"context"
	"errors"
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
	ctx, cancel := context.WithCancel(m.lifetime)
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
	case "f9":
		if m.build.cancel == nil {
			return m.startExport()
		}
	case "f6":
		if m.build.cancel == nil {
			return m.startChild()
		}
		if m.modal == "child" {
			m.restartChild = true
			m.build.cancel()
			m.build.output = "Restarting game…"
		}
	case "esc", "ctrl+c":
		if m.build.cancel != nil {
			m.build.cancel()
			m.build.output = "Canceling build…"
			if m.modal == "export" {
				m.build.output = "Canceling export…"
			}
			if m.modal == "child" {
				m.restartChild = false
				m.build.output = "Stopping game…"
			}
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
	title := "Project build"
	if m.modal == "child" {
		title = "Project game"
		help = "Up/Down scroll | F6 restart | Ctrl+B build | Esc close"
		if m.build.cancel != nil {
			help = "F6 restart | Esc stop"
		}
	} else if m.build.cancel == nil {
		help = "Up/Down scroll | Ctrl+B rebuild | F6 run | Esc close"
	}
	if m.modal == "export" {
		title = "Project export"
		help = "Esc cancel"
		if m.build.cancel == nil {
			help = "Up/Down scroll | F9 export again | Esc close"
		}
	}
	return fitView(title+"\n"+help+"\n"+strings.Join(lines[first:min(len(lines), first+rows)], "\n"), m.width, m.height)
}

type childResult struct {
	output string
	err    error
}

func (r childResult) status() string {
	status := "Game exited successfully"
	if errors.Is(r.err, context.Canceled) {
		status = "Game stopped"
	} else if r.err != nil {
		status = "Game failed: " + r.err.Error()
	}
	if r.output != "" {
		status += "\n" + r.output
	}
	return status
}
func (m projectEditorModel) startChild() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(m.lifetime)
	m.restartChild = false
	m.modal = "child"
	m.build = buildPanel{cancel: cancel, output: "Running last build with saved project files…"}
	p := m.project
	return m, func() tea.Msg { output, err := p.RunBuilt(ctx); return childResult{output, err} }
}

type projectExportResult struct {
	path string
	err  error
}

func (m projectEditorModel) startExport() (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(m.lifetime)
	m.modal = "export"
	m.build = buildPanel{cancel: cancel, output: "Exporting saved project…"}
	p := m.project
	return m, func() tea.Msg { path, err := p.Export(ctx); return projectExportResult{path, err} }
}

// Let Bubble Tea suspend its renderer and input reader while the child owns
// the terminal, then restore the editor before reporting the exit status.
func (m projectEditorModel) startInteractiveChild() (tea.Model, tea.Cmd) {
	cmd, err := m.project.BuiltCommand(m.lifetime, m.project.Manifest.TerminalArgs...)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return interactiveChildResult{err} })
}

type interactiveChildResult struct{ err error }
