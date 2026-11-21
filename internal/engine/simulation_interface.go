package engine

// SimulationCommand is an event, independent of held camera movement keys.
type SimulationCommand uint8

const (
	PauseSimulation SimulationCommand = iota
	ResumeSimulation
	StepSimulation
)

// SimulationCommandCapacity bounds both input and pending single-step queues.
// New requests beyond capacity are ignored and counted for diagnostics.
const SimulationCommandCapacity = 64

// SimulationCommands transfers ownership of a drained batch to the application.
// Dropped counts events rejected since the previous drain.
type SimulationCommands struct {
	Commands []SimulationCommand
	Dropped  uint64
}

// SimulationCommandInput is an optional input capability. Draining must be
// atomic and independent of ClearKeys. Backends without it remain compatible.
type SimulationCommandInput interface{ DrainSimulationCommands() SimulationCommands }

// SimulationStatus is a renderer-neutral snapshot for optional status displays.
type SimulationStatus struct {
	Tick            uint64
	Hz              float64
	Paused          bool
	DroppedSeconds  float64
	DroppedCommands uint64
}

// SimulationStatusRenderer optionally displays simulation state outside geometry.
type SimulationStatusRenderer interface{ SetSimulationStatus(SimulationStatus) }
