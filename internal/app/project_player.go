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
	room          *switchRoom
	renderer      *terminal.TerminalRenderer
	project       *project.Project
	options       physics.Options
	colorMode     string
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
	if m.room != nil {
		*m.room = switchRoom{}
		m.room.camera(s)
	}
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
		if m.room != nil {
			if err := m.room.tick(m.session, now.Sub(m.last), now); err != nil {
				m.err = err
				return m, tea.Quit
			}
		}
		m.err = m.session.Advance(now.Sub(m.last))
		if m.room != nil {
			m.room.camera(m.session)
		}
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
		if m.room != nil {
			m.err = m.room.key(m.session, key)
			if m.err != nil {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.playerID != "" {
			j, ok := playerImpulse(key)
			if !ok {
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
	if m.room != nil {
		out.WriteString("WASD walk | Arrows look | F push | G grab/drop | E button | ")
		out.WriteString(m.room.message + "\n")
	} else if m.playerID != "" {
		out.WriteString("Arrows move | Space jump | ")
	}
	out.WriteString("Q quit\n")
	w, h := max(1, m.width-1), max(1, m.height-strings.Count(out.String(), "\n"))
	if m.renderer == nil {
		m.renderer = terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
		m.renderer.UseColor = false
		m.renderer.ShadowRenderer = nil
	} else {
		m.renderer.Resize(w, h)
	}
	r := m.renderer
	r.SetLightingSystem(m.session.Lighting)
	r.RenderScene(m.session.Scene)
	aimX, aimY := -1, -1
	if m.room != nil {
		aimX, aimY = w/2, h/2
		r.Surface[aimY][aimX] = '+'
	}
	if m.colorMode != "" && m.colorMode != "none" {
		hud := fitView(strings.TrimSuffix(out.String(), "\n"), m.width, m.height)
		count := max(0, m.height-1-strings.Count(hud, "\n"))
		if count == 0 {
			return hud
		}
		return hud + "\n" + strings.Join(coloredRows(r, aimX, aimY, m.colorMode)[:min(h, count)], "\n")
	}
	for _, row := range r.Surface {
		out.WriteString(string(row))
		out.WriteByte('\n')
	}
	return fitView(out.String(), m.width, m.height)
}
func runProjectPlayer(root, id string, firstPerson bool, color string, options physics.Options) error {
	p, err := project.Open(root)
	if err != nil {
		return err
	}
	mode, _ := terminal.ResolveColorMode(color)
	m := &projectPlayer{colorMode: mode, project: p, options: options, playerID: id, width: 80, height: 24}
	if firstPerson {
		m.room = &switchRoom{}
	}
	if err = m.reset(); err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	return m.err
}

func playerImpulse(key string) (engine.Point, bool) {
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
		return j, false
	}
	return j, true
}
