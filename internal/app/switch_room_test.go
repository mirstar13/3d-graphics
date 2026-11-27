package app

import (
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

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
