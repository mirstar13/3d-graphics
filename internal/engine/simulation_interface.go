package engine

// SimulationCommand is an event, independent of held camera movement keys.
type SimulationCommand uint8

const (
	PauseSimulation SimulationCommand = iota
	ResumeSimulation
	StepSimulation
	TogglePhysicsDebug
	ToggleFrameTiming
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
	PhysicsBodies, PhysicsPairs, PhysicsContacts int
	PhysicsAwake, PhysicsSleeping                int
	PhysicsMilliseconds                          float64
	PhysicsOverlay                               bool
	Tick                                         uint64
	Hz                                           float64
	Paused                                       bool
	DroppedSeconds                               float64
	DroppedCommands                              uint64
}

// SimulationStatusRenderer optionally displays simulation state outside geometry.
type SimulationStatusRenderer interface{ SetSimulationStatus(SimulationStatus) }

// DebugLine describes a transient world-space diagnostic overlay, never a scene node.
type DebugLine struct {
	Start, End Point
	Color      Color
}

// PhysicsDebugRenderer optionally draws diagnostics after ordinary geometry and AA.
type PhysicsDebugRenderer interface{ RenderPhysicsDebug([]DebugLine, *Camera) }
