# Input actions and keyboard selection

Open a project with `go run . -editor` or the launcher's **Project editor** row.
The editor now has three focus contexts: **hierarchy → inspector → viewport**.
Tab cycles between them. Text fields and dialogs consume input before actions.

## Picking and focus

The viewport draws a `+` reticle. With viewport focus, arrow presses move it by
one terminal cell. Press **F** to select the nearest collider or visual-only
primitive under that cell. A miss clears selection. **0** clears explicitly;
**[** and **]** cycle backward/forward through the authored hierarchy, independent
of focus. The normal hierarchy arrows still work while hierarchy has focus.

The hit message reports persistent identity, distance from the camera's near
plane, and world normal. Bounds highlight the selected object. Visual-only
primitives are labeled **visual proxy**; picking is not a pixel/triangle test of
rendered shading. A primitive visual proxy follows its hierarchy world transform.
Physics picking uses authoritative body poses. Picking and cycling never advance
physics, queue impulses, or wake sleeping bodies. During Play, pause for precise
inspection of a moving object.

Camera presets (1/2/3) and zoom (+/-) remain fixed controls. The ray passes through
the centre of the displayed cell, respecting camera FOV and near/far planes.
Resizing retains relative reticle position and clamps it into the new viewport.
Narrow terminals show the focused panel; viewport focus displays the reticle
instead of an inspector. Extremely short terminals may truncate surrounding help.

## Project key bindings

Press **K** in Edit mode to open **Project key bindings**. Select an action with
Up/Down, press Enter, enter one to three comma-separated keys, and press Enter to
validate and save. Escape cancels/closes. **D** in the list restores all defaults.
Bindings save immediately to `project.yaml`, independently of scene undo/redo.
They apply when editing either the entry scene or a resource in that project.

| Action | Default keys | Context |
| --- | --- | --- |
| Play | P, F5 | Edit |
| Pause/resume | Space, F6 | Play |
| Step | N, F7 | Paused Play |
| Configured impulse (upward by default) | I | Play, selected dynamic body |
| Pick / clear | F / 0 | Workspace |
| Previous / next | [ / ] | Workspace |
| Reticle up/down/left/right | Arrow keys | Viewport focus |

Workspace help displays the current action bindings. Movement help appears with
viewport focus; K always shows all configured actions. A project override replaces
an action's default aliases. To swap keys between two actions, move one to an
unused temporary key first. Conflicts, unsupported keys, and fixed editor keys
are rejected without changing the manifest or active bindings.

Supported binding tokens are unused lowercase letters/digits, `[`/`]`, `space`,
`f5`/`f6`/`f7`, and `f9` through `f12`. Arrow tokens `up`, `down`, `left`, `right`
are permitted only for reticle actions. Fixed controls include Escape/Q/Ctrl+C,
F8, Tab, E/Enter, B/G, H, A/R, K, Delete, Ctrl+L Play tools, camera/zoom keys and the existing Ctrl
commands. Launcher, browser and text-entry navigation remain fixed.

For example, the manifest can contain:

```yaml
bindings:
  action.play: [v, f5]
  action.pick: [t]
  action.aim_up: [w]
  action.aim_down: [s]
```

Old manifests without `bindings` use defaults. Unknown actions or conflicts fail
project loading before the current workspace is replaced.

Terminal input is press-based. Key-repeat messages act as additional presses;
there is no inferred held-key/release state. When several printable keys arrive
in one terminal batch, each is dispatched once in order. Bracketed paste outside
a text field does not execute commands. Impulses remain queued while paused and
are consumed once on the next simulation tick, never replayed across catch-up ticks.

## Read-only query APIs

`internal/physics` exposes `World.Raycast`, `World.OverlapSphere`, and
`RaycastShape` for standalone visual proxies. `internal/game.Session.Pick` maps
physics/visual hits to session-local generation-checked entity identities. These
APIs remain internal until the supported public gameplay API is defined.

- `Ray` contains world origin, nonzero direction (normalized by the query), and a
  finite nonnegative maximum distance in metres. Invalid input returns an error.
- `QueryFilter.LayerMask` intersects collider layers; zero matches nothing.
  `IncludeSensors` opts sensors in. Collision-response masks do not affect queries.
  Disabled bodies and bodies reserved for future activation are always excluded.
- Hits contain body ID, point, normal, distance, and compound child index (`-1`
  for non-compounds). Closest-ray ties use exact distance then stable body slot,
  child order, and triangle order. Session picking prefers physics over visual
  proxies at an exact tie; visual ties use entity slot order.
- Spheres, oriented boxes, planes, static triangle meshes, compounds and authored
  convex hulls are supported. Rays starting inside solids return distance zero
  with a normal opposite the ray. A plane's negative half-space is solid. Meshes
  are two-sided surfaces; parallel/coplanar rays do not intersect them.
- Sphere overlap includes touching within `1e-9` metres of rounding tolerance,
  returning one result per body/compound child, in stable order. Points lie on
  target geometry; normals point outward from it. Overlap distance is zero.
  Ray endpoints use the same rounding tolerance and clamp reported distance to
  the requested maximum. This is not the larger physics contact slop.
- Queries do not mutate bodies, contacts, caches, event streams, or simulation
  timing. They run on the world/session update goroutine, not concurrently with
  simulation. Body scans are linear; static mesh queries use the existing BVH.

Tests cover shape geometry, transformed and inside hits, endpoint boundaries,
mesh BVH versus brute force, filters, sleeping-state purity, deferred activation,
stale handles, projection/cell agreement, remapping and input focus. The editor
PTY test edits/persists/resets a binding, picks/cycles/clears, steps a queued impulse,
resizes, and reopens in another process.

Mouse selection, reliable held-key axes, arbitrary overlap shapes, configurable
launcher keys, arbitrary runtime property editing are outside this
milestone. Off-centre impulses and runtime spawning are now available through
[Play tools](editor-play-tools.md). See the [implementation plan](editor-input-selection-plan.md).
