package creator

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/mirstar13/go-3d-graphics/internal/audio"
	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// RunTerminal plays the current game in a centred monochrome viewport.
// Behaviors can request capture with World.CaptureMouse. Esc releases it,
// Q exits, and terminals without motion reporting can still deliver keys.
func (g *Game) RunTerminal(help string) error {
	return g.RunTerminalWithOptions(TerminalOptions{Help: help})
}

// TerminalOptions configures the HUD and the optional entry-scene restart.
type TerminalOptions struct {
	Help      string
	Status    func() string
	Restart   func() Behavior
	Reticle   bool
	MuteAudio bool
}

// RunTerminalWithOptions runs the terminal host with an optional status line,
// reticle and R restart action.
func (g *Game) RunTerminalWithOptions(o TerminalOptions) error {
	m := &terminalGame{game: g, options: o}
	defer g.world.CaptureMouse(false)
	defer g.Close()
	if !o.MuteAudio && len(g.sounds) > 0 {
		g.audio = audio.NewNative()
		g.world.session.AttachAudio(g.audio)
	}
	g.world.session.EnableBehaviors(context.Background(), g.project)
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus()).Run()
	if err != nil {
		return err
	}
	return m.err
}

type terminalGame struct {
	game          *Game
	frame         string
	glyphRows     []string
	options       TerminalOptions
	dragging      bool
	ticking       bool
	audioStatus   string
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
	m.dragging = false
	if on {
		return tea.EnableMouseAllMotion
	}
	return tea.DisableMouse
}

type terminalAudioFailure struct{ err error }

func (m *terminalGame) Init() tea.Cmd {
	if m.game.audio != nil {
		manager := m.game.audio
		return tea.Batch(m.initFrame(), func() tea.Msg {
			<-manager.Finished()
			return terminalAudioFailure{manager.Err()}
		})
	}
	return m.initFrame()
}

func (m *terminalGame) initFrame() tea.Cmd {
	m.last = time.Now()
	m.ticking = true
	return tea.Batch(m.syncCapture(), nextFrame())
}

func (m *terminalGame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case terminalAudioFailure:
		if v.err != nil {
			m.audioStatus = v.err.Error()
			m.draw()
		}
	case tea.WindowSizeMsg:
		m.columns, m.rows = v.Width, v.Height
		m.pointer.Reset(m.pointer.Captured)
		m.dragging = false
		m.draw()
	case tea.BlurMsg:
		m.dragging = false
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
		case "ctrl+y":
			if m.game.audio != nil {
				m.audioStatus = "Audio on"
				if m.game.audio.ToggleMute() {
					m.audioStatus = "Audio muted"
				}
				m.draw()
			}
		case "r":
			if m.options.Restart == nil {
				if !m.game.world.session.Objective().Finished() {
					m.keys = append(m.keys, "r")
				}
				break
			}
			m.err = m.game.restart(m.game.project.Manifest.EntryScene, m.options.Restart())
			if m.err != nil {
				return m, tea.Quit
			}
			m.keys = nil
			m.debt = 0
			m.last = time.Now()
			m.draw()
			if !m.ticking {
				m.ticking = true
				return m, tea.Batch(m.syncCapture(), nextFrame())
			}
			return m, m.syncCapture()
		case "esc":
			m.dragging = false
			m.keys = nil
			m.game.world.CaptureMouse(false)
			return m, m.syncCapture()
		default:
			if !m.game.world.session.Objective().Finished() {
				m.keys = append(m.keys, v.String())
			}
		}
	case tea.MouseMsg:
		switch {
		case v.Action == tea.MouseActionPress && v.Button == tea.MouseButtonRight && m.pointer.Captured:
			m.dragging = true
			m.pointer.Reanchor()
			m.pointer.Motion(v.X, v.Y)
		case v.Action == tea.MouseActionRelease:
			m.dragging = false
			m.pointer.Reanchor()
		case v.Action == tea.MouseActionMotion && (!m.game.project.Manifest.MouseLook.DragOnly || m.dragging):
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
			m.ticking = false
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
	if m.options.Reticle {
		m.renderer.Surface[h/2][w/2] = '+'
	}
	m.glyphRows = terminal.GlyphRows(m.glyphRows, m.renderer.Surface)
	hud := "Q quit | "
	if m.game.MouseCaptured() {
		hud += "Esc release | "
	}
	if objective := m.game.Objective(); objective.Outcome != "" {
		hud += objective.Outcome + " | "
	}
	if m.options.Restart != nil {
		hud += "R restart | "
	}
	if m.audioStatus != "" {
		hud += m.audioStatus + " | "
	}
	if m.game.audio != nil {
		hud += "Ctrl+Y mute | "
	}
	if m.options.Status != nil {
		hud += strings.ReplaceAll(m.options.Status(), "\n", " ") + " | "
	}
	hud += strings.ReplaceAll(m.options.Help, "\n", " ")
	m.frame = terminal.CenteredGameView(runewidth.Truncate(hud, m.columns-1, ""), m.glyphRows, w, m.columns, m.rows)
}
func (m *terminalGame) View() string { return m.frame }
