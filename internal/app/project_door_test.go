package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestPressurePlateDoorAuthorSavePlayExitReset(t *testing.T) {
	m := inputEditor(t)
	add := func() string {
		t.Helper()
		id, err := m.document.Add("box")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	door, other, second := add(), add(), add()
	edit := func(id, key, value string) {
		t.Helper()
		m.selectID(id)
		for i, p := range project.Properties {
			if p.Key == key {
				m.property = i
				m, _ = editorRune(m, "e")
				m.draft = value
				m = resourceKey(m, tea.KeyEnter)
				if m.modal != "" {
					t.Fatalf("edit %s: %s", key, m.message)
				}
				return
			}
		}
		t.Fatalf("missing property %s", key)
	}
	edit("floor", "sensor", "true")
	edit("floor", "plate_signal", "entry")
	for _, id := range []string{door, other} {
		edit(id, "body", "static")
		edit(id, "position.x", "12")
		edit(id, "door_lift", "5")
	}
	edit(other, "position.z", "7")
	edit(door, "door_signal", "entry")
	edit(other, "door_signal", "different")
	edit("box", "position.y", "0.25")
	edit(second, "position.x", "3")
	edit(second, "position.y", "0.25")
	m.save()
	p, err := project.Open(m.project.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.install(p); err != nil {
		t.Fatal(err)
	}
	m.startPlay()
	m.play.World.SetPaused(true)
	step := func() {
		t.Helper()
		if err := m.play.Step(); err != nil {
			t.Fatal(err)
		}
	}
	height := func(id string) float64 {
		t.Helper()
		live, _ := m.play.Find(id)
		state, err := m.play.Inspect(live)
		if err != nil {
			t.Fatal(err)
		}
		visual := m.play.Scene.FindNode("entity:" + id).Transform.GetWorldPosition()
		if visual != state.Body.Position {
			t.Fatalf("visual and collider diverged %v %v", visual, state.Body.Position)
		}
		return visual.Y
	}
	closed := height(door)
	otherClosed := height(other)
	check := func(want float64) {
		t.Helper()
		if height(door) != want || height(other) != otherClosed {
			t.Fatalf("door heights %g %g, expected %g %g", height(door), height(other), want, otherClosed)
		}
	}
	leave := func(id string) {
		t.Helper()
		live, _ := m.play.Find(id)
		actor, err := m.play.Inspect(live)
		if err != nil {
			t.Fatal(err)
		}
		if err = m.play.World.Teleport(actor.Body.ID, engine.Point{X: 30, Y: 3}, actor.Body.Rotation); err != nil {
			t.Fatal(err)
		}
	}
	step()
	step()
	check(closed + 5)
	leave("box")
	step()
	step()
	check(closed + 5)
	leave(second)
	step()
	step()
	check(closed)
	m.resetPlay()
	step()
	step()
	check(closed + 5)
	m.resetPlay()
	check(closed)
}
