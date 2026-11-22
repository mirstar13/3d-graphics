# Editor milestone 2: scene hierarchy authoring

Status: implemented and validated. Follows the first editor vertical slice;
completes the hierarchy prerequisite for reusable scene resources in roadmap stage 2.

## Contract and implementation order

1. Expose Project editor as a selectable launcher row, retaining E and existing
   demo/settings navigation.
2. Extend scene version 1 with optional `parent` persistent IDs. Existing files
   without parent remain roots. Positions/rotations are local to the parent;
   primitive sizes affect geometry only, never descendant scale. Reject missing
   parents, cycles, and physics bodies below roots before changing state.
3. Provide deterministic tree traversal and undoable reparent, subtree duplicate,
   and subtree delete commands. Reparent preserves local transforms (so world pose
   can change); root detaches. Duplicate remaps all subtree IDs/parent links,
   retains the external parent, and offsets only the copied root by +1 local X.
   Failed commands leave document/history untouched; the 128 entity limit applies.
4. Build runtime hierarchies independent of serialization order. Visual children
   follow moving/rotating physics roots. Live positions and highlights use world
   transforms. Runtime subtree removal invalidates every descendant handle.
5. Show an indented hierarchy and keyboard parent picker (H, arrows, Enter,
   Escape), with root and valid parents labeled by name and identity. Preserve
   selection by ID across reparent/duplicate. Show parent and local transform
   semantics; block authoring during Play and consume picker input exclusively.

## Acceptance and validation

Old projects load unchanged; nested projects save/reopen with equivalent content.
Test cycles/missing parents/body restrictions, out-of-order entities, atomic failed
commands, duplicate remapping, subtree delete/undo/redo, live child transforms,
stale descendants after runtime deletion, edit/play isolation, parent picker and
small terminals. Run race tests, vet, Staticcheck, editor/demo PTY flows, and
Windows/macOS cross-builds. Commit and push the tested milestone.

## Validation results

- `go test -race ./...`, `go vet ./...`, and Staticcheck passed.
- The editor PTY flow creates a visual child, selects its parent, saves, and
  reopens in another process; the launcher opens the editor through its menu row.
- Existing demo/settings/physics PTY workflows passed after updating Exit's row.
- Windows amd64 and macOS arm64 cross-builds passed. Native terminal testing on
  those systems remains unverified; these builds establish compilation only.

## Deferred

Reusable linked scenes and overrides are the next milestone after this foundation.
Empty grouping nodes, nested physics bodies, world-pose-preserving reparenting,
transform scale, sibling reordering, components, asset browsing, scripting, and
export are not part of this increment. Do not equate this milestone with completion
of roadmap stages 1–2 or the creator alpha.
