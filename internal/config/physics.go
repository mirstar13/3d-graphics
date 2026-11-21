package config

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// PhysicsSettings is scene-session policy. Applying it creates a new world.
// Distances use metres, time uses seconds, and angular speeds use radians/second.
type PhysicsSettings struct {
	Gravity               engine.Point `yaml:"gravity"`
	VelocityIterations    int          `yaml:"velocity_iterations"`
	PositionIterations    int          `yaml:"position_iterations"`
	Sleeping              bool         `yaml:"sleeping"`
	CCD                   bool         `yaml:"ccd"`
	Interpolation         bool         `yaml:"interpolation"`
	SleepLinearThreshold  float64      `yaml:"sleep_linear_threshold"`
	SleepAngularThreshold float64      `yaml:"sleep_angular_threshold"`
	SleepDelay            float64      `yaml:"sleep_delay"`
	CCDMaxImpacts         int          `yaml:"ccd_max_impacts"`
}

func defaultPhysics() PhysicsSettings {
	o := physics.DefaultOptions()
	return PhysicsSettings{Gravity: o.Gravity, VelocityIterations: o.VelocityIterations, PositionIterations: o.PositionIterations, Sleeping: o.Sleeping, Interpolation: true, SleepLinearThreshold: o.SleepLinearThreshold, SleepAngularThreshold: o.SleepAngularThreshold, SleepDelay: o.SleepDelay, CCDMaxImpacts: o.CCDMaxImpacts}
}
func (s PhysicsSettings) validate() error {
	if !finiteRange(s.Gravity.X, -1000, 1000) || !finiteRange(s.Gravity.Y, -1000, 1000) || !finiteRange(s.Gravity.Z, -1000, 1000) {
		return fmt.Errorf("physics gravity must be finite and within -1000..1000 m/s² per axis")
	}
	if s.VelocityIterations < 1 || s.VelocityIterations > 100 || s.PositionIterations < 1 || s.PositionIterations > 100 {
		return fmt.Errorf("physics iterations must be within 1..100")
	}
	if s.CCDMaxImpacts < 1 || s.CCDMaxImpacts > 64 {
		return fmt.Errorf("physics CCD impact budget must be within 1..64")
	}
	if !finiteRange(s.SleepLinearThreshold, .0001, 10) || !finiteRange(s.SleepAngularThreshold, .0001, 10) || !finiteRange(s.SleepDelay, .01, 60) {
		return fmt.Errorf("physics sleep thresholds must be within 0.0001..10 and delay within 0.01..60 seconds")
	}
	return nil
}

// WorldOptions converts validated settings without involving renderer contracts.
func (s PhysicsSettings) WorldOptions() physics.Options {
	o := physics.DefaultOptions()
	o.Gravity = s.Gravity
	o.VelocityIterations = s.VelocityIterations
	o.PositionIterations = s.PositionIterations
	o.Sleeping = s.Sleeping
	o.DefaultCCD = s.CCD
	o.SleepLinearThreshold = s.SleepLinearThreshold
	o.SleepAngularThreshold = s.SleepAngularThreshold
	o.SleepDelay = s.SleepDelay
	o.CCDMaxImpacts = s.CCDMaxImpacts
	return o
}
