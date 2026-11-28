package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/audio"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/input"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

// projectPlayer owns the terminal while a saved scene runs outside the editor.
type projectPlayer struct {
	adaptive                  bool
	present                   func(projectView)
	previousView              projectView
	quality                   terminal.AdaptiveQuality
	work                      time.Duration
	budget                    time.Duration
	behind                    func(time.Duration) bool
	rasterWidth, rasterHeight int
	controlStatus             string
	screens                   bool
	menu                      bool
	mouseLook                 bool
	dragging                  bool
	help                      string
	control                   playerControl
	newControl                func() playerControl

	audio                           *audio.Manager
	sounds                          map[string]*audio.Clip
	audioStatus                     string
	frame                           string
	glyphRows                       []string
	frameLayout                     [4]int
	pointer                         input.Pointer
	room                            *switchRoom
	renderer                        *terminal.TerminalRenderer
	drawFrame                       func(*terminal.TerminalRenderer)
	frameClock                      func() time.Time
	manualTicks                     bool
	project                         *project.Project
	options                         physics.Options
	colorMode                       string
	playerID                        string
	session                         *game.Session
	last                            time.Time
	width, height                   int
	requestedWidth, requestedHeight int
	err                             error
}

func (m *projectPlayer) reset() error {
	m.quality = terminal.AdaptiveQuality{}
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
	if m.sounds == nil && len(m.project.Manifest.Sounds) > 0 {
		m.sounds, err = m.project.LoadSounds()
		if err != nil {
			return err
		}
	}
	if err := s.ConfigureAudio(m.sounds); err != nil {
		return err
	}
	if m.session != nil {
		m.session.Close()
	}
	s.AttachAudio(m.audio)
	s.SetResultSounds(m.project.Manifest.WinSound, m.project.Manifest.LossSound)
	s.EnableBehaviors(context.Background(), m.project)
	m.session = s
	if m.room != nil {
		*m.room = switchRoom{}
		m.room.camera(s)
	}
	m.controlStatus = ""
	m.pointer.Reset(false)
	m.mouseLook, m.dragging = false, false
	if m.newControl != nil {
		m.control = m.newControl()
	}
	m.last = time.Now()
	return nil
}
func (m *projectPlayer) Init() tea.Cmd {
	if m.menu {
		return waitAudio(m.audio)
	}
	return tea.Batch(nextEditorTick(), waitAudio(m.audio))
}

func (m *projectPlayer) gameScreen() (terminal.GameScreen, bool) {
	if !m.screens {
		return terminal.GameScreen{}, false
	}
	screen := terminal.GameScreen{Title: m.project.Manifest.Name, Detail: m.session.Objective().Text, Help: m.help}
	if m.menu {
		return screen, true
	}
	switch m.session.Objective().Outcome {
	case game.Won:
		screen.Result = "You win"
	case game.Lost:
		screen.Result = "Game over"
	default:
		return screen, false
	}
	screen.Help = "Enter or R replay · Esc main menu · Q quit"
	if m.audioStatus != "" {
		screen.Help += " · " + m.audioStatus
	}
	return screen, true
}

func (m *projectPlayer) startRound() (tea.Model, tea.Cmd) {
	m.err = m.reset()
	if m.err != nil {
		return m, tea.Quit
	}
	m.menu = false
	return m, tea.Batch(tea.EnableMouseCellMotion, nextEditorTick())
}
func (m *projectPlayer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	now := time.Now
	if m.frameClock != nil {
		now = m.frameClock
	}
	model, cmd := m.update(msg, now)
	m.submitView()
	return model, cmd
}

func (m *projectPlayer) update(msg tea.Msg, nowFn func() time.Time) (tea.Model, tea.Cmd) {
	// Motion is accumulated until the next simulation tick. Bubble Tea may
	// ask for View after every mouse report, so keep the completed frame.
	if _, motion := msg.(tea.MouseMsg); !motion {
		m.frame = ""
	}
	if screen, active := m.gameScreen(); active {
		switch screen.Input(msg, m.width, m.height) {
		case terminal.ScreenStart:
			m.frame = ""
			return m.startRound()
		case terminal.ScreenMenu:
			m.frame = ""
			m.menu = true
			m.session.Close()
			return m, nil
		case terminal.ScreenQuit:
			return m, tea.Quit
		}
		switch msg.(type) {
		case tea.WindowSizeMsg, audioFailure:
		default:
			return m, nil
		}
	}
	switch v := msg.(type) {
	case audioFailure:
		if v.err != nil {
			m.audioStatus = v.err.Error()
		}
	case tea.BlurMsg:
		m.quality = terminal.AdaptiveQuality{}
		m.pointer.Reset(false)
		m.dragging, m.mouseLook = false, false
		return m, tea.EnableMouseCellMotion
	case tea.MouseMsg:
		if m.room != nil && !m.session.Objective().Finished() {
			switch {
			case v.Action == tea.MouseActionPress && v.Button == tea.MouseButtonRight:
				m.dragging = true
				m.pointer.Reset(true)
				m.pointer.Motion(v.X, v.Y)
			case v.Action == tea.MouseActionRelease && m.dragging:
				m.dragging = false
				if m.mouseLook {
					m.pointer.Reanchor()
				} else {
					m.pointer.EndDrag()
				}
			case v.Action == tea.MouseActionMotion && m.pointer.Captured &&
				(!m.project.Manifest.MouseLook.DragOnly || m.dragging):
				m.pointer.Motion(v.X, v.Y)
			}
		}
	case tea.WindowSizeMsg:
		m.quality = terminal.AdaptiveQuality{}
		m.dragging = false
		m.pointer.Reset(m.mouseLook)
		m.width = v.Width
		m.height = v.Height
	case editorTick:
		now := time.Time(v)
		if m.room != nil {
			dx, dy := m.pointer.Take()
			right, up := m.project.Manifest.MouseLook.Angles(dx, dy)
			m.room.yaw += right
			m.room.pitch = max(-1.1, min(1.1, m.room.pitch+up))
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
			if m.manualTicks {
				return m, nil
			}
			return m, nextEditorTick()
		}
		m.pointer.Reset(false)
		m.dragging, m.mouseLook = false, false
		return m, tea.EnableMouseCellMotion
	case tea.KeyMsg:
		if v.Paste {
			return m, nil
		}
		key := v.String()
		if key == "ctrl+y" && m.audio != nil && len(m.sounds) > 0 {
			m.audioStatus = "Audio on"
			if m.audio.ToggleMute() {
				m.audioStatus = "Audio muted"
			}
			return m, nil
		}
		if m.room != nil && ((key == "esc" && m.mouseLook) || (key == "m" && !m.session.Objective().Finished())) {
			m.mouseLook = !m.mouseLook
			m.dragging = false
			m.pointer.Reset(m.mouseLook)
			if m.mouseLook && !m.project.Manifest.MouseLook.DragOnly {
				return m, tea.EnableMouseAllMotion
			}
			return m, tea.EnableMouseCellMotion
		}
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
		m.session.BehaviorInput(key)
		if m.room != nil {
			m.err = m.room.keyAt(m.session, key, nowFn())
			if m.err != nil {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.control != nil {
			status, err := m.control(m.session, key)
			if status != "" {
				m.controlStatus = status
			}
			m.err = err
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
			if m.err == nil && m.playerID == "tug" {
				m.err = optionalSound(m.session, "thruster")
			}
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
	if screen, active := m.gameScreen(); active {
		return screen.View(m.width, m.height)
	}
	layout := [4]int{m.width, m.height, m.requestedWidth, m.requestedHeight}
	if m.frame == "" || layout != m.frameLayout {
		m.frame = m.renderView()
		m.frameLayout = layout
	}
	return m.frame
}

func (m *projectPlayer) renderView() string {
	start := time.Now()
	defer func() { m.work = time.Since(start) }()
	var out strings.Builder
	out.WriteString(objectiveHUD(m.session.Objective()))
	if m.room != nil {
		if m.mouseLook {
			out.WriteString("Esc release | ")
		} else {
			out.WriteString("Right-drag look | M free look | ")
		}
		out.WriteString("WASD walk | Arrows look | F push | G grab/drop | E button | ")
		out.WriteString(m.room.message + "\n")
	} else if m.help != "" {
		out.WriteString(m.help + "\n")
		if m.controlStatus != "" {
			out.WriteString(m.controlStatus + "\n")
		}
	} else if m.playerID != "" {
		out.WriteString("Arrows move | Space jump | ")
	}
	out.WriteString("Q quit")
	if len(m.sounds) > 0 {
		out.WriteString(" | Ctrl+Y mute")
	}
	if m.audioStatus != "" {
		out.WriteString(" | " + m.audioStatus)
	}
	out.WriteByte('\n')
	hud := fitView(strings.TrimSuffix(out.String(), "\n"), m.width, m.height)
	availableHeight := m.height - strings.Count(hud, "\n") - 1
	if availableHeight <= 0 {
		return hud
	}
	w, h := max(1, m.width-1), availableHeight
	w, h = terminal.FitViewport(m.requestedWidth, m.requestedHeight, w, h)
	if m.renderer == nil {
		m.renderer = terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), h, w)
		m.renderer.UseColor = false
		m.renderer.ShadowRenderer = nil
	} else {
		m.renderer.Resize(w, h)
	}
	r := m.renderer
	r.SetLightingSystem(m.session.Lighting)
	tier := terminal.FullQuality
	if m.adaptive {
		tier = m.quality.Update(m.session.Scene.Camera, m.work, m.behind != nil && m.behind(m.budget), m.budget)
	}
	m.rasterWidth, m.rasterHeight = r.RenderQuality(tier, func() {
		if m.drawFrame != nil {
			m.drawFrame(r)
		} else {
			r.RenderScene(m.session.Scene)
		}
	})
	aimX, aimY := -1, -1
	if m.room != nil {
		aimX, aimY = w/2, h/2
		r.Surface[aimY][aimX] = '+'
	}
	var rows []string
	if m.colorMode != "" && m.colorMode != "none" {
		rows = coloredRows(r, aimX, aimY, m.colorMode)
	} else {
		m.glyphRows = terminal.GlyphRows(m.glyphRows, r.Surface)
		rows = m.glyphRows
	}
	return terminal.CenteredGameView(hud, rows, w, m.width, m.height)
}

func runProjectPlayer(root, id string, firstPerson bool, color string, options physics.Options, width, height int) error {
	return runPlayer(root, gameProfile{player: id, firstPerson: firstPerson}, false, color, options, width, height)
}

func runPlayer(root string, profile gameProfile, screens bool, color string, options physics.Options, width, height int) error {
	p, err := project.Open(root)
	if err != nil {
		return err
	}
	mode, _ := terminal.ResolveColorMode(color)
	m := &projectPlayer{colorMode: mode, project: p, options: options, playerID: profile.player, width: 80, height: 24, screens: screens, menu: screens, help: profile.help, newControl: profile.controls}
	m.audio = audio.NewNative()
	defer m.audio.Close()
	m.requestedWidth, m.requestedHeight = width, height
	if profile.firstPerson {
		m.room = &switchRoom{}
	}
	if err = m.reset(); err != nil {
		return err
	}
	defer func() { m.session.Close() }()
	if os.Getenv("GRAPHICS_ADAPTIVE_QUALITY") == "1" {
		err = runAdaptivePlayer(m)
	} else {
		_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithReportFocus(), tea.WithMouseCellMotion()).Run()
	}
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

// RunSwitchRoom runs a packaged Switch Room project with first-person controls.
func RunSwitchRoom(root string) error {
	return RunExample(root, "switch-room")
}
