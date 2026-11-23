# Milestone 5: interactive viewport manipulation

Status: implemented and validated. See the [Play tools guide](editor-play-tools.md).

1. Add runtime inspection and explicit spawned-entity ownership. Runtime hierarchy
   and selection include spawned objects; stale selections clear after deletion.
   Keep the authored document and its selection independent of runtime state.
2. Queue configurable world-space impulses at a body's centre or a picked local
   anchor. Capture the anchor at picking time, follow the body's pose at execution,
   and reject stale/wrong-entity/visual-only picks. Queueing does not wake or tick.
3. Preview a deterministic box/sphere at a bounded distance along the reticle ray.
   Check a conservative bounding-sphere overlap against enabled colliders (including
   sensors), entity budget and document bounds before confirmation and again at the
   tick boundary. Competing queued spawns cannot create overlapping bodies.
   Preview is textual position/shape/size/mass; Escape cancels without mutation.
4. Add a Play tools panel (Ctrl+L, pauses Play) for impulse XYZ, centre/hit mode,
   spawn shape, size, mass, distance and total entity budget. Enter edits a field;
   I applies configured impulse, G previews spawn, D deletes selected runtime-spawned
   subtree, R resets, Escape closes. Existing semantic Impulse uses these values.
   Settings are workspace-local, not scene content, and survive Play reset.
5. Reset constructs a fresh session from the authored document, preserves pause
   state, clears pending commands/picks/runtime selection, and invalidates handles.
   Show position/rotation, body type, mass, linear/angular velocity, sleep state and
   runtime ownership in a read-only inspector. Runtime deletion is not authoring undo;
   reset/Stop restore the authored scene.
6. Validate queue ordering, collision rechecks, budgets, cancellation, off-centre
   angular response, stale anchors, selection cleanup, reset, edit/play isolation,
   focus, narrow layouts, and PTY flows. Run race tests, vet, Staticcheck and Windows/
   macOS cross-builds. Commit buildable units and push the tested milestone.

## Bounds and deliberate limits

Impulse components: -1000..1000 N s. Spawn size: .05..10 m (box half extents or
sphere radius); mass .01..10000 kg; distance .5..100 m from near-plane reticle origin;
entity budget 1..128, including authored bodies and visual-only entities. Boxes use
an enclosing sphere for placement, so narrow valid gaps can be rejected. Placement
checks colliders, not visual-only geometry. No background worker writes the scene.
Authored objects are protected by explicit ownership, never name matching.
Opening tools pauses the session; closing does not implicitly resume. Commands apply
on the next tick; Escape/F8 stops Play only after modal dialogs are dismissed.
No runtime change is captured back into the authored scene in this milestone.

## Validation results

- Full `go test -race ./...`, `go vet ./...`, and Staticcheck passed.
- Runtime tests verify picked anchors follow transformed bodies, generate angular
  response, stay pending while paused, and reject stale handles. Competing queued
  spawns recheck overlap and budget; authored deletion is rejected by ownership.
- Model tests cover preview cancellation, spawned-object selection, live inspector,
  deletion cleanup, reset clearing pending state, invalid fields, hit-mode checks,
  text focus, short-terminal tools and authored selection/content isolation.
- Editor PTY tests passed configure/preview/cancel/spawn/impulse/delete/reset and
  saved-file isolation. All previous resource/input/editor and demo/settings/physics
  terminal workflows passed.
- Windows amd64 and macOS arm64 cross-builds passed. Native terminal interaction
  on those platforms remains unverified.

The [remaining-work assessment](creator-alpha-remaining.md) estimates 8–12 further
milestones for a creator alpha; this is a planning range, not a schedule.
