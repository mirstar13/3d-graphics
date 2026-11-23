# Editor milestone 4: input actions and selection queries

Status: implemented and validated. See the [usage guide](editor-input-selection.md).

1. Add read-only closest-ray and sphere-overlap world queries for spheres, oriented
   boxes, planes, static meshes, compounds and convex hulls. Return body identity,
   compound child index (-1 for ordinary shapes), point, normal and ray distance.
   Layer mask and sensor inclusion are explicit; disabled/reserved bodies are skipped.
   Inside solid rays hit at zero with normal opposite direction; planes represent
   their solid negative half-space; triangle meshes remain two-sided surfaces.
   Equal ray distances prefer stable body slot, child index, triangle index. Sphere
   overlap includes touching, one result per child, and reuses narrow-phase/BVH
   geometry without changing contacts, sleeping state, caches or simulation time.
2. Add camera terminal-cell rays matching perspective projection. Runtime picking
   maps body hits to generation-checked entities and tests visual-only primitives
   in their world pose. Physics owns authoritative positions; visual proxies are
   explicitly reported. Keep queries on the update goroutine.
3. Add semantic press actions for Play/Pause/Step/Impulse, pick/clear/cycle selection,
   and reticle movement. Persist project bindings in the manifest; defaults keep
   existing projects working. Validate conflicts, unsupported keys and collisions
   with fixed editor controls. Fixed menus and text fields consume input first.
4. Add a keyboard bindings screen and a viewport focus context (Tab cycles hierarchy,
   inspector, viewport). Viewport arrows move the reticle; pick uses the displayed
   cell, highlights the chosen object, and reports hit data. Selection misses clear
   the selection. Picking never queues simulation changes. Context help derives
   action keys from the binding table. Narrow layouts retain usable focus/help.
5. Test geometry, mesh BVH versus brute-force, filters/ties/inside/invalid rays,
   stale identities and sleeping purity; test binding persistence/conflicts/focus,
   action remapping, selected-cell ray projection, editor PTY and existing demos.
   Run race tests, vet, Staticcheck, Windows/macOS cross-builds, then commit/push.

## Input and scope contract

Existing fixed controls: Up/Down hierarchy/inspector, Tab focus, E/Enter edit,
B/G primitives, H parenting, A/R resources, Ctrl+S/Z/Y/N/O/D/R/U/T, Delete,
1/2/3 cameras, +/- zoom, K bindings, Escape/Q/Ctrl+C exit/stop, F8 stop.
Modal dialogs own all keys until dismissed. Defaults P/F5 Play, Space/F6 pause,
N/F7 step, I impulse remain; F picks, 0 clears, [/] cycle. Reticle arrows apply
only with viewport focus. Bindings edit through K, persist independently of scene
undo, and affect both entry/source editing in the same project.

Terminal input is press/repeat based: no inferred held-key or release state.
Each key in an input batch resolves once; bracketed paste is not dispatched as commands; queued impulses apply once at the next tick,
including pause/single-step. Picking, cycling and reticle motion are presentation
operations and do not tick/wake physics. Mouse, movement axes, arbitrary scene
manipulation, off-centre impulse UI, arbitrary overlap shapes, public SDK and
configurable launcher shortcuts are deferred. Query implementations favor a stable
linear body scan at the existing 128-entity editor budget; mesh geometry uses BVHs.

## Validation results

- `go test -race ./...`, `go vet ./...`, and Staticcheck passed.
- Shape/query tests cover transformed primitives, compounds, hulls, meshes, rays
  starting inside solids, filters, exact-distance ties, reserved/disabled bodies,
  sleeping-state purity, and the triangle BVH against a brute-force scan.
- Endpoint/touching comparisons use a documented 1e-9 metre rounding tolerance.
- Camera tests project ray points back into their selected cell for several sizes.
- Model tests cover focus/modal isolation, manifest persistence and failed saves,
  batched printable keys, paste exclusion, clearing/cycling, and one-shot impulses.
- The editor PTY workflow passed binding edit/persist/reset, picking/clear/cycle,
  paused actions, resizing, and separate-process reopening. Existing resource and
  demo/settings/physics terminal workflows passed.
- Windows amd64 and macOS arm64 builds passed; native terminal interaction there
  remains unverified.

Next: interactive viewport manipulation (configurable impulses, placement queries,
runtime spawn/delete/reset and live physics inspection), planned before coding.
