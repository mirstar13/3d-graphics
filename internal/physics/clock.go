package physics

import (
	"fmt"
	"math"
	"time"
)

// Tick describes the new simulation state time. Time and Delta are seconds.
type Tick struct {
	Index       uint64
	Time, Delta float64
}

// StepHooks separate commands/kinematic targets from post-solve scene synchronization.
type StepHooks struct{ BeforeStep, AfterStep func(Tick) }

// Advance is a compatibility entry point whose callback runs before body integration.
func (w *World) Advance(elapsed time.Duration, update func(Tick)) error {
	return w.AdvanceWithHooks(elapsed, StepHooks{BeforeStep: update})
}

// AdvanceWithHooks accepts elapsed render time and invokes hooks synchronously for each
// fixed tick. Excess elapsed time is discarded, leaving a fractional remainder.
// Negative elapsed and recursive advancement fail without changing world state.
// Zero elapsed is a no-op. Empty hooks still integrate bodies and solve contacts.
func (w *World) AdvanceWithHooks(elapsed time.Duration, hooks StepHooks) error {
	if elapsed < 0 {
		return fmt.Errorf("physics elapsed time must not be negative")
	}
	if w.advancing {
		return fmt.Errorf("physics callback cannot advance recursively")
	}
	if elapsed == 0 {
		return nil
	}
	w.substeps = 0
	if w.paused {
		return nil
	}
	if elapsed > w.options.MaxFrameDelta {
		w.discard(elapsed - w.options.MaxFrameDelta)
		elapsed = w.options.MaxFrameDelta
	}
	w.remainder += elapsed
	w.advancing = true
	defer func() { w.advancing = false }()
	for w.remainder >= w.options.FixedStep && w.substeps < w.options.MaxSubsteps {
		w.remainder -= w.options.FixedStep
		w.tick(hooks)
	}
	backlog := w.remainder / w.options.FixedStep * w.options.FixedStep
	w.discard(backlog)
	w.remainder -= backlog
	return nil
}

// StepOnce is a compatibility entry point for a pre-integration callback.
func (w *World) StepOnce(update func(Tick)) error {
	return w.StepOnceWithHooks(StepHooks{BeforeStep: update})
}

// StepOnceWithHooks executes exactly one tick while paused, leaving the accumulator and
// pause state unchanged. Calls while running are ignored.
func (w *World) StepOnceWithHooks(hooks StepHooks) error {
	if w.advancing {
		return fmt.Errorf("physics callback cannot step recursively")
	}
	if !w.paused {
		return nil
	}
	w.substeps = 0
	w.advancing = true
	defer func() { w.advancing = false }()
	w.tick(hooks)
	return nil
}
func (w *World) tick(hooks StepHooks) {
	pending := w.pending
	w.pending = nil
	for _, apply := range pending {
		apply()
	}
	w.ticks++
	w.substeps++
	tick := Tick{Index: w.ticks, Time: float64(w.ticks) * w.options.FixedStep.Seconds(), Delta: w.options.FixedStep.Seconds()}
	if hooks.BeforeStep != nil {
		hooks.BeforeStep(tick)
	}
	w.stepBodies(tick.Delta)
	if hooks.AfterStep != nil {
		hooks.AfterStep(tick)
	}
}
func (w *World) discard(duration time.Duration) {
	// Saturate the diagnostic counter instead of wrapping after repeated stalls.
	if duration > time.Duration(math.MaxInt64)-w.discarded {
		w.discarded = time.Duration(math.MaxInt64)
	} else {
		w.discarded += duration
	}
}
