# Physics world: implementation plan

Status: milestone implemented. This document records the design and scope.

Implementation lives in `internal/physics`, `internal/app/simulation.go`, and
optional engine input/status interfaces. Defaults are centralized in
`physics.DefaultOptions`; the two command queues each hold 64 requests.
The fixed duration is `time.Second / 60` (16,666,666 ns). Scene animations now
use simulation time while cameras remain at frame cadence. Gravity integration
is tested with probes; rigid bodies and automatic scene gravity are not included.

Validation includes deterministic clock/integration tests, pause and command
ordering, concurrent input draining, scene/light freezing, rendering independence,
Linux PTY pause/step/resume with hidden debug help, and platform cross-builds.

## Goal and scope

Introduce a renderer-independent simulation world with a fixed timestep,
gravity, pause/resume, and single-step controls. Rendering may run at a different
rate from simulation. This milestone establishes timing and ownership before
rigid bodies and collision response are added.

Deliver gravity configuration and a small linear-state integration function,
verified with test-only falling probes. Do not add mass, angular dynamics,
colliders, contacts, a solver, or a physics playground yet. Those belong to the
next milestones. Existing curated demos will demonstrate pause and stepping.

## Integration points before implementation

- `internal/app/runner.go` measures elapsed frame time, advances the camera,
  evaluates demos from wall-clock elapsed time, calls `Scene.Update(dt)`, and
  then renders synchronously.
- `internal/engine/scene.go` uses `Scene.Update` for node callbacks and LOD fades.
- `internal/demo` already evaluates poses from an elapsed-time argument.
- `internal/terminal/keyboard.go` stores input in a map and later clears it.
  This is unsuitable for queued single-step commands: events arriving between
  reading and clearing can be lost, and repeated events collapse together.
- `P` opens settings, `R` resets the camera, and X/Escape/Ctrl+C leave the scene.

## 1. Define the simulation world and timing contract

Add `internal/physics/world.go`, `clock.go`, and package documentation.
The application owns one World per scene session. The physics package may use
engine math types; the engine package must not import physics. The world does
not reference renderers, terminal input, scene nodes, or wall-clock APIs.

Initial defaults:

| Setting | Default | Meaning |
| --- | --- | --- |
| Fixed step | 1/60 second | Simulation advances at 60 Hz. |
| Maximum frame delta | 250 ms | Bounds catch-up after stalls or suspension. |
| Maximum substeps | 8 per rendered frame | Keeps input/rendering responsive under overload. |
| Gravity | (0, -9.81, 0) | World-space acceleration; +Y is up. |

Use Go durations for the accumulator and fixed-step comparisons. Compute
simulation time from an integer tick count; convert the fixed duration to
seconds for integration. Validate positive step duration and substep limits,
and finite gravity components. Reject invalid options at construction.

Expose an API along these lines:

- `NewWorld(options)` validates configuration and starts at tick zero.
- `Advance(elapsed, onStep)` runs zero or more fixed ticks, calling the supplied
  update function once per tick with the new tick time and fixed delta.
- `SetPaused(bool)`, `StepOnce(onStep)`, and `Reset()` manage lifecycle.
- Read-only timing statistics expose tick count, simulation time, substeps,
  fractional remainder, paused state, and discarded elapsed time.

Specify these semantics before coding:

1. Clamp excessive elapsed time and record the discarded amount.
2. Add accepted elapsed time to the accumulator.
3. Execute at most eight whole fixed ticks.
4. If whole ticks remain, discard that backlog, record it, and retain only the
   fractional remainder. Never increase the integration timestep to catch up.
5. Under sustained overload, simulation intentionally falls behind real time.
   Equivalent elapsed time produces equivalent ticks only when no time is dropped.
6. Negative elapsed input is rejected without changing state. Zero does nothing.

The callback is synchronous, runs on the update goroutine, and must not call
Advance/StepOnce recursively. A callback keeps the clock testable without
introducing a general-purpose system scheduler.

## 2. Establish gravity integration and transform ownership

Add `internal/physics/integration.go` with a minimal linear state containing
world-space position and velocity, plus a semi-implicit Euler integration
function:

    velocity += acceleration * dt
    position += velocity * dt

World gravity supplies acceleration; use seconds and document one physics unit
as one metre. Existing decorative demo dimensions need not be reinterpreted as
physically accurate objects. Gravity does not automatically move scene nodes.

For this milestone, exercise integration with test-only probes attached to the
fixed-step callback. Production body registration and node synchronization wait
for rigid bodies. This avoids introducing a temporary particle API to replace
immediately afterward.

Document the future ownership contract now: physics owns the authoritative pose
of dynamic bodies. A synchronization phase writes that pose to scene transforms
before render-cache preparation. Demo animation must not also write those poses.
Begin with unparented dynamic bodies; transformed-parent support is a separate
rigid-body design decision.

## 3. Integrate the fixed clock into the application

Add `internal/app/simulation.go` to encapsulate session timing and command handling.
Update runner ordering to:

1. Measure elapsed frame time and consume input/navigation commands.
2. Update camera movement and terminal dimensions at render cadence.
3. Process pause/resume/step commands.
4. Advance the simulation clock. For each tick, evaluate demo animation and
   animated lighting using simulation time, then call `Scene.Update(fixedDt)`.
5. Render and present the latest completed state after all updates finish.

Stop using `time.Since(startTime)` as the demo animation clock. Otherwise paused
animations would jump forward on resume. Node callbacks and LOD transitions
advance on simulation ticks too. The camera remains usable while paused; LOD
selection stays frozen until the next simulation tick.

No physics goroutine is needed. Existing rendering workers finish before the
next update, maintaining the current transform-cache ownership rules. Physics
must never run in a renderer, tile worker, or AA sampling pass.

Frames with no simulation tick render the previous state. Pose interpolation
is deferred until rigid bodies provide previous/current authoritative poses;
this milestone may show 60 Hz motion on a faster display. Keep the fractional
remainder available for that later interpolation work.

## 4. Add reliable pause and step commands

Use separate, idempotent commands to avoid repeated toggle events:

| Key | Action |
| --- | --- |
| Space | Pause simulation. |
| Enter | Resume simulation. |
| N | Advance one fixed tick while paused. |

N while running does nothing. Each queued N event while paused produces one
step; holding N may repeat according to the terminal's key-repeat behavior.
Bound command storage and process at most eight step requests per frame;
retain unprocessed requests while paused. Resume clears queued step requests.
Document that input beyond the queue capacity is ignored.

Drain commands atomically under the input mutex, independently of ClearKeys.
Keep their order so pause, step, and resume are unambiguous. Add an optional
simulation-command input interface alongside InputManager, preserving existing
backend implementations. If unsupported, the application receives no commands.

Pausing clears the fractional accumulator. Resuming starts with no backlog and
skips the current frame's elapsed time, since part of it was spent paused.
Single-step advances time exactly once without unpausing or altering the normal
accumulator. Restarting a scene creates a fresh world. Entering settings retains
the application's existing behavior of ending the scene session.

Show RUNNING/PAUSED, simulation Hz, and tick count in the terminal status area.
If steps or elapsed time are dropped, expose that in debug statistics. Display
pause/step help while paused even when normal debug statistics are hidden.

## 5. Validate behavior

Unit tests, with supplied elapsed durations and no sleeping:

- Equal total elapsed time split into 30/60/144 FPS-sized chunks yields equal
  tick counts and integrated state when catch-up limits are not reached.
- Fractional remainder carries across frames without an early or missed tick.
- A long stall respects both elapsed-time and substep limits, reports dropped
  time, and leaves less than one tick of backlog.
- Pause preserves state; repeated pause is harmless; resume never catches up
  paused time; single-step advances one tick and remains paused.
- Negative elapsed values and invalid world options fail without mutation.
- Gravity integration matches the discrete semi-implicit Euler result, including
  zero gravity and nonzero initial velocity. Do not require exact agreement with
  the continuous free-fall equation at a finite step size.
- Command draining preserves event order and commands arriving after a drain;
  test queue capacity, pending steps, and resume behavior under the race detector.

Integration tests:

- Demo transforms and light positions stop while paused and advance on N.
- Camera input still works while paused.
- Multiple renders and AA samples never advance the simulation clock.
- Scene restart resets timing and gravity state.
- Extend the PTY script for Space, N, Enter, settings return, resize, and exit.

Run the existing scene/rendering suite, `go test -race ./...`, `go vet ./...`,
Staticcheck, and Windows/macOS cross-builds. Native console validation remains
separate from cross-compilation.

## Delivery sequence and completion criteria

1. World clock, gravity integration, documentation, and deterministic unit tests.
2. Application fixed-step integration with simulation-time animation tests.
3. Queued terminal controls, status display, and PTY regression coverage.

Keep timestep and catch-up defaults in one options constructor initially.
Do not expand the YAML/settings editor until the scheduling contract is stable.

This milestone is complete when simulation timing is independent of render FPS
within the documented overload policy, gravity integration is verified, and
pause/step/resume work without animation jumps or lost control events. It does
not claim rigid-body or collision simulation is available.
