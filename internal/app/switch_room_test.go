package app

import (
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

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
