# Physics follow-up implementation plan

Status: implementation in progress. Milestones 1–2 are implemented; subsequent milestones
remain queued in the order below. Builds on the completed [rigid-body milestone](rigid-bodies-colliders-plan.md).
This document tracks the remaining physics work.

## Goal and priorities

Make the terminal playground stable, responsive, and useful for more scenes before
adding expensive shape types. Preserve the fixed 60 Hz clock, pause/step semantics,
renderer independence, and deterministic per-tick command ordering.

| Order | Milestone | Value for the terminal renderer | Depends on |
| --- | --- | --- | --- |
| 1 | Stability baselines and solver boundaries | Detect regressions before optimizations hide them. | Current implementation |
| 2 | Contact islands and sleeping | Reduce work and visible jitter in settled scenes. | 1 |
| 3 | Focused continuous collision detection (CCD) | Stop fast spheres crossing thin obstacles. | 1–2 |
| 4 | Allocation reduction and accelerated broad phase | Leave more frame time for rasterization and terminal output. | 1–3 |
| 5 | Render interpolation | Smooth motion when rendering and simulation rates differ. | 1; validate against 2–4 |
| 6 | Physics YAML/settings integration | Make supported behavior tunable without code changes. | 2–5 |
| 7 | Joints | Enable pendulums, chains, doors, and articulated demos. | 1–2; integrate with 3–6 |
| 8 | Compound primitive colliders | Build useful complex objects without dynamic triangle meshes. | 1–7 |
| 9 | Convex hulls and compound convex objects | Support irregular moving assets. | 8 |

Implement in this order, with a buildable, tested commit at each boundary. The first
release target is milestones 1–3. Milestones 4–6 complete the performance and
presentation pass; 7–9 expand scene capabilities. Soft bodies and direct dynamic
concave triangle collision are separate projects, not completion requirements.

## Existing implementation and boundaries

- `internal/physics/solver.go` combines brute-force pair selection, constraints,
  warm starting, position correction, events, and metrics. `detect` runs before
  solving, on every position iteration, and after correction.
- `body.go` owns generation-checked handles and deferred lifecycle operations;
  `dynamics.go` integrates motion. Bodies already retain previous/current poses.
- `clock.go` exposes pre/post hooks and bounded catch-up. CCD subdivisions must
  remain inside one clock tick; they are distinct from catch-up ticks.
- Static triangle meshes have immutable local-space BVHs. Keep using those trees.
- `internal/app/physics_bindings.go` copies authoritative poses into scene nodes.
  Rendering never writes back to body state or determines which bodies simulate.
- Physics options currently come from `DefaultOptions`, not YAML. Configuration
  belongs in config/app composition; do not put physics policy into render adapters.

Keep the simulation single-threaded. Introduce small internal interfaces only at
real interchangeable boundaries, such as broad-phase implementations. Reuse the
contact solver for joint constraints instead of creating another physics engine.

## 1. Stability baselines and solver boundaries

Extract candidate generation, contact caching, and constraint solving from the
combined solver into focused files without changing results. Preserve the current
brute-force broad phase as a test oracle. Define stable pair ordering using full
body handles, including generation; never derive solve order from map iteration.

Add fixtures for longer stacks, supported mass ratios, rolling/sliding over seams,
kinematic crushing, and unsupported initial overlaps. Record maximum penetration,
resting linear/angular speeds, and energy trends against explicit fixture tolerances.
Keep the existing metre-scale defaults; centralize contact margin, penetration
slop, restitution threshold, and correction caps with documented units. Establish
a tested size/mass/speed envelope before advertising arbitrary scene scales.

Measure integration, candidate generation, narrow phase, and solving separately.
Report total pair tests as well as accepted candidates: current CandidatePairs
alone does not reveal the cost of quadratic scans. Add sparse, dense, and settled
fixtures at 16/64/128 bodies, with larger stress cases outside normal tests.
Clarify the World documentation: timing metrics use wall-clock time, but simulation
decisions do not. Preserve snapshot ownership when reusing internal buffers later.

Acceptance: existing tests retain their results; new fixtures have explicit bounds;
benchmarks record work counts, allocations, and physics-only time. Fix demonstrated
instability before enabling sleeping; sleeping must not conceal a failing stack.

## 2. Contact islands and sleeping

Build islands from dynamic bodies connected by non-sensor contacts, later extended
by joints. Static bodies and kinematic bodies act as boundaries, not bridges that
merge every object touching the same floor into one island. Sort traversal and
constraint order by stable handles.

Track sleep eligibility and quiet time per island, in simulation seconds. Start
with tunable trial thresholds of 0.03 m/s linear speed, 0.05 rad/s angular speed,
and 0.5 seconds quiet time; finalize them against milestone 1 fixtures. Require
bounded constraint error as well as low velocities. Expose read-only sleeping
state and awake/sleeping counts. Allow disabling sleeping for comparison.

Wake the affected connected island on nonzero force/impulse, velocity assignment,
teleport, mass/collider/filter/enabled changes, impacts from awake bodies, moving
kinematic support, or removal/relocation of support. New bodies start awake.
Gravity alone must not continually wake a supported sleeping body. Rebuild island
connectivity when contacts disappear; do not leave unsupported objects suspended.

Sleeping bodies remain collision candidates. Skip their integration/constraint
work only when the island can remain asleep. Preserve contact Begin/Stay/End
semantics while sleeping; sensor overlaps still update and do not themselves
form physical islands. Pause freezes sleep timers; stepping advances one tick.

Acceptance: stacks sleep together, remain bounded, and wake correctly when struck
or when support changes. An unrelated impact does not wake the entire scene.
Compare sleeping enabled/disabled behavior and measure saved work in settled scenes.

## 3. Continuous collision detection

Implement opt-in CCD on dynamic spheres first, with an explicit supported-pair
matrix: sphere against static plane, static oriented box, and static triangle mesh.
Use swept AABBs to query candidates and the mesh BVH. Sphere/box sweeps must account
for rounded edge/corner geometry; an expanded box alone is only a conservative
candidate test. Triangle sweeps include face, edge, and vertex impacts.

Compute earliest time of impact (TOI), with stable tie ordering. Separate velocity
integration from pose advancement so forces/gravity/impulses are applied once per
fixed tick. Advance the world coherently to impact, generate contacts, solve, and
process the remaining tick interval. Do not move one body into the future while
solving against other bodies at inconsistent times. Use the actual interval for
position-dependent solving; adapt cached impulses if interval length changes.

Set bounded impact iterations and a minimum progress tolerance. On budget exhaustion,
conservatively stop remaining motion for affected CCD bodies and report it; never
silently advance them through an obstacle. Handle initial overlap through discrete
penetration recovery. Avoid zero-time impact loops at resting contacts.

Hooks, force consumption, sleep timers, interpolation history, and tick counters
run once per outer tick. Buffer transient impacts even when bodies separate before
the tick ends: specify deterministic Begin/End ordering within that tick, with Stay
only for persistent pairs. Define sensor sweeps separately; initially sensors keep
their documented discrete overlap behavior.

Acceptance: fast spheres cannot cross thin walls or mesh faces within the supported
speed envelope; grazing, seams, corners, multiple impacts, and budget exhaustion
are covered. Render rate does not change results. Paused N still advances one tick.

Follow-on CCD scope: relative-motion sphere/sphere, then moving kinematic targets,
then translating/rotating boxes via conservative advancement. Do not claim these
pairs are continuous until each has tests. Rotating boxes require angular swept
bounds; endpoint AABB unions alone are insufficient. Unsupported pairs retain
explicitly documented discrete behavior.

## 4. Allocation reduction and broad phase

Reuse world-owned bounds, candidate, constraint, and event scratch storage. Retain
copying snapshot APIs so callers cannot observe later buffer mutation. Measure
contact-cache reuse separately; avoid clearing useful warm-start data accidentally.

Introduce deterministic sweep-and-prune for finite bodies, with stable endpoint
ties and a separate infinite-plane list. Reuse mesh BVHs in the narrow phase.
Keep the brute-force implementation selectable in tests. Refresh bounds after
position corrections and supply swept bounds to CCD; do not reuse stale candidates
unless their conservativeness is proven.

Acceptance: randomized small-world candidate sets match the brute-force oracle
for rotation, teleports, masks, removals, sleeping, and swept bounds. Contact and
body results remain equivalent within declared tolerances. Show allocation and
pair-test reductions on sparse/settled scenes without material regression in the
small default playground. Dense overlap remains a documented worst case.

## 5. Render interpolation

Interpolate position and shortest-arc quaternion orientation using accumulator
remainder / fixed step. This presents a state up to one tick behind simulation;
document the latency. Apply presentation poses to bound nodes immediately before
render cache preparation, without invoking scene callbacks or updating physics.
Restore authoritative node poses before the next simulation callbacks.

Paused and single-stepped views show the latest completed pose immediately.
Creation, teleport, re-enable, reset, and resume must not blend across discontinuities.
When the collider overlay is enabled, render bound objects at the authoritative
pose too, so debug geometry and contact markers agree with the displayed body.

Acceptance: fractional-alpha endpoint/midpoint checks, quaternion wraparound,
zero-tick frames, multiple-tick frames, pause/step/resume, and teleport tests pass.
Physics snapshots are identical with interpolation on/off across AA and CPU modes.
Do not claim interpolation adds motion samples when output is below simulation FPS.

## 6. YAML and settings screen

Add a nested `physics` configuration group with defaults preserving old YAML files.
Expose gravity, velocity/position iteration counts, sleeping enabled, the default
CCD policy for supported dynamic bodies, and interpolation. Keep advanced sleep
thresholds and CCD budgets in validated YAML first; avoid crowding the TUI.
Use unit labels and bounded numeric ranges; reject nonfinite/invalid combinations.

Retain fixed 60 Hz initially. Distinguish CCD impact budget from the existing
catch-up step limit in naming and help. Use the existing settings save/apply flow;
applying physics settings restarts the scene, clearing caches and sleep state.
Document whether a body-specific CCD setting inherits or overrides the world default.

Acceptance: existing YAML loads unchanged, omitted nested fields retain defaults,
unknown fields fail, save/load round trips, and CLI overrides keep their documented
precedence for exposed flags. PTY tests cover editing, saving, applying, and restart.

## 7. Joints

Implement in three independently testable increments: distance, ball-and-socket,
then hinge. Use generation-checked JointID handles, local anchors, explicit world
anchors, and validated descriptors. Begin with rigid constraints; add compliant
springs, hinge limits, and motors only after basic constraints pass.

Share velocity/position solving, accumulated impulses, warm starting, and deterministic
ordering with contacts. Remove attached joints safely when a body is removed;
reset invalidates old IDs. Joint edits wake connected islands. Offer an explicit
collide-connected option; do not silently disable unrelated contacts. Integrate
joints into CCD interval solving and sleep eligibility.

Acceptance: pendulum, short hanging chain, and hinged door fixtures meet anchor
error and energy bounds; motors obey torque limits and angular limits do not explode.
Test static/dynamic and kinematic/dynamic attachments, stale IDs, removal, pause,
and impact-driven wakeup. Add a small joint demonstration after these gates pass.

## 8. Compound primitive colliders

Allow immutable child spheres/boxes with local rigid poses and stable child IDs.
Keep a single-child descriptor compatible with existing scenes. Combine child
bounds and dispatch child pairs; feature/cache keys include child identity. Limit
and reduce manifolds deliberately without dropping distinct support regions.

Compute centre of mass and the full local inertia tensor using child masses and
the parallel-axis theorem. The current diagonal inertia representation must change
before arbitrary offset/rotated children are supported. Store the visual-to-centre-
of-mass transform in application bindings and update joint-anchor conversion.
Start with one body-level material/filter policy; per-child overrides are separate.

Acceptance: analytic two-mass inertia, off-centre impulses, rotated child shapes,
compound-on-mesh contacts, cache invalidation, and joint anchors all agree. Reject
unsupported compound CCD combinations explicitly. Add a dumbbell or simple vehicle
chassis as a focused visual fixture.

## 9. Convex hulls and moving complex assets

Accept validated, closed convex hull geometry first; automatic hull generation is
a separate import step. Use a support-map narrow phase (GJK distance/intersection,
EPA penetration) with iteration limits and explicit degeneracy handling. Build
stable multi-point manifolds suitable for stacks; one EPA point is insufficient.
Compute volume, centre of mass, and inertia consistently with the compound API.

Acceptance: compare against primitive collision fixtures, test coplanar/duplicate
vertices, thin hulls, containment, reversed pair order, and resting hull stacks.
Add hull/mesh tests before enabling terrain interaction. Extend CCD support only
after hull distance queries and conservative advancement pass dedicated tests.

Represent moving concave-looking assets as authored compound convex parts. Automatic
convex decomposition needs its own asset pipeline and dependency evaluation.
Direct dynamic concave triangle/triangle collision and soft-body simulation remain
outside this plan; they have different robustness and solver requirements.

## Validation and delivery

For each milestone, run focused headless regressions and relevant benchmarks. At
integration boundaries run `go test -race ./...`, `go vet ./...`, Staticcheck, Linux
PTY controls/exit checks, and Windows amd64/macOS arm64 cross-builds. Native terminal
checks remain distinct from cross-compilation. Scope deterministic behavior to
supported builds and numerical tolerances, not bit-identical results across CPUs.

Keep changes in milestone-sized commits, separating larger core/API work from
application integration where both intermediate commits build. Update Go docs,
README controls/settings, and the audit with measured outcomes. Commit and push a
completed implementation milestone before starting the next one, as requested.
