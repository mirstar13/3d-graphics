# Rigid bodies and colliders: implementation plan

Status: implemented, including static triangle meshes. The sections below record
the original design. This builds on `physics-world-plan.md`.

## Implementation notes

- `internal/physics` owns body handles, integration, collision detection, mesh BVHs,
  contact events, and the impulse solver. Application bindings synchronize poses.
- `AdvanceWithHooks` and `StepOnceWithHooks` expose pre/post phases; the original
  callback methods remain compatible wrappers around the pre-step hook.
- Static meshes keep an immutable local-space BVH. Queries transform into local
  space, so relocating a static body does not require rebuilding its BVH.
- Previous and current poses are retained; rendering uses the latest completed
  pose. Interpolation remains deferred.
- Tolerances target metre-sized scenes: contact margin is 5 mm and position
  correction allows 2 mm penetration. Arbitrary scene scales need further tuning.
- The seventh preset is Physics playground. C toggles an overlay after AA resolve.
  The six existing presets retain their scripted animation.

Validation covers primitive pairs, stacks, restitution, friction, moving platforms,
angular momentum, lifecycle changes, sensors/masks, mesh seams and rotated ramps,
BVH queries against brute force, and matching simulation at 30/60/144 render FPS.
The playground also runs for 1,200 ticks and through Linux PTY pause/step/overlay
controls. Dynamic concave meshes, CCD, joints, sleeping, and interpolation remain
outside this milestone.

## Intended result

A Physics playground scene with dynamic spheres, a static floor and box
obstacles, followed by rotating dynamic boxes and a kinematic platform. Bodies
fall, collide, bounce, slide, and settle. Pause/single-step continues to work;
collider outlines and contact markers make simulation behavior inspectable.

Keep visible geometry separate from collision geometry. Start with spheres,
oriented boxes, and planes. Add static triangle meshes after primitive contacts
and response are reliable. Dynamic concave meshes, general convex hulls, joints,
soft bodies, sleeping, and continuous collision detection are separate work.

## Existing code and boundaries

- `internal/physics.World` already owns fixed timing, gravity, pause, and catch-up.
- `physics.Integrate` provides tested semi-implicit linear integration.
- `internal/app/simulation.go` currently evaluates demo animation, lights, and
  `Scene.Update` in one tick callback. Body integration needs explicit pre/post
  phases so animated targets are prepared before physics and nodes synchronize
  after physics.
- Engine AABB, sphere, OBB, ray, quaternion, and transform helpers are candidates
  for reuse. Test their conventions before relying on them. Boolean overlap
  functions do not provide sufficient information for a contact solver.
- Existing octrees/BVHs store SceneNode pointers and target rendering queries.
  Do not make physics depend on them or on render meshes changing with LOD.

Physics imports engine math types. Engine does not import physics. The application
owns bindings between body IDs and scene nodes; renderers only consume snapshots.
All mutation stays on the simulation/update goroutine. No physics workers yet.

## Stage A — Bodies, lifecycle, and motion

Suggested files: `body.go`, `body_id.go`, `dynamics.go`, and their tests under
`internal/physics`.

### Data and API

Introduce validated BodyDesc input and read-only BodyState snapshots. Runtime
state stays private to World. Use generation-checked BodyID handles so removing
and reusing a slot cannot make an old handle refer to another body.

| Body type | Pose authority | Forces/gravity | Collision response |
| --- | --- | --- | --- |
| Static | Explicit placement/teleport | Ignored | Infinite effective mass; never moved by solver. |
| Dynamic | Physics integration and solver | Applied | Finite positive mass and inertia. |
| Kinematic | Application supplies a target pose per tick | Ignored | Infinite effective mass; its motion can push dynamics. |

State includes position, normalized quaternion orientation, linear velocity,
world-space angular velocity, mass/inverse mass, local inertia/inverse inertia,
force/torque accumulators, linear/angular damping, enabled state, and collider ID.
One physics unit is one metre; velocities use seconds and angular velocity uses
radians/second. The body origin is its centre of mass.

Provide creation/removal, snapshots, teleport, velocity assignment, force,
torque, linear/angular impulse, and kinematic-target operations. Validate finite
values, positive dynamic mass, nonnegative damping, and nonzero quaternions.
Do not expose mutable body pointers or silently accept stale handles.

Initially use one centred primitive collider per body. Dimensions determine
inertia; resizing or changing mass recomputes it and clears cached contacts.
Compound colliders, offset centres of mass, and parallel-axis inertia wait for a
later extension. Shape dimensions are explicit; body poses have no scale.

### Motion rules

- Dynamic linear acceleration is gravity plus force divided by mass.
- Apply semi-implicit Euler; update velocities before predicted poses.
- Angular acceleration uses the inertia tensor transformed into world space.
  Include and test the gyroscopic term for freely rotating asymmetric bodies.
- Integrate orientation using world-space angular velocity and normalize it.
  Verify quaternion multiplication order against engine rotation conventions.
- Use exponential damping `exp(-coefficient * fixedDt)`.
- Forces/torques submitted before a tick apply for that tick and then clear.
  Continuous forces must be submitted each pre-step. Queued impulses apply once
  at a tick boundary, including when stepping from pause.
- Derive kinematic linear/angular velocity from previous and target poses over
  fixedDt. Missing targets mean zero motion. Use the shortest quaternion arc.
- Teleport is explicit and distinct from target motion: it must not generate a
  huge kinematic velocity, and it invalidates contacts and interpolation history.
- Creating/removing bodies is allowed between steps. Requests from callbacks or
  contact observers are queued to the next boundary; never mutate an active pair
  traversal. Removal also clears bindings/contact caches.

Define reset semantics before changing World: Reset returns an empty, running,
tick-zero world with original options and invalidates every old BodyID. Restart
rebuilds body descriptors with the scene. It must not resurrect stale handles.

Deliverable: bodies integrate reproducibly, static poses remain fixed, kinematic
poses follow targets, and angular inertia/force/impulse tests pass. No collision
response is claimed at this stage.

## Stage B — Primitive colliders and contact generation

Suggested files: `collider.go`, `broadphase.go`, `contact.go`, and pair-specific
narrow-phase files.

### Shapes and filtering

- Sphere: positive radius.
- Box: positive local half-extents plus body orientation (an OBB in world space).
- Plane: normalized normal and offset, representing a static half-space.
  Define permitted space as `dot(normal, point) >= offset`; the opposite side is
  solid. Reject planes on dynamic or kinematic bodies initially.

Sphere and box colliders support all three body types. Visual mesh transforms
and LOD changes never alter collider dimensions. Reject scaled physics parents;
start with nodes directly under the identity scene root. Bake desired scale into
collider dimensions and render geometry explicitly at construction.

Add collision layers/masks and optional sensors. A sensor generates overlap
notifications but never solver constraints. Skip disabled bodies, same-body
pairs, filtered pairs, and static-static pairs. Initially generate contacts only
when at least one body is dynamic, except sensor queries explicitly requested.

### Broad phase

Start with deterministic brute-force AABB pair generation for the small demo:
compute conservative rotated-box bounds, test each eligible pair once, and sort
pairs by stable IDs. Keep infinite planes in a separate list. This O(n²) baseline
is an oracle for later sweep-and-prune or a dynamic AABB tree; add an acceleration
structure only after measuring candidate counts and time.

### Narrow-phase support matrix

| Pair | Method | Contact requirement |
| --- | --- | --- |
| Sphere–sphere | Centre distance | Surface point pair; stable fallback normal for coincident centres. |
| Sphere–plane | Signed distance | Projection and radius penetration. |
| Sphere–box | Closest point in box-local coordinates | Handle a sphere centre inside the box explicitly. |
| Box–plane | Signed distances of box corners | Reduce to at most four supporting contacts. |
| Box–box | 15-axis separating-axis test | Face clipping or edge-edge closest points; at most four contacts. |

Handle reversed pair order through one dispatch path. Document a normal from A
toward B, paired world-space surface points, signed separation, and stable feature
IDs. Penetration is negative separation. Touching is within a defined contact
slop. Reject near-zero cross-product SAT axes and use scale-aware tolerances.

Do not substitute a centre-to-centre normal for the box contact manifold. Test
coincident centres, containment, corners, edges, parallel axes, rotated boxes,
large/small dimensions, and reversed pair symmetry.

Deliverable: contact geometry is independently testable and can be visualized;
bodies may still overlap until Stage C enables response.

## Stage C — Contact response and stable resting behavior

Suggested files: `solver.go`, `contact_cache.go`, and solver regression tests.

Implement a sequential impulse solver in a fixed order. Start with configurable
internal iteration defaults (for example 10 velocity and 4 position iterations),
then tune using penetration, resting jitter, and timing measurements.

- Compute contact-point relative velocity including `angularVelocity × offset`.
- Include both inverse masses and world inverse inertias in effective mass.
- Accumulate and clamp normal impulses to nonnegative values.
- Add Coulomb friction with accumulated tangential impulses bounded by the
  normal impulse. Start with one friction coefficient per collider.
- Combine pair friction with geometric mean and restitution with maximum;
  validate friction >= 0 and restitution in [0,1]. Document these choices.
- Apply restitution only above a small closing-speed threshold to avoid bounce
  at rest. Capture the impact velocity before solver iterations.
- Correct residual penetration using slop and capped positional correction,
  without converting the correction into artificial bounce energy.
- Warm-start stable contacts using feature IDs. Invalidate caches on removal,
  teleport, collider change, filtering change, or lost contact.
- Never displace static/kinematic bodies in response to contacts. Include the
  kinematic body's prescribed velocity when solving dynamic contacts.

Discrete contacts can tunnel at high speed. This release documents that limit
and keeps playground speeds/sizes conservative. Increasing render FPS is not a
solution. Swept tests/continuous collision detection are a later milestone.

Deliverable: a falling sphere rests on a plane, restitution behaves predictably,
friction slows sliding, and off-centre impacts rotate bodies. Small box stacks
must remain bounded during long runs; improve manifolds/solver convergence rather
than relying on sleeping to hide instability.

## Stage D — World pipeline and scene integration

Refactor the current callback API into explicit StepHooks with BeforeStep and
AfterStep callbacks. Migrate all Advance/StepOnce callers and tests together.
The existing accumulator, catch-up limits, and pause semantics remain unchanged.
For each fixed tick:

1. Apply deferred lifecycle/impulse commands and preserve previous poses.
2. BeforeStep: evaluate ordinary demo animation and submit kinematic targets and
   continuous forces. Never animate a dynamically bound node.
3. Integrate dynamic velocities and predicted poses; advance kinematic targets.
4. Refresh bounds, find candidates, and generate contact manifolds.
5. Solve velocity constraints and correct penetrations.
6. Clear per-tick force/torque accumulators and publish body/contact snapshots.
7. AfterStep: copy authoritative poses to bound scene nodes, update LODs and
   other post-physics scene behavior, and deliver buffered contact notifications.
8. Rendering prepares caches only after all fixed ticks complete.

Split Scene.Update responsibilities where necessary: its current OnUpdate
callbacks must not run after synchronization and overwrite dynamic transforms.
Define body-aware pre-step hooks separately from post-step visual updates. Use
an explicit binding registry in the application, not tags or names as ownership.

Physics continues to own its poses even if a node is offscreen. Scene traversal
or culling must never decide which bodies simulate. Define disabling a bound
node through the binding API so render-enabled and physics-enabled state do not
diverge silently.

Initially render the latest completed pose. Add previous/current interpolation
only as a separate presentation change after correctness passes; it must never
write an interpolated pose back into physics. Debug contacts correspond to the
authoritative fixed-step state, especially while paused.

Add the Physics playground to the existing catalog without renumbering its six
scenes. Keep the first version small: approximately 12 spheres, a plane, and
three static box obstacles. Add dynamic boxes and a kinematic platform after
the solver gates pass. Use deterministic spawn positions, no runtime downloads,
and no broad configuration UI expansion.

Expose read-only debug snapshots for collider wireframes, contact points and
normals, body count, candidate-pair count, contact count, and physics tick cost.
Draw overlays as a renderer pass after ordinary geometry, avoiding extra scene
nodes that might accidentally enter physics. Add an unused key such as C for
collider/contact overlays; preserve all camera, pause, and settings controls.

## Stage E — Static mesh colliders

This stage follows the primitive playground; it is not required to call the
primitive-body milestone complete.

- Accept explicit indexed triangles as immutable collision geometry. Render
  meshes may supply input once at creation, including baked mesh offsets/scale,
  but later render LOD or animation must not modify that collision data.
- Validate finite vertices, complete index triples, valid indices, and
  nondegenerate triangles. Report invalid input instead of silently using it.
- Restrict these colliders to static bodies. Bake the initial transform into a
  collider-owned representation and build a triangle BVH; explicit relocation
  rebuilds/invalidate bounds and contacts. Ordinary scene-node movement cannot
  move the collider implicitly.
- Implement sphere–triangle using closest points first, then oriented-box–triangle
  SAT and contact clipping. Support both orders through the pair dispatcher.
- Treat the initial mesh as a two-sided surface, not a filled solid. Open meshes
  do not have an interior; an object spawned inside a closed mesh is not
  automatically expelled. Define stable normals for zero-distance contacts.
- Merge/reduce nearby contacts across adjacent triangles and test smooth travel
  across coplanar seams, avoiding duplicate constraints and edge snagging.
- Do not support mesh–mesh or dynamic concave collision in this stage. Future
  moving complex objects use convex hulls or compound convex colliders.

Deliverable: spheres and boxes move over a static ramp/terrain mesh without
falling through at the documented test speeds or catching on coplanar seams.

## Validation gates

Use headless tests with exact fixed ticks; no sleeps or terminal timing in the
physics tests. Define tolerances alongside each fixture's dimensions.

| Area | Required evidence |
| --- | --- |
| Bodies | Invalid descriptor rejection, stale IDs, cleanup/reset, force lifetime, impulse applied once, static/kinematic rules. |
| Dynamics | Analytic/discrete linear checks, known sphere/box inertia, off-centre torque, normalized orientation, damping over equal simulated time. |
| Detection | All supported pair directions, separated/touching/penetrating cases, containment and degeneracy, conservative broad-phase bounds. |
| Solver | Restitution impact, friction slide, no impulses on separating contacts, bounded penetration/resting jitter, short stacks, moving-platform interaction. |
| Determinism | Same initial state and per-tick commands at 30/60/144 render FPS give equivalent state when no time is discarded; stable pair/constraint order. |
| Lifecycle | Safe removal during contact notifications, queued changes, no stale manifolds after teleport or filter changes. |
| Application | Dynamic transforms cannot be overwritten by demos; pause/step/restart/AA/CPU mode do not change tick results. |
| Mesh stage | Invalid assets, closest-point edge/vertex cases, BVH versus brute-force queries, seams, transformed static terrain. |

For performance, benchmark 16, 64, and 128 primitive bodies and report candidate
counts, contacts, allocations, and tick duration separately from rendering.
Aim for a typical demo tick well below 16.67 ms on the reference machine; do not
promise a hardware-independent body count or frame rate.

Run `go test -race ./...`, `go vet ./...`, Staticcheck, existing scene/PTY tests,
and Windows/macOS cross-builds. Native terminal checks remain separate. Keep
numerical determinism claims scoped to supported builds and tolerances rather
than promising bit-identical results across every CPU.

## Suggested implementation commits

1. Body descriptors/IDs, lifecycle, mass/inertia, forces and angular integration.
2. Primitive shapes, filtering, broad phase, and contact-generation tests.
3. Sequential impulses, friction, restitution, contact caching, and resting tests.
4. Step hooks, scene bindings, Physics playground, and debug overlays.
5. Static triangle meshes, triangle BVH, primitive–mesh contacts, and terrain tests.

Build and validate each stage before expanding its scope. The first reviewable
milestone is sphere bodies interacting with a floor and static boxes. Dynamic
boxes and the kinematic platform complete the primitive-body milestone; static
mesh colliders have their own completion gate.
