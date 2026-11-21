package app

import (
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// simulation owns fixed updates for one scene session. The camera and rendering
// remain at frame cadence; node callbacks, animation, lighting and LOD use ticks.
type simulation struct {
	world           *physics.World
	stepBudget      int
	update          func(physics.Tick)
	pendingSteps    int
	droppedCommands uint64
}

func newSimulation(scene *engine.Scene, lighting *engine.LightingSystem, preset int) (*simulation, error) {
	options := physics.DefaultOptions()
	world, err := physics.NewWorld(options)
	if err != nil {
		return nil, err
	}
	return &simulation{world: world, stepBudget: options.MaxSubsteps, update: func(tick physics.Tick) {
		demo.AnimateSceneDemo(scene, preset, tick.Time, tick.Delta)
		demo.AnimateLighting(lighting, preset, tick.Time)
		scene.Update(tick.Delta)
	}}, nil
}

// advance preserves command order. Steps execute up to the per-frame budget;
// excess steps wait while paused, and resume cancels outstanding requests.
func (s *simulation) advance(elapsed time.Duration, batch engine.SimulationCommands) error {
	if elapsed < 0 {
		return s.world.Advance(elapsed, nil)
	}
	s.droppedCommands += batch.Dropped
	stepped := 0
	resumed := false
	step := func() error {
		if err := s.world.StepOnce(s.update); err != nil {
			return err
		}
		stepped++
		return nil
	}
	for _, command := range batch.Commands {
		switch command {
		case engine.PauseSimulation:
			s.world.SetPaused(true)
		case engine.ResumeSimulation:
			if s.world.Stats().Paused {
				resumed = true
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
		return nil
	}
	if resumed {
		return nil
	}
	return s.world.Advance(elapsed, s.update)
}

func (s *simulation) status() engine.SimulationStatus {
	stats := s.world.Stats()
	return engine.SimulationStatus{Tick: stats.Tick, Hz: 1 / stats.FixedStep.Seconds(), Paused: stats.Paused, DroppedSeconds: stats.Discarded.Seconds(), DroppedCommands: s.droppedCommands}
}
