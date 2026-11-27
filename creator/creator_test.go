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

func TestSceneChangeRetainsRoomOnErrorAndExpiresHandles(t *testing.T) {
	const exit = "scenes/exit.yaml"
	var triedInvalid, changed bool
	g, err := creator.Open("../examples/relay-rooms", func(w *creator.World, _ creator.Tick) error {
		if w.ActionPressed("use") {
			if !triedInvalid {
				triedInvalid = true
				if err := w.ChangeScene("scenes/missing.yaml"); err == nil {
					t.Error("accepted missing scene")
				}
				return nil
			}
			changed = true
			return w.ChangeScene(exit)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	relay, ok := g.World().Find("relay")
	if !ok {
		t.Fatal("missing relay")
	}
	if err := g.StepWithKeys("u"); err != nil || !triedInvalid || g.Scene() != "scenes/relay.yaml" {
		t.Fatalf("failed transition changed the current room: %v", err)
	}
	if err := g.Step(); err != nil || changed {
		t.Fatalf("action persisted into the next tick: %v", err)
	}
	if err := g.StepWithKeys("u"); err != nil || !changed || g.Scene() != exit {
		t.Fatalf("valid transition did not load the exit room: %v", err)
	}
	if _, err := g.World().Inspect(relay); err == nil {
		t.Fatal("accepted a handle from the previous room")
	}
	beacon, ok := g.World().Find("exit")
	if !ok {
		t.Fatal("missing exit beacon")
	}
	if err := g.Restart(nil); err != nil || g.Scene() != exit {
		t.Fatalf("restart did not reload the current room: %v", err)
	}
	if _, err := g.World().Inspect(beacon); err == nil {
		t.Fatal("accepted a handle from before restart")
	}
}
