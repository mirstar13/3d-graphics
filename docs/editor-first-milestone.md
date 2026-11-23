# Project editor: authoring guide

The editor implements **create/open → add/edit → save → Play → Stop → reopen**.
It is an early authoring tool, not the complete editor described in the
[engine roadmap](terminal-game-engine-roadmap.md).

## Launch and first project

From the repository:

```sh
go run . -editor
go run . -project /path/to/MyGame
```

The regular `go run .` launcher offers a selectable **Project editor** row
and the **E** shortcut. Existing demo
and renderer-settings workflows remain available.

1. Press N in the project manager, enter a new directory path, and press Enter.
   The parent directory must exist. Creation refuses to overwrite an existing
   directory. Press O instead to open an existing project directory.
2. The starter scene contains a static floor and a dynamic box. Use Up/Down to
   select an object in the hierarchy. Its oriented bounds appear in the viewport.
3. Press B to add a box or G to add a sphere. Tab cycles hierarchy, inspector, and viewport focus. Up/Down chooses an inspector property; E or Enter edits it.
   Backspace changes the value; Enter validates/applies; Escape cancels.
4. Press Ctrl+S to save. An asterisk marks unsaved authored changes.
5. Press P or F5 to play a separate copy. Space pauses/resumes, N or F7 steps
   while paused, and I queues a 5-unit upward impulse on the selected dynamic body.
6. Press F8 or Escape to Stop. The original authored positions/properties return.
7. Quit with Q or Escape, then use `-project` or O to reopen the saved project.

Project changes and exit prompt before discarding unsaved edits: D discards;
Escape cancels. A failed open keeps the previous project and edits intact.
Save during Play saves the authored document, never the simulated body positions.
See [Play tools](editor-play-tools.md) for checked runtime spawning, picked-point
impulses, deletion/reset and the read-only live inspector.

## Controls

| Context | Key | Action |
| --- | --- | --- |
| Project manager | N / O | New / open project directory. |
| Workspace | Ctrl+N / Ctrl+O | New / open another project. |
| Workspace | Tab, Up/Down | Cycle hierarchy/inspector/viewport; navigate focused panel. |
| Workspace | F / 0 / [ / ] | Pick reticle / clear / previous / next selection. |
| Viewport | Arrow keys | Move reticle one cell per press. |
| Edit | K | Configure project action bindings. |
| Edit | E / Enter | Edit selected property. |
| Edit | B / G | Add box / sphere. |
| Edit | A | Browse resources; Enter instantiate/open, E edit source. |
| Edit | R | Extract selection as a new resource. |
| Edit | Ctrl+R | Refresh links from disk, preserving overrides. |
| Edit | Ctrl+T | Reset selected property's override from source. |
| Edit | Ctrl+U | Unpack selected member's entire instance. |
| Edit | H | Choose a parent (Up/Down, Enter apply, Escape cancel). |
| Edit | Delete / Ctrl+D | Delete / duplicate the selected subtree. |
| Edit | Ctrl+Z / Ctrl+Y | Undo / redo (up to 64 authored changes). |
| Workspace | Ctrl+S | Save authored scene. |
| Workspace | 1 / 2 / 3, + / - | Front / side / perspective view, zoom. |
| Edit | P / F5 | Start Play from current authored document, including unsaved edits. |
| Play | Space / F6 | Pause / resume. |
| Paused Play | N / F7 | Apply pending runtime commands and advance one tick. |
| Play | I | Queue the configured impulse (default: upward, at centre). |
| Play | Ctrl+L | Pause and open manipulation tools: configure, spawn, delete, reset. |
| Play | F8 / Escape / Ctrl+C | Stop; preserve authored state. |
| Edit | Q / Escape / Ctrl+C | Exit editor, guarding unsaved edits. |

Play/selection/reticle action keys in this table are defaults; **K** changes them.
See the [input and picking guide](editor-input-selection.md) for binding and query
semantics. Text entry consumes keys until applied/cancelled. Gameplay commands cannot fire
through a property/path dialog. In narrow terminals, the focused hierarchy or
inspector replaces the viewport to keep selected properties accessible.

## Hierarchy authoring

Set an object's **Body** to **none**, then press **H** to choose a parent by name
and persistent ID. Choose **Scene root** to detach. Physics bodies must remain
roots; a dynamic body may have visual children. The picker excludes the selected
object and its descendants, preventing cycles. Names may be duplicated.

The hierarchy displays children indented beneath their parent. Position and
rotation in the inspector are **local** to the parent. Reparenting keeps those
local values, so the object's world pose may change. Children inherit translation
and rotation during editing and Play. Primitive size changes geometry only;
children do not inherit size as scale. Selection bounds follow world poses.

Delete removes the whole selected subtree. Ctrl+D duplicates the subtree with
fresh IDs and remapped internal parent links, keeping any external parent and
moving the copied root +1 along local X. Each operation, including reparenting,
is one undoable edit. Undo/redo preserves selected identity when it still exists.
Saving/reopening preserves the tree even if children precede parents in YAML.
Older root-only files load without migration; `parent` is optional in version 1.
Older engine binaries reject files containing the new field instead of discarding it.

See the [hierarchy implementation plan](editor-hierarchy-plan.md) for acceptance
criteria and the boundaries of that milestone. Linked reusable scenes are described
in the [resource guide](editor-scene-resources.md).

## Files and schema

A new project contains:

```text
MyGame/
  project.yaml
  scenes/
    main.yaml
```

`project.yaml` contains `version: 1`, project `name`, and a project-relative
`entry_scene`. The scene contains `version: 1`, scene `name`, and ordinary `entities`.
Scenes with reusable content also store `instances` and `root_order`; expanded
instance members are reconstructed from project-local resources, not duplicated
in the saved scene. See the [resource schema](editor-scene-resources.md).
Each entity has a persistent `id`, optional `parent` ID, display `name`, `shape`, `body`, `position`,
Euler-degree `rotation`, `size`, `mass`, and RGB `color`.

- Shapes: `box` and `sphere`.
- Bodies: `none` (visual only), `static`, and `dynamic`.
- Box size is half extents; sphere radius is `size.x`. Y/Z are retained for
  switching shapes and must remain valid positive dimensions.
- Position/size use metres, mass uses kilograms, and rotations use degrees.
- Names need not be unique; IDs must be. Duplicate creates a new persistent ID.
- Version 1 supports nested visual objects, root-level physics bodies, at most
  128 entities, and files up to
  1 MiB. Unknown fields/versions, invalid values, and duplicate IDs are rejected.
- Scene paths must remain inside the project, including existing symlink targets.

Saving validates the complete scene before writing a temporary sibling file and
renaming it into place. Loading constructs and validates a replacement before the
editor switches projects. YAML ordering/comments are not a formatting contract.
The editor never serializes runtime handles, velocities, contact caches, or sleep
state. The current application physics settings supply each new Play world;
project-specific simulation overrides are not in this first schema.

## Architecture

`internal/project` owns schema validation, project storage, inspector metadata,
and undoable document edits. The inspector and serialization use the same entity
fields and validation rules; changes never mutate a live World directly.

`internal/game` owns a headless scene/physics session, session-local generation
handles, persistent-ID lookup, and bounded tick-boundary commands. Visual-only
entities do not need physics. Session creation copies authored values. Destroying
an entity removes its entire subtree, bodies (and attached physics joints), and visuals; stale handles
cannot refer to a replacement or another Play session. Commands remain pending
while paused. A single step consumes each queued command once.

Demo body/joint description types now alias runtime-owned descriptions, and the
existing demo bindings no longer depend on the demo package's types. The existing
demo runner is retained; migrating every demo into project documents is later work.
These packages remain internal while the creator API is being developed.

`internal/app/project_editor.go` composes Bubble Tea, document services, a preview
session, and a separate Play session. The initial viewport uses monochrome,
single-threaded rendering at up to 20 UI updates per second with preset cameras.
Physics remains fixed at 60 Hz with bounded catch-up. Small viewport dimensions
are capped to keep editor interaction responsive. This preview profile does not
replace the configurable full-color demo renderer.

## Validation and remaining scope

Tests cover project round trips, invalid-save preservation, existing-directory
protection, identity/history ownership, stale runtime handles, paused commands,
authored/play isolation, input focus, failed-open preservation, Unicode clipping,
and narrow layouts. Run:

```sh
go test -race ./...
go vet ./...
go build -race -o /tmp/graphics .
python3 scripts/check_editor.py /tmp/graphics
python3 scripts/check_terminal.py /tmp/graphics
```

The editor PTY test creates a project, edits/saves an object, runs/pause/steps it,
queues an impulse, stops, discards an unsaved change, and reopens in a separate
process. It also verifies input-mode restoration and narrow-window operation.
Windows/macOS cross-builds check compilation; native editor interaction still
needs verification on those systems.

Not implemented yet: empty grouping nodes, world-pose-preserving reparenting,
arbitrary component attachment, full asset browsing/import, authored joints/compound/hull resources,
nested linked scenes/inherited variants, behavior scripts, a supervised external Play process,
project export, mouse picking/manipulation, and a public SDK. Play currently runs
in-process; there is no process isolation for future user-authored code. The next
step is extending the document/editor model, not claiming engine/editor parity
from this first workflow.
