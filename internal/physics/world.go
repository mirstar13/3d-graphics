package physics

import (
	"fmt"
	"math"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// Options defines world-space gravity and the catch-up policy. One physics unit
// is one metre. FixedStep is rounded to nanoseconds by time.Duration.
type Options struct {
	CCDMaxImpacts                                           int
	FixedStep                                               time.Duration
	MaxFrameDelta                                           time.Duration
	MaxSubsteps                                             int
	VelocityIterations, PositionIterations                  int
	Gravity                                                 engine.Point
	Sleeping                                                bool
	SleepLinearThreshold, SleepAngularThreshold, SleepDelay float64
}

// DefaultOptions supplies the initial 60 Hz simulation profile.
func DefaultOptions() Options {
	return Options{CCDMaxImpacts: 8, Sleeping: true, SleepLinearThreshold: .03, SleepAngularThreshold: .05, SleepDelay: .5, FixedStep: time.Second / 60, MaxFrameDelta: 250 * time.Millisecond, MaxSubsteps: 8, VelocityIterations: 20, PositionIterations: 8, Gravity: engine.Point{Y: -9.81}}
}

// World owns simulation timing on the application update goroutine. It has no
// render, input, or scene-node dependencies and is not thread-safe. Wall-clock
// measurements are diagnostic only and never influence simulation decisions.
type World struct {
	bodies       []bodySlot
	pending      []func()
	contacts     []Contact
	events       []ContactEvent
	cache        map[contactKey]cachedImpulse
	contactPairs map[bodyPair]bool
	pairOrder    []bodyPair
	solverStats  SolverStats

	options   Options
	ticks     uint64
	remainder time.Duration
	discarded time.Duration
	paused    bool
	substeps  int
	advancing bool
}

// NewWorld validates options and creates a running world at tick zero.
func NewWorld(options Options) (*World, error) {
	if options.CCDMaxImpacts < 1 {
		return nil, fmt.Errorf("CCD impact budget must be positive")
	}
	if options.FixedStep <= 0 || options.MaxFrameDelta < options.FixedStep || options.MaxSubsteps < 1 || options.VelocityIterations < 1 || options.PositionIterations < 1 {
		return nil, fmt.Errorf("physics requires a positive timestep and substep limit, and frame limit >= timestep")
	}
	if options.MaxFrameDelta > time.Duration(math.MaxInt64)-options.FixedStep {
		return nil, fmt.Errorf("physics timing limits exceed duration capacity")
	}
	if options.Sleeping && (!finite(options.SleepLinearThreshold) || options.SleepLinearThreshold <= 0 || !finite(options.SleepAngularThreshold) || options.SleepAngularThreshold <= 0 || !finite(options.SleepDelay) || options.SleepDelay <= 0) {
		return nil, fmt.Errorf("sleep thresholds and delay must be finite and positive")
	}
	if !finitePoint(options.Gravity) {
		return nil, fmt.Errorf("physics gravity must be finite")
	}
	return &World{options: options}, nil
}

// Gravity returns the immutable acceleration configured for this world.
func (w *World) Gravity() engine.Point { return w.options.Gravity }

// SetPaused clears partial elapsed time when transitioning into pause. Repeated
// pause/resume commands are harmless. The caller must skip elapsed time on resume.
// Control mutations must occur outside simulation callbacks.
func (w *World) SetPaused(paused bool) {
	if w.advancing {
		panic("physics: pause control during callback")
	}
	if paused && !w.paused {
		w.remainder = 0
	}
	w.paused = paused
}

// Reset restores tick-zero, running state with the original world options.
// It must not be called from a simulation callback.
func (w *World) Reset() {
	if w.advancing {
		panic("physics: reset during callback")
	}
	slots := w.bodies
	for i := range slots {
		slots[i].body = nil
		slots[i].generation++
	}
	*w = World{options: w.options, bodies: slots}
}

// Stats is a read-only timing snapshot. SimulationTime is in seconds; Substeps
// counts ticks executed by the most recent nonzero Advance or StepOnce operation.
type Stats struct {
	Tick           uint64
	SimulationTime float64
	FixedStep      time.Duration
	Remainder      time.Duration
	Discarded      time.Duration
	Substeps       int
	Paused         bool
}

// Stats returns timing state without exposing mutable internals.
func (w *World) Stats() Stats {
	return Stats{Tick: w.ticks, SimulationTime: float64(w.ticks) * w.options.FixedStep.Seconds(), FixedStep: w.options.FixedStep, Remainder: w.remainder, Discarded: w.discarded, Substeps: w.substeps, Paused: w.paused}
}
