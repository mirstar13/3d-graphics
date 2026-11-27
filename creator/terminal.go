package creator

import (
	"bufio"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// RunTerminal plays the current game in a centred monochrome viewport.
// Behaviors can request capture with World.CaptureMouse. Esc releases it,
// Q exits, and terminals without motion reporting can still deliver keys.
func (g *Game) RunTerminal(help string) error {
	m := &terminalGame{game: g, help: help}
	defer g.world.CaptureMouse(false)
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus()).Run()
	if err != nil {
		return err
	}
	return m.err
}

type terminalGame struct {
	game          *Game
	help, frame   string
	columns, rows int
	keys          []string
	pointer       input.Pointer
	renderer      *terminal.TerminalRenderer
	err           error
	last          time.Time
	debt          time.Duration
}

type frameTick time.Time

func nextFrame() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return frameTick(t) })
}

func (m *terminalGame) syncCapture() tea.Cmd {
	on := m.game.MouseCaptured()
	if on == m.pointer.Captured {
		return nil
	}
	m.pointer.Reset(on)
	if on {
		return tea.EnableMouseAllMotion
	}
	return tea.DisableMouse
}

func (m *terminalGame) Init() tea.Cmd {
	m.last = time.Now()
	return tea.Batch(m.syncCapture(), nextFrame())
}

func (m *terminalGame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.columns, m.rows = v.Width, v.Height
		m.pointer.Reset(m.pointer.Captured)
		m.draw()
	case tea.BlurMsg:
		m.keys = nil
		m.game.world.CaptureMouse(false)
		return m, m.syncCapture()
	case tea.KeyMsg:
		if v.Paste {
			break
		}
		switch v.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.keys = nil
			m.game.world.CaptureMouse(false)
			return m, m.syncCapture()
		default:
			if !m.game.world.session.Objective().Finished() {
				m.keys = append(m.keys, v.String())
			}
		}
	case tea.MouseMsg:
		if v.Action == tea.MouseActionMotion {
			m.pointer.Motion(v.X, v.Y)
		}
	case frameTick:
		now := time.Time(v)
		m.debt += min(250*time.Millisecond, max(0, now.Sub(m.last)))
		m.last = now
		for !m.game.world.session.Objective().Finished() {
			step := m.game.world.session.World.Stats().FixedStep
			if m.debt < step {
				break
			}
			m.debt -= step
			dx, dy := m.pointer.Take()
			m.err = m.game.StepWithInput(Input{m.keys, dx, dy})
			m.keys = m.keys[:0]
			if m.err != nil {
				return m, tea.Quit
			}
		}
		if m.game.world.session.Objective().Finished() {
			m.keys = nil
			m.game.world.CaptureMouse(false)
			m.draw()
			return m, m.syncCapture()
		}
		m.draw()
		return m, tea.Batch(m.syncCapture(), nextFrame())
	}
	return m, nil
}

func (m *terminalGame) draw() {
	if m.columns < 2 || m.rows < 2 {
		m.frame = ""
		return
	}
	w, h := terminal.FitViewport(0, 0, m.columns-1, m.rows-1)
	if m.renderer == nil {
		m.renderer = terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
		m.renderer.UseColor = false
		m.renderer.ShadowRenderer = nil
	} else {
		m.renderer.Resize(w, h)
	}
	s := m.game.world.session
	m.renderer.SetLightingSystem(s.Lighting)
	m.renderer.RenderScene(s.Scene)
	lines := make([]string, h)
	for i, row := range m.renderer.Surface {
		lines[i] = string(row)
	}
	hud := "Q quit | "
	if m.game.MouseCaptured() {
		hud += "Esc release | "
	}
	if objective := m.game.Objective(); objective.Outcome != "" {
		hud += objective.Outcome + " | "
	}
	hud += strings.ReplaceAll(m.help, "\n", " ")
	m.frame = terminal.CenteredGameView(runewidth.Truncate(hud, m.columns-1, ""), lines, w, m.columns, m.rows)
}
func (m *terminalGame) View() string { return m.frame }
