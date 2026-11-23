# Remaining work toward a creator alpha

Assessment after the interactive-manipulation milestone. This is a scope estimate,
not a delivery schedule or a measured completion percentage.

## What works now

The renderer and substantial rigid-body physics implementation support a working
editor foundation: create/open projects, primitive hierarchy authoring, undo/redo,
linked subtree resources and overrides, source editing, isolated-state Play/Stop,
configurable press actions, collision/visual picking, and runtime manipulation and
inspection. Five creator-facing milestones have been delivered. Their source and
runtime packages are still internal; the original demo runner also remains.

This is useful for building and testing scenes. It is not yet a complete workflow
for attaching custom gameplay and exporting a standalone terminal game. Those are
major remaining architectural and product tasks, not finishing touches.

## Proposed remaining work packages

| Priority | Work package | Exit criterion |
| --- | --- | --- |
| 1 | Component/document authoring | Add optional camera/light/visual/body components, inspectable metadata, simulation overrides and validated migrations without losing existing projects. |
| 2 | Assets and physics authoring | Local mesh/material resource workflow; authored colliders, materials/filters/CCD, joints and stable references can be edited and reopened. |
| 3 | Gameplay lifecycle and actions/events | Registered behavior descriptions, spawn/fixed-update/removal hooks, collision/trigger events, action access and scene transitions work headlessly and in Play. |
| 4 | Supported public API | A separate Go module can create/load/run a game without importing internal packages or the editor. Demo/runtime ownership gaps are resolved. |
| 5 | Project-local Go build workflow | Creators attach/edit typed project behaviors; rebuild errors include source locations, and failed builds preserve the workspace. |
| 6 | Supervised Play process | Compiled games run in a child process with pause/step/stop/debug communication; a panic cannot take down the editor or lose authored edits. |
| 7 | Game presentation and flow | Focus-aware terminal HUD, gameplay movement/input policy, scoring, win/lose screens and restart compose with world rendering and simulation. |
| 8 | Standalone export | Target profiles package the entry scene, assets and game code into a launchable game that does not require this repository or editor. |
| 9 | Complete editable example/template | A small target game demonstrates scene authoring, attached code, collision scoring, HUD, restart, and export through the same supported workflow. |
| 10 | Alpha validation and documentation | Clean-checkout quickstart, native terminal checks, representative performance measurements, failure/cleanup tests and a documented pre-1.0 compatibility contract. |

These packages are not equal in size. Some can combine; component/resource authoring
and the build/Play/export pipeline may each need multiple increments. A reasonable
planning allowance is **8–12 further milestones after manipulation**, subject to
what the first complete game exposes. A calendar estimate would require agreed
alpha scope, staffing and measured delivery pace. Passing native cross-builds does
not replace native terminal testing.

The smallest useful alpha can remain single-player, keyboard-first, Go-based and
focused on simple games. Reusable resources already work; nested linked scenes,
inherited variants, a full graphical editor, advanced physics expansion, networking,
and custom scripting languages need not block that alpha. General asset management
can start with supported local meshes/materials rather than a broad importer suite.

## Immediate next milestone

Plan component/document authoring first, narrowing it into a reviewable vertical
slice (camera/light components and scene simulation settings are good first clients).
Preserve the existing project and resource migration contracts. Then use actual
sample-game needs to prioritize richer authored physics and gameplay APIs.

The [full roadmap](terminal-game-engine-roadmap.md) remains the longer-term target.
Completion should be judged by the author → attach gameplay → Play → export workflow,
not by counting renderer or physics features.
