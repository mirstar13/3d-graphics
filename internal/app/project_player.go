package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// projectPlayer owns the terminal while a saved scene runs outside the editor.
type projectPlayer struct {
	project       *project.Project
	options       physics.Options
	playerID      string
	session       *game.Session
	last          time.Time
	width, height int
	err           error
}

func (m *projectPlayer) reset() error {
	s, err := game.NewProjectSession(m.project.Root, m.project.Scene, m.options)
	if err != nil {
		return err
	}
	if m.playerID != "" {
		id, ok := s.Find(m.playerID)
		if !ok {
			return fmt.Errorf("unknown player entity %q", m.playerID)
		}
		state, err := s.State(id)
		if err != nil {
			return err
		}
		if state.Body != "dynamic" {
			return fmt.Errorf("player entity %q must be dynamic", m.playerID)
		}
	}
	m.session = s
	m.last = time.Now()
	return nil
}
func (m *projectPlayer) Init() tea.Cmd { return nextEditorTick() }
func (m *projectPlayer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case editorTick:
		now := time.Time(v)
		m.err = m.session.Advance(now.Sub(m.last))
		m.last = now
		if m.err != nil {
			return m, tea.Quit
		}
		if !m.session.Objective().Finished() {
			return m, nextEditorTick()
		}
	case tea.KeyMsg:
		key := v.String()
		if key == "esc" || key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.session.Objective().Finished() {
			if key == "r" {
				m.err = m.reset()
				if m.err != nil {
					return m, tea.Quit
				}
				return m, nextEditorTick()
			}
			return m, nil
		}
		if m.playerID != "" {
			var j engine.Point
			switch key {
			case "left":
				j.X = -2
			case "right":
				j.X = 2
			case "up":
				j.Z = 2
			case "down":
				j.Z = -2
			case " ":
				j.Y = 5
			default:
				return m, nil
			}
			id, _ := m.session.Find(m.playerID)
			m.err = m.session.QueueImpulse(id, j)
			if m.err != nil {
				return m, tea.Quit
			}
		}
	}
	return m, nil
}
func objectiveHUD(o game.ObjectiveStatus) string {
	if o.Text == "" {
		return ""
	}
	if o.Finished() {
		return fmt.Sprintf("Objective: %s\n%s | R restart\n", o.Text, o.Outcome)
	}
	if o.TickLimit > 0 {
		return fmt.Sprintf("Objective: %s\nTicks left: %d\n", o.Text, o.TickLimit-o.Tick)
	}
	return "Objective: " + o.Text + "\n"
}
func (m *projectPlayer) View() string {
	var out strings.Builder
	out.WriteString(objectiveHUD(m.session.Objective()))
	if m.playerID != "" {
		out.WriteString("Arrows move | Space jump | ")
	}
	out.WriteString("Q quit\n")
	w, h := max(1, min(120, m.width-1)), max(1, min(40, m.height-4))
	r := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
	r.UseColor = false
	r.ShadowRenderer = nil
	r.SetLightingSystem(m.session.Lighting)
	r.RenderScene(m.session.Scene)
	for _, row := range r.Surface {
		out.WriteString(string(row))
		out.WriteByte('\n')
	}
	return fitView(out.String(), m.width, m.height)
}
func runProjectPlayer(root, id string, options physics.Options) error {
	p, err := project.Open(root)
	if err != nil {
		return err
	}
	m := &projectPlayer{project: p, options: options, playerID: id, width: 80, height: 24}
	if err = m.reset(); err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	return m.err
}
