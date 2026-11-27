package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func TestSwitchRoomCenteredFrame(t *testing.T) {
	p, err := project.Open("../../examples/switch-room")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"none", "256", "truecolor"} {
		m := projectPlayer{project: p, room: &switchRoom{}, playerID: "player", options: physics.DefaultOptions(), colorMode: mode}
		if err := m.reset(); err != nil {
			t.Fatal(err)
		}
		for _, size := range [][2]int{{321, 121}, {401, 151}, {80, 24}, {1, 1}} {
			m.width, m.height = size[0], size[1]
			lines := strings.Split(ansi.Strip(m.View()), "\n")
			if len(lines) > m.height {
				t.Fatal("frame scrolls past the terminal")
			}
			if m.width == 1 {
				continue
			}
			headerRows := 0
			for i, line := range lines {
				if strings.Contains(line, "Q quit") {
					headerRows = i + 1
					break
				}
			}
			w, h := m.renderer.GetDimensions()
			x, y := (m.width-1-w)/2, headerRows+(m.height-headerRows-h)/2
			if []rune(lines[y+h/2])[x+w/2] != '+' {
				t.Fatalf("%s %v: reticle is not centred", mode, size)
			}
			for _, line := range lines {
				if len([]rune(line)) >= m.width {
					t.Fatal("frame wraps at the terminal edge")
				}
			}
		}
	}
}

func TestSwitchRoomViewportBudgetAndResize(t *testing.T) {
	p, err := project.Open("../../examples/switch-room")
	if err != nil {
		t.Fatal(err)
	}
	m := projectPlayer{project: p, room: &switchRoom{}, playerID: "player", options: physics.DefaultOptions(), colorMode: "256"}
	if err := m.reset(); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{400, 150}, {80, 24}, {320, 100}} {
		m.width, m.height = size[0], size[1]
		if view := m.View(); view == "" {
			t.Fatal("empty game view")
		}
		w, h := m.renderer.GetDimensions()
		if w > 160 || h > 60 || w >= m.width || h >= m.height {
			t.Fatalf("unbounded viewport %dx%d in %v", w, h, size)
		}
	}
	m.requestedWidth, m.requestedHeight = 200, 70
	m.View()
	w, h := m.renderer.GetDimensions()
	if w != 200 || h != 70 {
		t.Fatalf("explicit viewport ignored: %dx%d", w, h)
	}
	// The game and demo use the same sizing policy.
	if w, h := terminal.FitViewport(0, 0, 400, 150); w != 160 || h != 60 {
		t.Fatalf("unexpected automatic budget: %dx%d", w, h)
	}
}

func TestSwitchRoomInteractions(t *testing.T) {
	p, err := project.Open("../../examples/switch-room")
	if err != nil {
		t.Fatal(err)
	}
	m := projectPlayer{project: p, room: &switchRoom{}, playerID: "player", options: physics.DefaultOptions()}
	if err := m.reset(); err != nil {
		t.Fatal(err)
	}
	key := func(k string) {
		t.Helper()
		if err := m.room.key(m.session, k); err != nil {
			t.Fatal(err)
		}
	}
	step := func() {
		t.Helper()
		if err := m.session.Advance(time.Second / 30); err != nil {
			t.Fatal(err)
		}
	}
	key("g")
	if m.room.held != "crate" {
		t.Fatal("grab", m.room.message)
	}
	if err := m.room.tick(m.session, time.Second/20, time.Now()); err != nil {
		t.Fatal(err)
	}
	step()
	key("g")
	if m.room.held != "" {
		t.Fatal("release")
	}
	key("f")
	step()
	crate, _ := m.session.Find("crate")
	b, _ := m.session.Inspect(crate)
	if b.Body.Velocity.Z <= 0 {
		t.Fatal("crate did not move forward", m.room.message, b.Body.Velocity, b.Entity.Position)
	}
	key("w")
	if err := m.room.tick(m.session, time.Second/20, time.Now()); err != nil {
		t.Fatal(err)
	}
	step()
	id, _ := m.session.Find("player")
	player, _ := m.session.Inspect(id)
	if player.Body.Velocity.Z <= 0 {
		t.Fatal("walk")
	}
	for i := range p.Scene.Entities {
		if p.Scene.Entities[i].ID == "player" {
			p.Scene.Entities[i].Position.Z = 2
		}
	}
	if err := m.reset(); err != nil {
		t.Fatal(err)
	}
	key("e")
	step()
	if _, exists := m.session.Find("gate"); exists {
		t.Fatal("button did not open gate", m.room.message)
	}
}

func TestSwitchRoomCoalescesMouseMotion(t *testing.T) {
	p, err := project.Open("../../examples/switch-room")
	if err != nil {
		t.Fatal(err)
	}
	m := projectPlayer{project: p, room: &switchRoom{}, options: physics.DefaultOptions(), width: 80, height: 30}
	if err = m.reset(); err != nil {
		t.Fatal(err)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	initial := m.View()
	for _, x := range []int{1, 8, 20} {
		m.Update(tea.MouseMsg{X: x, Y: 4, Action: tea.MouseActionMotion})
		if m.View() != initial || m.room.yaw != 0 {
			t.Fatal("rasterized or turned before tick")
		}
	}
	m.Update(editorTick(m.last.Add(time.Second / 30)))
	if m.room.yaw <= 0 || m.View() == initial {
		t.Fatal("missing camera turn")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.pointer.Captured {
		t.Fatal("escape did not release capture")
	}
}
