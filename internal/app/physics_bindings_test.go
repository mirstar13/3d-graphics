package app

import (
	"bufio"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
	"io"
	"testing"
)

func TestPhysicsPlaygroundBindingsAndPause(t *testing.T) {
	sim, scene, lighting := testSimulation(t, demo.PhysicsPlayground)
	if len(sim.bindings.items) < 20 {
		t.Fatal("playground bodies missing")
	}
	step := physics.DefaultOptions().FixedStep
	initial := scene.FindNode("Ball0").Transform.Position
	for i := 0; i < 120; i++ {
		sendSimulation(t, sim, step)
	}
	if scene.FindNode("Ball0").Transform.Position.Y >= initial.Y {
		t.Fatal("gravity not synchronized")
	}
	for _, binding := range sim.bindings.items {
		s, err := sim.world.Body(binding.id)
		offset := s.Rotation.RotateVector(binding.visualOffset)
		expected := engine.Point{X: s.Position.X + offset.X, Y: s.Position.Y + offset.Y, Z: s.Position.Z + offset.Z}
		if err != nil || expected != binding.node.Transform.Position {
			t.Fatal("pose ownership lost", err)
		}
	}
	sendSimulation(t, sim, step, engine.PauseSimulation)
	tick := sim.status().Tick
	pose := scene.FindNode("Ball0").Transform.Position
	base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 40, 120)
	base.ShadowRenderer = nil
	base.SetLightingSystem(lighting)
	for mode := engine.RenderModeSingle; mode <= engine.RenderModeParallelBatched; mode++ {
		renderer := terminal.ComposeRenderer(base, engine.EngineConfig{RenderMode: mode, AAMode: engine.AAMSAA4x, NumWorkers: 2, TileSize: 16})
		renderer.RenderScene(scene)
		base.RenderPhysicsDebug(sim.world.DebugLines(), scene.Camera)
	}
	if sim.status().Tick != tick || scene.FindNode("Ball0").Transform.Position != pose {
		t.Fatal("rendering altered physics")
	}
	sendSimulation(t, sim, step, engine.TogglePhysicsDebug, engine.StepSimulation)
	if !sim.overlay || sim.status().Tick != tick+1 {
		t.Fatal("overlay/step command failed")
	}
	binding := sim.bindings.items[1]
	if err := sim.bindings.setEnabled(binding.id, false); err != nil {
		t.Fatal(err)
	}
	if binding.node.Enabled {
		t.Fatal("visual enabled while physics disabled")
	}
	sim.world.RemoveBody(binding.id)
	sim.bindings.synchronize()
	if scene.FindNode(binding.node.Name) != nil {
		t.Fatal("removed body retained binding")
	}
}
func TestBindingsRejectConflictingOwnership(t *testing.T) {
	sim, scene, _ := testSimulation(t, demo.PhysicsPlayground)
	node := scene.FindNode("Ball0")
	node.Transform.SetScale(2, 1, 1)
	if err := sim.bindings.add(demo.BodySpec{Node: node, Desc: physics.NewBodyDesc(physics.Dynamic, physics.Sphere(1))}); err == nil {
		t.Fatal("scaled binding accepted")
	}
	node.Transform.SetScale(1, 1, 1)
	node.OnUpdate = func(*engine.SceneNode, float64) {}
	if err := sim.advance(physics.DefaultOptions().FixedStep, engine.SimulationCommands{}); err == nil {
		t.Fatal("conflicting callback accepted")
	}
}

func TestPlaygroundLongRunRemainsBounded(t *testing.T) {
	sim, _, _ := testSimulation(t, demo.PhysicsPlayground)
	for i := 0; i < 1200; i++ {
		sendSimulation(t, sim, physics.DefaultOptions().FixedStep)
	}
	for _, state := range sim.world.Bodies() {
		if state.Type != physics.Dynamic {
			continue
		}
		p := state.Position
		if !(p.Y > -5 && p.Y < 30 && p.X > -50 && p.X < 50 && p.Z > -50 && p.Z < 50) {
			t.Fatalf("unbounded body: %+v", state)
		}
	}
}
