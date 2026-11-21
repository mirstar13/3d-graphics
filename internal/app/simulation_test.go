package app

import (
	"bufio"
	"io"
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

func testSimulation(t *testing.T, index int) (*simulation, *engine.Scene, *engine.LightingSystem) {
	t.Helper()
	scene := engine.NewScene()
	demo.ConfigureCamera(scene.Camera, index, engine.OrientationTerminal)
	demo.BuildScene(scene, index)
	lighting := demo.SetupLighting(scene.Camera, index)
	sim, err := newSimulation(scene, lighting, index)
	if err != nil {
		t.Fatal(err)
	}
	return sim, scene, lighting
}
func sendSimulation(t *testing.T, s *simulation, elapsed time.Duration, commands ...engine.SimulationCommand) {
	t.Helper()
	if err := s.advance(elapsed, engine.SimulationCommands{Commands: commands}); err != nil {
		t.Fatal(err)
	}
}
func TestSimulationPauseStepResumeAndCamera(t *testing.T) {
	sim, scene, lighting := testSimulation(t, 3)
	step := physics.DefaultOptions().FixedStep
	callbacks := 0
	scene.Root.OnUpdate = func(*engine.SceneNode, float64) { callbacks++ }
	sendSimulation(t, sim, step*2)
	before := scene.FindNode("Lantern").Transform.GetWorldMatrix()
	light := lighting.Lights[0].Position
	sendSimulation(t, sim, time.Second, engine.PauseSimulation)
	sendSimulation(t, sim, time.Hour, engine.PauseSimulation)
	if sim.status().Tick != 2 || callbacks != 2 || scene.FindNode("Lantern").Transform.GetWorldMatrix() != before || lighting.Lights[0].Position != light {
		t.Fatal("pause changed simulation")
	}
	controller := engine.NewCameraController(scene.Camera)
	controller.EnableAutoOrbit(false)
	position := scene.Camera.GetPosition()
	controller.Update(engine.InputState{Forward: true}, engine.OrientationTerminal, .1)
	if scene.Camera.GetPosition() == position {
		t.Fatal("paused camera is frozen")
	}
	sendSimulation(t, sim, time.Hour, engine.StepSimulation)
	if sim.status().Tick != 3 || !sim.status().Paused || scene.FindNode("Lantern").Transform.GetWorldMatrix() == before || lighting.Lights[0].Position == light {
		t.Fatal("step did not advance exactly once")
	}
	sendSimulation(t, sim, time.Hour, engine.ResumeSimulation)
	if sim.status().Tick != 3 || sim.status().Paused {
		t.Fatal("resume caught up paused time")
	}
	sendSimulation(t, sim, step, engine.ResumeSimulation)
	if sim.status().Tick != 4 {
		t.Fatal("repeated resume suppresses running time")
	}
	restarted, _, _ := testSimulation(t, 3)
	if restarted.status().Tick != 0 || restarted.status().Paused || restarted.world.Gravity() != physics.DefaultOptions().Gravity {
		t.Fatal("new session retained world state")
	}
}
func TestSimulationCommandOrderAndStepBudget(t *testing.T) {
	sim, _, _ := testSimulation(t, 0)
	commands := []engine.SimulationCommand{engine.StepSimulation, engine.PauseSimulation}
	for i := 0; i < 20; i++ {
		commands = append(commands, engine.StepSimulation)
	}
	sendSimulation(t, sim, time.Second, commands...)
	if sim.status().Tick != 8 || sim.pendingSteps != 12 {
		t.Fatal(sim.status(), sim.pendingSteps)
	}
	sendSimulation(t, sim, time.Second)
	if sim.status().Tick != 16 || sim.pendingSteps != 4 {
		t.Fatal(sim.status(), sim.pendingSteps)
	}
	sendSimulation(t, sim, time.Second, engine.ResumeSimulation)
	if sim.status().Tick != 16 || sim.pendingSteps != 0 {
		t.Fatal("resume failed to cancel pending steps")
	}
	sendSimulation(t, sim, time.Second, engine.PauseSimulation, engine.StepSimulation, engine.ResumeSimulation)
	if sim.status().Tick != 17 || sim.status().Paused {
		t.Fatal("command order lost")
	}
	// A repeated pause must not erase queued steps.
	commands = []engine.SimulationCommand{engine.PauseSimulation}
	for i := 0; i < 100; i++ {
		commands = append(commands, engine.StepSimulation)
	}
	sendSimulation(t, sim, 0, commands...)
	if sim.pendingSteps != engine.SimulationCommandCapacity || sim.status().DroppedCommands != 28 {
		t.Fatal(sim.status(), sim.pendingSteps)
	}
	sendSimulation(t, sim, 0, engine.PauseSimulation)
	if sim.pendingSteps != 56 {
		t.Fatal("pause discarded pending commands")
	}
	before := sim.status()
	if sim.advance(-1, engine.SimulationCommands{Commands: []engine.SimulationCommand{engine.ResumeSimulation}}) == nil || sim.status() != before {
		t.Fatal("negative elapsed consumed commands")
	}
}
func TestRenderingDoesNotAdvanceSimulation(t *testing.T) {
	sim, scene, lighting := testSimulation(t, 0)
	sendSimulation(t, sim, physics.DefaultOptions().FixedStep, engine.PauseSimulation, engine.StepSimulation)
	before := sim.status()
	pose := scene.FindNode("Prism").Transform.GetWorldMatrix()
	for aa := engine.AANone; aa <= engine.AASSAA; aa++ {
		base := terminal.NewTerminalRenderer(bufio.NewWriter(io.Discard), 20, 60)
		base.ShadowRenderer = nil
		renderer := terminal.ComposeRenderer(base, engine.EngineConfig{AAMode: aa, RenderMode: engine.RenderModeParallelTiles, NumWorkers: 2, TileSize: 8})
		renderer.SetLightingSystem(lighting)
		for i := 0; i < 3; i++ {
			renderer.RenderScene(scene)
		}
	}
	if sim.status() != before || scene.FindNode("Prism").Transform.GetWorldMatrix() != pose {
		t.Fatal("rendering changed simulation")
	}
}
