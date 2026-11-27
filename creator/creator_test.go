package creator_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirstar13/go-3d-graphics/creator"
)

func scene(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for file, data := range map[string]string{"project.yaml": "version: 1\nname: Test\nentry_scene: scene.yaml\n", "scene.yaml": `version: 1
name: Launch
entities:
  - id: pad
    name: Pad
    shape: box
    body: static
    sensor: true
    size: {x: 2, y: 0.5, z: 2}
    mass: 1
  - id: player
    name: Player
    shape: sphere
    body: dynamic
    size: {x: 0.5, y: 0.5, z: 0.5}
    mass: 1
    position: {x: 0, y: 0.8, z: 0}
`} {
		if err := os.WriteFile(filepath.Join(root, file), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestExternalContactBehavior(t *testing.T) {
	root := scene(t)
	calls := 0
	launch := func(w *creator.World, tick creator.Tick) error {
		calls++
		if tick.Index != uint64(calls) || tick.Delta <= 0 {
			t.Fatal("invalid tick", tick)
		}
		player, ok := w.Find("player")
		if !ok {
			t.Fatal("missing player")
		}
		for _, event := range w.Contacts() {
			if event.Sensor && event.Phase == creator.ContactBegin && (event.A == player || event.B == player) {
				return w.Impulse(player, creator.Vec3{Y: 5})
			}
		}
		return nil
	}
	g, err := creator.Open(root, launch)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := g.Step(); err != nil {
			t.Fatal(err)
		}
	}
	player, _ := g.World().Find("player")
	state, err := g.World().Inspect(player)
	if err != nil || state.Velocity.Y <= 4 || state.Position.Y <= .8 {
		t.Fatal("behavior failed to launch player", state, err)
	}
	fresh, err := creator.Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fresh.World().Inspect(player); err == nil {
		t.Fatal("foreign handle accepted")
	}
	newPlayer, _ := fresh.World().Find("player")
	initial, err := fresh.World().Inspect(newPlayer)
	if err != nil || initial.Velocity.Y != 0 || initial.Position.Y != .8 {
		t.Fatal("authored state changed", initial, err)
	}
}

func TestBehaviorFailureStopsGame(t *testing.T) {
	sentinel := errors.New("stop")
	calls := 0
	g, err := creator.Open(scene(t), func(w *creator.World, tick creator.Tick) error { calls++; return sentinel })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !errors.Is(g.Step(), sentinel) {
			t.Fatal("lost behavior error")
		}
	}
	if calls != 1 {
		t.Fatal("failed behavior reran")
	}
}

func TestObjectiveStatusAndRestart(t *testing.T) {
	root := scene(t)
	sceneFile := filepath.Join(root, "scene.yaml")
	data, err := os.ReadFile(sceneFile)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\nobjective: {text: Reach the exit, signal: exit, tick_limit: 1}\n")...)
	if err = os.WriteFile(sceneFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	g, err := creator.Open(root, func(*creator.World, creator.Tick) error { calls++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := g.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if o := g.Objective(); o.Outcome != "Lost" || o.Tick != 1 || calls != 1 {
		t.Fatalf("terminal state %+v; calls %d", o, calls)
	}
	fresh, err := creator.Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o := fresh.Objective(); o.Outcome != "Playing" || o.Tick != 0 {
		t.Fatal(o)
	}
}
