package app

import (
	"math"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

func TestInterpolationIsPresentationOnly(t *testing.T) {
	sim, _, _ := testSimulation(t, demo.PhysicsPlayground)
	step := physics.DefaultOptions().FixedStep
	sendSimulation(t, sim, step+step/2)
	before := sim.world.Bodies()
	binding := sim.bindings.items[1]
	state, _ := sim.world.Body(binding.id)
	sim.prepareRender()
	want := (state.PreviousPosition.Y + state.Position.Y) / 2
	if math.Abs(binding.node.Transform.Position.Y-want) > 1e-8 {
		t.Fatal("midpoint not presented", binding.node.Transform.Position, want)
	}
	if !reflect.DeepEqual(before, sim.world.Bodies()) {
		t.Fatal("presentation changed world")
	}
	sim.advance(0, engine.SimulationCommands{})
	if binding.node.Transform.Position != state.Position {
		t.Fatal("authoritative pose not restored before callbacks")
	}
	sim.overlay = true
	sim.prepareRender()
	if binding.node.Transform.Position != state.Position {
		t.Fatal("debug geometry disagrees")
	}
	sim.overlay = false
	sendSimulation(t, sim, 0, engine.PauseSimulation)
	sim.prepareRender()
	if binding.node.Transform.Position != state.Position {
		t.Fatal("paused pose interpolated")
	}
	sendSimulation(t, sim, 0, engine.StepSimulation)
	sim.prepareRender()
	state, _ = sim.world.Body(binding.id)
	if binding.node.Transform.Position != state.Position {
		t.Fatal("single step interpolated")
	}
	sendSimulation(t, sim, 0, engine.ResumeSimulation)
	sim.prepareRender()
	if binding.node.Transform.Position != state.Position {
		t.Fatal("resume blended old history")
	}
}
func TestInterpolationTeleportAndQuaternionWrap(t *testing.T) {
	sim, _, _ := testSimulation(t, demo.PhysicsPlayground)
	binding := sim.bindings.items[1]
	q := engine.QuaternionFromEuler(0, math.Pi-.01, 0)
	if err := sim.world.Teleport(binding.id, engine.Point{Y: 20}, q); err != nil {
		t.Fatal(err)
	}
	sim.bindings.present(.5)
	if binding.node.Transform.Position.Y != 20 {
		t.Fatal("teleport smeared")
	}
	opposite := engine.Quaternion{W: -q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
	result := q.Slerp(opposite, .5)
	v := result.RotateVector(engine.Point{Z: 1})
	want := q.RotateVector(engine.Point{Z: 1})
	if math.Abs(v.X-want.X) > 1e-8 || math.Abs(v.Z-want.Z) > 1e-8 {
		t.Fatal("long quaternion arc")
	}
}
