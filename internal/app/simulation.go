package app

import (
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// simulation owns fixed updates for one scene session. The camera and rendering
// remain at frame cadence; node callbacks, animation, lighting and LOD use ticks.
type simulation struct {
	interpolate                bool
	noInterpolationThroughTick uint64
	world                      *physics.World
	stepBudget                 int
	hooks                      physics.StepHooks
	bindings                   *physicsBindings
	failure                    error
	overlay                    bool
	pendingSteps               int
	droppedCommands            uint64
}

func newSimulation(scene *engine.Scene, lighting *engine.LightingSystem, preset int) (*simulation, error) {
	return newConfiguredSimulation(scene, lighting, preset, config.Default().Physics)
}
func newConfiguredSimulation(scene *engine.Scene, lighting *engine.LightingSystem, preset int, settings config.PhysicsSettings) (*simulation, error) {
	options := settings.WorldOptions()
	world, err := physics.NewWorld(options)
	if err != nil {
		return nil, err
	}
	specs, err := demo.PhysicsSpecs(scene, preset)
	if err != nil {
		return nil, err
	}
	bindings, err := newPhysicsBindings(world, scene, specs)
	if err != nil {
		return nil, err
	}
	if err := bindings.addJoints(demo.PhysicsJoints(scene, preset)); err != nil {
		return nil, err
	}
	sim := &simulation{interpolate: settings.Interpolation, world: world, stepBudget: options.MaxSubsteps, bindings: bindings}
	sim.hooks = physics.StepHooks{
		BeforeStep: func(tick physics.Tick) {
			demo.AnimateSceneDemo(scene, preset, tick.Time, tick.Delta)
			demo.AnimateLighting(lighting, preset, tick.Time)
			scene.UpdateCallbacks(tick.Delta)
			if err := bindings.before(tick); err != nil {
				sim.failure = err
			}
		},
		AfterStep: func(tick physics.Tick) { bindings.synchronize(); scene.UpdateVisuals(tick.Delta) },
	}
	return sim, nil
}

// advance preserves command order. Steps execute up to the per-frame budget;
// excess steps wait while paused, and resume cancels outstanding requests.
func (s *simulation) advance(elapsed time.Duration, batch engine.SimulationCommands) error {
	s.bindings.synchronize()
	if elapsed < 0 {
		return s.world.Advance(elapsed, nil)
	}
	s.droppedCommands += batch.Dropped
	stepped := 0
	resumed := false
	step := func() error {
		if err := s.world.StepOnceWithHooks(s.hooks); err != nil {
			return err
		}
		stepped++
		return nil
	}
	for _, command := range batch.Commands {
		switch command {
		case engine.TogglePhysicsDebug:
			s.overlay = !s.overlay
		case engine.PauseSimulation:
			s.world.SetPaused(true)
		case engine.ResumeSimulation:
			if s.world.Stats().Paused {
				resumed = true
				s.noInterpolationThroughTick = s.world.Stats().Tick
				s.world.SetPaused(false)
			}
			s.pendingSteps = 0
		case engine.StepSimulation:
			if !s.world.Stats().Paused {
				continue
			}
			if stepped < s.stepBudget {
				if err := step(); err != nil {
					return err
				}
			} else if s.pendingSteps < engine.SimulationCommandCapacity {
				s.pendingSteps++
			} else {
				s.droppedCommands++
			}
		}
	}
	if s.world.Stats().Paused {
		for s.pendingSteps > 0 && stepped < s.stepBudget {
			if err := step(); err != nil {
				return err
			}
			s.pendingSteps--
		}
		return s.failure
	}
	if resumed {
		return nil
	}
	if err := s.world.AdvanceWithHooks(elapsed, s.hooks); err != nil {
		return err
	}
	return s.failure
}

func (s *simulation) status() engine.SimulationStatus {
	stats := s.world.Stats()
	metrics := s.world.SolverStats()
	return engine.SimulationStatus{Tick: stats.Tick, Hz: 1 / stats.FixedStep.Seconds(), Paused: stats.Paused, DroppedSeconds: stats.Discarded.Seconds(), DroppedCommands: s.droppedCommands, PhysicsBodies: metrics.Bodies, PhysicsAwake: metrics.AwakeBodies, PhysicsSleeping: metrics.SleepingBodies, PhysicsPairs: metrics.CandidatePairs, PhysicsContacts: metrics.Contacts, PhysicsMilliseconds: float64(metrics.Duration) / float64(time.Millisecond), PhysicsOverlay: s.overlay}
}

// prepareRender interpolates only running, non-debug presentation. It never
// advances callbacks or writes presentation state back into physics.
func (s *simulation) prepareRender() {
	stats := s.world.Stats()
	alpha := 1.0
	if s.interpolate && !s.overlay && !stats.Paused && stats.Tick > s.noInterpolationThroughTick {
		alpha = float64(stats.Remainder) / float64(stats.FixedStep)
	}
	s.bindings.present(alpha)
}
