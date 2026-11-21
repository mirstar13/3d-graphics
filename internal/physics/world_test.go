package physics

import (
	"math"
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

func newTestWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func TestRenderRateIndependentGravity(t *testing.T) {
	var reference LinearState
	for _, fps := range []int{30, 60, 144} {
		w := newTestWorld(t)
		state := LinearState{Position: engine.Point{Y: 10}, Velocity: engine.Point{X: 2, Y: 3}}
		total := w.options.FixedStep * 600
		last := time.Duration(0)
		for frame := 1; frame <= fps*10; frame++ {
			now := total * time.Duration(frame) / time.Duration(fps*10)
			err := w.Advance(now-last, func(tick Tick) {
				var err error
				state, err = Integrate(state, w.Gravity(), tick.Delta)
				if err != nil {
					t.Fatal(err)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			last = now
		}
		if w.Stats().Tick != 600 || w.Stats().Discarded != 0 {
			t.Fatal(w.Stats())
		}
		if fps == 30 {
			reference = state
		} else if state != reference {
			t.Fatalf("%d FPS changed integration", fps)
		}
		dt := w.options.FixedStep.Seconds()
		n := 600.0
		want := 10 + 3*n*dt - 9.81*dt*dt*n*(n+1)/2
		if math.Abs(state.Position.Y-want) > 1e-8 {
			t.Fatal(state.Position.Y, want)
		}
	}
}
func TestClockRemainderAndOverload(t *testing.T) {
	w := newTestWorld(t)
	step := w.options.FixedStep
	w.Advance(step-1, nil)
	if w.Stats().Tick != 0 {
		t.Fatal("early tick")
	}
	w.Advance(1, nil)
	if w.Stats().Tick != 1 || w.Stats().Remainder != 0 {
		t.Fatal(w.Stats())
	}
	w.Reset()
	w.Advance(time.Second, nil)
	stats := w.Stats()
	if stats.Tick != 8 || stats.Remainder >= step || stats.Discarded+stats.Remainder+8*step != time.Second {
		t.Fatal(stats)
	}
	before := w.Stats()
	if w.Advance(-1, nil) == nil || w.Stats() != before {
		t.Fatal("negative elapsed mutated state")
	}
	w.Advance(0, nil)
	if w.Stats() != before {
		t.Fatal("zero changed state")
	}
}
func TestPauseStepReset(t *testing.T) {
	w := newTestWorld(t)
	w.Advance(w.options.FixedStep/2, nil)
	w.SetPaused(true)
	if w.Stats().Remainder != 0 {
		t.Fatal("pause retained remainder")
	}
	w.SetPaused(true)
	w.Advance(time.Hour, nil)
	if w.Stats().Tick != 0 || w.Stats().Discarded != 0 {
		t.Fatal(w.Stats())
	}
	w.StepOnce(nil)
	if w.Stats().Tick != 1 || !w.Stats().Paused {
		t.Fatal(w.Stats())
	}
	w.SetPaused(false)
	w.StepOnce(nil)
	if w.Stats().Tick != 1 {
		t.Fatal("stepped while running")
	}
	w.Advance(w.options.FixedStep, nil)
	if w.Stats().Tick != 2 {
		t.Fatal(w.Stats())
	}
	w.Reset()
	if w.Stats().Tick != 0 || w.Stats().Paused || w.Gravity() != DefaultOptions().Gravity {
		t.Fatal(w.Stats())
	}
}
func TestInvalidOptionsAndRecursion(t *testing.T) {
	for _, modify := range []func(*Options){func(o *Options) { o.FixedStep = 0 }, func(o *Options) { o.MaxSubsteps = 0 }, func(o *Options) { o.MaxFrameDelta = 1 }, func(o *Options) { o.Gravity.Y = math.NaN() }, func(o *Options) { o.Gravity.X = math.Inf(1) }} {
		o := DefaultOptions()
		modify(&o)
		if _, err := NewWorld(o); err == nil {
			t.Fatal("accepted invalid options")
		}
	}
	w := newTestWorld(t)
	w.Advance(w.options.FixedStep, func(Tick) {
		before := w.Stats()
		if w.Advance(time.Second, nil) == nil || w.StepOnce(nil) == nil || w.Stats() != before {
			t.Fatal("recursive mutation")
		}
	})
}
func TestIntegrationWithoutGravityAndInvalidInput(t *testing.T) {
	initial := LinearState{Position: engine.Point{X: 1}, Velocity: engine.Point{X: 2}}
	got, err := Integrate(initial, engine.Point{}, .5)
	if err != nil || got.Position.X != 2 || got.Velocity.X != 2 {
		t.Fatal(got, err)
	}
	for _, dt := range []float64{-1, math.NaN(), math.Inf(1)} {
		got, err := Integrate(initial, engine.Point{}, dt)
		if err == nil || got != initial {
			t.Fatal("invalid delta changed state")
		}
	}
}
