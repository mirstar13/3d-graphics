package physics

import (
	"fmt"
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

// LinearState is a world-space position in metres and velocity in metres/second.
// It is an integration value, not a registered rigid body or scene component.
type LinearState struct{ Position, Velocity engine.Point }

// Integrate applies semi-implicit Euler: update velocity, then position.
// Invalid or overflowing inputs return the original state and an error.
func Integrate(state LinearState, acceleration engine.Point, dt float64) (LinearState, error) {
	if math.IsNaN(dt) || math.IsInf(dt, 0) || dt < 0 || !finitePoint(state.Position) || !finitePoint(state.Velocity) || !finitePoint(acceleration) {
		return state, fmt.Errorf("integration requires finite state and acceleration, and nonnegative finite delta")
	}
	next := state
	next.Velocity.X += acceleration.X * dt
	next.Velocity.Y += acceleration.Y * dt
	next.Velocity.Z += acceleration.Z * dt
	next.Position.X += next.Velocity.X * dt
	next.Position.Y += next.Velocity.Y * dt
	next.Position.Z += next.Velocity.Z * dt
	if !finitePoint(next.Position) || !finitePoint(next.Velocity) {
		return state, fmt.Errorf("integration overflow")
	}
	return next, nil
}
func finitePoint(p engine.Point) bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsNaN(p.Z) && !math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0) && !math.IsInf(p.Z, 0)
}
