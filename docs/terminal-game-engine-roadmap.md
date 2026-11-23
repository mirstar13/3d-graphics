# Terminal game engine: implementation roadmap

Status: the first editor vertical slice, hierarchy authoring, and basic linked
scene resources, semantic input bindings, and keyboard query selection are implemented.
See the [resource milestone](editor-resources-plan.md) and
[input/selection milestone](editor-input-selection-plan.md).
See the [hierarchy milestone](editor-hierarchy-plan.md). See the
[editor guide](editor-first-milestone.md) for delivered scope and tests. Remaining
stages below are planned. Builds on the completed [physics follow-up work](physics-follow-up-plan.md).
This is a staged product and architecture plan. The initial slice provides
primitive documents with nested visual hierarchies and root-level physics bodies,
project creation/opening, undoable property and subtree edits, resource extraction,
instantiation, source editing, refresh, explicit overrides, and unpacking,
a monochrome viewport, and in-process isolated-state Play/Stop. It does not complete
all stages 0–2 or provide nested linked scenes, scripting, export, and public APIs.

## Product vision: an integrated terminal-game authoring environment

Build an engine and editor with a workflow analogous to Godot or Unity:
create/open a project, assemble scenes, inspect objects, attach gameplay behavior,
press Play, debug, and export a standalone terminal game. Creators should not need
to edit engine source or hand-write scene YAML to perform ordinary authoring.

The product consists of three independently testable parts:

- **Runtime:** scenes/entities, gameplay, input, rendering, physics, UI, assets,
  and lifecycle. Exported games run without the editor or its dependencies.
- **Editor:** project management, scene hierarchy, viewport, inspector, asset
  browser, reusable scenes, undo/redo, play controls, and debugging.
- **Project/build tools:** scene and resource formats, behavior compilation,
  dependency/version tracking, templates, and export profiles.

The terminal is the game's output target. That does not require the editor itself
always to be a terminal UI. Use the existing TUI for the first editor shell, but
keep document operations and editor services independent of presentation so a
graphical frontend can be evaluated later without rewriting the runtime.

The initial game target remains single-player and keyboard-first using the existing
3D renderer. Physics is optional per object. Unity/Godot describe the intended
creator workflow, not a promise of feature parity. Networking, a custom scripting
language, soft bodies, and additional render backends are separate later decisions.

Success means a creator can assemble and customize a complete game in the editor,
add project-local gameplay code, and export it without modifying the engine.
The sandbox becomes the editor's first working viewport, not the final product.

## Revised delivery order

| Stage | Deliverable | Completion evidence |
| --- | --- | --- |
| 0 | Runtime, document, and editor-service boundaries | The editor and demos use the same runtime without demo-specific ownership. |
| 1 | Project manager and editor shell | Create/open a project; hierarchy, inspector, viewport, assets, and play controls work together. |
| 2 | Scene authoring and reusable scene resources | Save/load, undo/redo, instantiate reusable scenes, and isolate edit/play state. |
| 3 | Gameplay attachment, public API, and export | A project-local behavior runs through Play and in an exported sample game. |
| 4 | Focused demos and project templates | Demonstrations become editable examples built through the same creator workflow. |
| 5 | Native compatibility and measured performance | Published native results, profiles, and reproducible benchmarks. |
| 6 | Targeted physics extensions | CCD, joint stress, and scaling improvements address demonstrated game needs. |

This supersedes the earlier SDK/sandbox-first ordering: project/editor/document
foundations move ahead of more demonstrations and advanced physics. Carry the
interaction tools into stage 1 rather than building a separate throwaway sandbox.
Run correctness, compatibility, and performance checks throughout every stage;
stages 5–6 deepen that work rather than delaying basic quality checks.

Commit each tested milestone in buildable units and push before starting the next.
Keep this roadmap updated with measured results and deliberate scope changes.

## 0. Establish the reusable runtime boundary

Current constraints:

- `internal/app/physics_bindings.go` accepts runtime-owned `game.BodySpec` and
  `game.JointSpec`; demos alias these descriptions. The demo runner still uses
  application bindings, while authored projects use `game.Session`.
- `internal/app/simulation.go` constructs a preset and schedules its callbacks.
- Renderer/input contracts live in `internal/engine`; Go prevents external projects
  from importing these packages directly.
- Persisted scene selection is currently a one-based index into `demo.Presets`.

Add an editor document model alongside the runtime. Document edits are commands
with validation and undo data; they are not direct writes into a running World.
Use property metadata (type, units, range, serialization key, editability) shared
by the inspector and serializer instead of adding another hard-coded field table.
Model authored objects as a hierarchy with optional components; this is the creator
model, without requiring an ECS storage rewrite. Keep the current physics-root
scale/parent restrictions explicit in editor validation.

Extract plain entity/body/joint descriptions and session ownership into a small
runtime package. Demos become clients producing those descriptions; the launcher
owns menus and terminal lifecycle. Avoid a wholesale ECS rewrite. Begin with
explicit optional components: transform, visual, body, and registered behavior.

Define these contracts before expanding the editor:

- Runtime EntityID identifies a live entity; generation checking rejects stale
  references. It is independent of physics BodyID and scene-file persistent IDs.
- A session owns entities, body/joint bindings, selection, world time, and lifecycle.
  An entity can be visual-only; render nodes are not the entity registry.
- Runtime commands create/remove/edit entities and constraints through a queue.
  Apply them at safe simulation boundaries in deterministic submission order.
  Publish events after solving; event callbacks enqueue future mutations.
- Retain physics pose authority and explicit visual-to-mass-centre offsets.
  Rendering/interpolation never changes gameplay state. Disabled, hidden, destroyed,
  static, kinematic, and sleeping states remain distinct concepts.
- Behaviors receive documented lifecycle hooks: spawn, fixed update, events, and
  removal. Presentation updates cannot advance physics. Start with registered Go
  behaviors and typed settings; no embedded language or arbitrary file callbacks.
- Offer a headless session runner with a fake clock and action source for tests.
  Renderer and input adapters are supplied at composition time.

Keep the 60 Hz simulation/catch-up contract. Input actions are resolved once per
input batch and assigned to ticks; events and one-shot actions are not replayed
for every catch-up tick. Define command ordering with pause/resume/single-step.

Acceptance: existing presets still run; entity removal cleans up bodies, attached
joints, selection, and visuals; restart invalidates old handles. Runtime tests do
not require a terminal, demo package, or rendering workers.

## 1. Project manager and editor shell

Add a project manifest (`project.yaml`) with project identity, schema/engine
compatibility versions, entry scene, asset roots, action mappings, and export
profiles. Keep machine-local editor layout and recent projects separate. Create
projects from templates; validate projects before replacing the active workspace.

The first workspace contains a scene viewport, hierarchy, component/property
inspector, asset browser, output/debug panel, and Save/Play/Pause/Step/Stop controls.
Support creating, renaming, reparenting, duplicating, and deleting objects through
validated document commands. Property metadata drives inspector controls and errors.
Use explicit keyboard focus and switchable panels when the terminal is narrow.
Mouse interaction can follow without becoming a requirement for authoring.

Deliver one complete vertical slice: create project, add a visual/body, edit a
property, save, Play, apply an impulse, Stop, and reopen the unchanged authored
scene. Implement the minimum stage-2 document format for this slice; do not create
a temporary persistence format. The viewport tools below become editor tools.


### Input actions and focus

Introduce semantic actions, such as SelectNext, ApplyImpulse, SpawnBox, and Reset,
with configurable bindings. Inventory current camera, simulation, and menu keys
before assigning defaults. Publish context-sensitive help from the binding table.

Use explicit focus contexts: viewport, inspector, command input, and menus. Text
editing consumes its keys and cannot accidentally move the camera, spawn bodies,
or quit the application. Every overlay can be dismissed using a documented key.

Terminal key-release and modifier reporting varies. Do not invent reliable held-key
state from ordinary terminal repeat events. Initial editing actions are press-based;
continuous movement uses a documented repeat/timeout policy. Rich keyboard protocols
and mouse capture are optional capabilities with a keyboard fallback. Provide
rebindable movement axes to later games through the same action system.

### Selection and physics queries

Implement read-only world queries before manipulation: closest ray hit and overlap
queries with layer/sensor filters. Return body/entity identity, world point, normal,
distance, and compound child identity where relevant. Reuse static mesh BVHs and
validate against brute force. Specify rays starting inside shapes and stable ties.

Selection must support cycling through selectable entities and selecting beneath a
centre-screen reticle. Do not require mouse support. Use the camera's existing ray
conventions; convert terminal-cell coordinates consistently with FOV and viewport.
Select collision geometry initially, with explicit bounds proxies for visual-only
entities. Document that thin visual features may not match their collider exactly.

When editing or selecting precisely, use authoritative poses (pause or temporarily
suppress interpolation), so the hit test and highlighted object agree. Read-only
selection must not wake a sleeping body.

### Commands and inspector

Deliver incrementally:

1. Select/deselect and show name, type, position, velocities, mass, and sleep state.
   Highlight through the diagnostic rendering pass, without adding collision shapes.
2. Apply a configurable impulse at the selected hit point or centre of mass.
   Show the direction and magnitude before application. Off-centre impulses rotate.
3. Spawn spheres/boxes from presets at the reticle or a bounded distance ahead.
   Check placement overlap, support cancellation, and enforce a configurable entity
   budget with a clear message. Use deterministic spawn parameters.
4. Delete selected runtime-spawned objects, with undo where practical. Mark required
   demo scaffolding explicitly instead of deciding deletability by entity names.
5. Reset to the scene's authored initial state; selection and pending commands clear.
6. Edit validated scalar/vector properties. Changes to shape/mass rebuild inertia,
   invalidate contacts, and wake affected islands through runtime APIs.

Initially queue simulation-affecting commands until the next tick, including while
paused. Show pending edits; N applies them and advances exactly one tick. Stage 2
adds a separate authoring mode where document edits need not advance simulation.
Do not overload paused gameplay with ambiguous editor semantics.

Acceptance: every action works by keyboard; no action leaks through an inspector;
spawn/delete/restart cannot leave stale bindings; frame rate, CPU mode, or AA do
not change a given tick-indexed command sequence. Test small terminals, resizing,
repeated commands, pending edits while paused, and normal/error exits.

## 2. Scene documents, save/load, and authoring

### Separate scene content from application preferences

Keep `renderer.yaml` for terminal/render preferences and default simulation settings.
Use a separate versioned YAML scene document for authored content. Define precedence:
engine defaults, application defaults, scene simulation overrides, then explicit
session overrides. Save only the intended document layer, not an accidental mixture
of machine preferences and scene content.

Version 1 contains:

- Schema version, scene identifier/name, optional deterministic seed, and metadata.
- Persistent entity identifiers independent of names, slice indices, or live handles.
- Transforms, primitive/asset visuals, materials, lights, and camera starting pose.
- Optional body descriptions, immutable collider definitions, material/filter rules,
  CCD policy, and explicit body/visual offsets. Preserve current hierarchy/scale
  restrictions for physics-bound roots; validate rather than silently baking edits.
- Joints referencing persistent entity IDs, local/world anchors, limits, and motors.
- Behavior type identifiers and validated data interpreted by a known registry.
  Replace existing function-valued animation/target callbacks with registered
  behavior descriptions for serializable scenes; never serialize function pointers.
- Project-relative asset references. Start with procedural primitives and supported
  local meshes/materials; handle missing assets and paths with contextual errors.

Do not serialize BodyID/JointID, caches, sleep timers, interpolation history,
renderer buffers, or closures. Saving an authored scene is different from saving
a running game. Exact gameplay save states are a later versioned feature.

### Reusable scenes and resource identity

Support saving a subtree as a reusable scene resource, then instantiating it into
another scene. Use stable resource IDs and explicit references; names are labels.
Duplicate operations remap entity IDs and internal joint/behavior references.
Validate recursive scene dependencies and missing resources before construction.

Begin with linked scene instances and explicit editable instance properties.
Define how source changes and instance overrides interact; never silently replace
an instance's edits on reload. Full inherited-scene variants and complex nested
override merging follow tested basic instancing. Materials and collider definitions
can be shared resources without sharing mutable per-instance gameplay state.

### Loading, editing, and play mode

Parse and validate a complete scene before replacing the active session. Validate
unique IDs, references, hierarchy cycles, supported component combinations, finite
values, geometry sizes, and behavior settings. Construct in a staging session;
commit only after all entities, assets, and joints are ready. On failure, preserve
the current session and report the entity/property involved.

Use atomic file replacement, stable serialization order, and explicit errors for
unknown future versions. Migrate supported older versions in small tested steps;
do not silently discard fields. Round trips preserve semantic content rather than
promising preservation of comments or exact YAML formatting.

Introduce an edit/play distinction:

- Edit mode operates on a scene document, with selection, property editing, duplicate,
  spawn/delete, and a bounded undo/redo history of document commands.
- Play creates a runtime copy. Stop restores the authored document. Runtime impulses,
  destruction, and gameplay state do not silently modify the saved scene.
- An explicit “capture current poses” command can update authored transforms after
  preview, with clear treatment of body/visual offsets and joint anchors.

Acceptance: save/load/restart reconstructs equivalent authored bodies and joints;
failed loads/saves preserve existing state/files; duplicate names work with unique
IDs; missing references, unknown behaviors/versions, and malformed geometry fail
clearly. Test stale handles after load, packaged relative assets, undo/redo, and
edit/play isolation. Reopen a scene in a separate process, not only in one test.

## 3. Gameplay attachment, public API, and export

Expose project-local Go behaviors as attachable components with registered types
and inspectable properties. The inspector lets creators select a behavior and
edit its data; a source editor command opens the file in their preferred editor.
Go is the initial gameplay language because it matches the current engine. This
does not rule out a later scripting language, but avoids adding a VM before the
lifecycle and authoring APIs are proven.

Define component lifecycle, action access, collision/trigger events, and entity
lookup without depending on scene-node names. Start with typed event subscriptions;
add inspector-configured event connections only with validated targets/signatures.
Expose a read-only live hierarchy and inspector during Play. Runtime modifications
are discarded on Stop unless an explicit supported operation copies them back.

Play compiles project behaviors and launches a supervised game session, preferably
in a child process so a game panic cannot take down the editor. Report build/runtime
errors with source locations; define pause/step/stop and diagnostic communication.
Avoid Go dynamic plugins as the cross-platform loading contract. Initial iteration
is rebuild/restart, not promised hot reload; preserve the authored document across
build failures and runtime crashes.

Export profiles choose target OS/architecture, starting scene, bundled assets, and
default input/display settings. A development export can retain diagnostics; a
release export starts the game directly with no editor. Clearly distinguish
cross-compilation success from native runtime validation.


Use experience from stages 0–2 to publish a small supported Go API outside
`internal`. Proposed surfaces: application/session creation, entities/components,
actions, scene loading, behavior registration, physics queries/events, renderer
selection, and HUD composition. Final package names are an implementation decision.
Do not expose all internal solver/render data structures as a compatibility promise.

The SDK must support:

- A direct game entry point without the repository's launcher or editor.
- Fixed gameplay updates and read-only presentation, with predictable action/event
  ordering and documented creation/destruction semantics.
- HUD text/panels in terminal cells composed after the world and AA. UI owns focus,
  clips to the viewport, remains legible in monochrome, and does not enter physics.
- Scene transitions, pause, restart, and clean error/exit handling.
- Embedded or project-relative assets so a distributed binary does not depend on
  the repository working directory. Document supported build targets and settings.
- Headless tests using fake input/clock and deterministic scene seeds.

Provide a template and one small “knock down targets” game: aim/fire, collision
scoring, limited ammunition or timer, HUD, win/lose screens, restart, and a loaded
scene. Use contact Begin events carefully so resting contacts do not score every
tick. Destroying a target safely updates gameplay, physics, selection, and visuals.

Build this sample in a separate Go module importing the SDK through its public
module path (a local replacement is fine in development). This catches accidental
internal imports and undocumented setup dependencies. Add a clean-checkout build
and packaged launch test. A second lightweight visual-only example verifies that
physics and the editor are optional.

Acceptance: someone following the quickstart can build and distribute the sample,
change a scene without editing engine code, and add a behavior through documented
APIs. Publish Go docs, examples, capability limitations, and a pre-1.0 compatibility
policy. API versioning and scene-schema versioning are independent contracts.

## 4. Focused demonstration suite

Ship the sandbox/playground as an editable example project. Add focused scenarios using
the same scene descriptions, action mapping, and runtime commands:

| Scene | Demonstration | Failure made visible |
| --- | --- | --- |
| Stack and slide | Resting boxes, contrasting friction, sleep/wake | Penetration, drift, incorrect wake propagation. |
| Pendulum and door | Distance/ball/hinge constraints, limits and motors | Anchor error, limit violation, excessive energy. |
| Fast projectile | Supported CCD versus discrete bodies against thin obstacles | Tunneling, impact-budget clamps, missed transient events. |
| Shape workshop | Primitive, compound, and authored convex bodies | Incorrect mass centres, inertia, or collider/visual alignment. |

Give each a concise description, stable seed, reset action, and useful camera.
Display relevant metrics without covering the demonstration. Let users toggle the
feature being demonstrated, with a restart when needed for a fair comparison.

Introduce stable string scene identifiers before expanding the catalog. Preserve
existing numeric CLI/YAML scene selections through a compatibility mapping; never
silently remap old numbers when reorganizing the picker.

Acceptance: headless fixtures assert the intended behavior, all CPU/AA modes can
render the scenes, geometry budgets remain explicit, and monochrome views retain
meaning without depending entirely on color. Include preview generation and a
manual checklist describing what a creator should observe.

## 5. Native compatibility and performance

Build a repeatable native matrix covering Windows 10/11 with PowerShell in Windows
Terminal and VS Code, Linux terminal emulators, and macOS Terminal/iTerm2. Record
actual versions and mark unavailable targets untested; cross-builds prove compilation
only. Include narrow windows, resize, focus loss, color modes, input repetition,
settings transitions, long sessions, clean shutdown, and interrupted runs.

Retain the documented VS Code GPU-rendering workaround. The reporter observed no
corruption so far with terminal GPU acceleration disabled; do not label its root
cause fixed by engine changes. Compare long runs with acceleration on/off where
possible and keep terminal-state/output failures distinct from rasterizer errors.

Measure representative games/scenes at fixed viewport sizes and seeds:

- Physics integration, queries, broad/narrow phase, constraints, and allocations.
- Frame preparation, rasterization, AA, and worker utilization.
- ANSI serialization, output bytes/frame, write/flush time, and input-to-visible
  response latency. Report median and tail timings, dropped ticks, and memory
  growth over long runs; averages alone can hide stalls.

Choose optimizations from profiles. Candidates include dirty-cell/color-run output,
buffer reuse, reduced repeated terminal formatting, and further broad-phase work.
Dirty output must invalidate completely after resize, mode changes, screen ownership
changes, and recovery from errors. It must preserve color/glyph correctness and
must not let physics workers write to the terminal.

Harden session initialization/cleanup and report output failures through the runtime
lifecycle. Establish the terminal modes the app requires and a restoration contract;
avoid resetting unrelated terminal preferences blindly.

Acceptance: publish measured results and a tested compatibility table. Use relative
benchmark comparisons on the same machine; set budgets for a named reference
profile rather than universal FPS/body-count promises. Preserve a simple baseline
output path for diagnosis when optimizing serialization.

## 6. Physics extensions driven by interactive tests

### 6A. Moving targets and box CCD

Extend the supported-pair matrix explicitly: relative-motion sphere/sphere, spheres
against moving kinematic primitives, then translating boxes, then rotating boxes.
Treat each as a separately tested increment. Angular motion needs conservative
swept bounds; endpoint AABB unions are insufficient for rotating shapes.

For convex distance queries, evaluate a bounded GJK distance implementation against
current SAT fixtures before using it for conservative advancement. Existing SAT
manifolds remain the resting-contact baseline. General hull CCD follows reliable
box/convex distance queries; it is not implied by accepting hull assets.

Preserve coherent sub-tick time, once-per-tick forces/hooks, transient events,
sleep/wake behavior, joint constraints, and visible CCD-budget diagnostics. Specify
CCD sensor crossing behavior separately from contact response. Never silently
claim continuous collision for an unsupported pair.

### 6B. Joint robustness and mass ratios

Publish a tested mass/size envelope, then expand it with short/long chains, motors
under load, moving supports, and CCD impacts into constrained bodies. Track anchor
error, limit error, energy, and solver cost. Fix observed failures through stable
constraint ordering, effective mass, warm-start rules, or block solving before
raising all iteration counts. Introduce compliance/springs only with explicit
units, timestep behavior, and validation; breakable joints require defined events.

### 6C. Larger worlds

Profile sparse, clustered, dense, mostly sleeping, and streaming-like workloads.
Compare sweep-and-prune with a dynamic AABB tree or spatial grid only where query
and motion patterns justify it. Retain brute-force correctness oracles and stable
pair ordering. Include ray/overlap queries, moving compounds, invalidation, memory,
and creation/removal churn. Parallel simulation is deferred until ownership and
reproducibility are proven and a measured bottleneck justifies it.

Acceptance: every extension has a reproduction fixture, supported limits, numerical
tolerances, and before/after timings. Demonstration scenes expose new capabilities;
the small default sandbox must not pay a large unconditional cost for them.

## Release checkpoints and validation

- Editor vertical slice: stage 0 plus the minimum of stages 1–2. Create a project,
  add/edit/save an object, Play, Stop, and reopen without losing authored state.
- Creator alpha: complete stages 1–3. Reusable scenes, project-local behavior, and
  standalone export work for a small complete game.
- Validated examples and release readiness: stages 4–5. Templates and documented
  native behavior demonstrate that the creator workflow is repeatable.
- Physics expansion: stage 6, prioritized by limitations exposed by real projects.

At each checkpoint, run focused document/runtime/physics tests, race tests, vet,
Staticcheck, PTY editor workflows, and supported cross-builds. Test transactional
loads, undo/redo, instance identity, failed builds, crashed play sessions, and clean
shutdown. Native terminal checks remain distinct from PTY and compilation tests.
Keep stress tests and long benchmarks separate from fast unit suites.

The next implementation milestone is interactive viewport manipulation: configurable
impulses at the picked point or centre of mass, overlap-checked runtime spawning,
runtime deletion/reset, and live physics inspection. Build on the new actions and
read-only queries; write its detailed plan before coding. Fuller component authoring,
gameplay attachment and export still follow. The creator alpha is not complete.
Keep physics extensions behind demonstrated creator workflow needs.
