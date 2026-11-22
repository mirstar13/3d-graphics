# Editor milestone 3: reusable linked scenes

Status: implemented; validation results below.

## Delivery order

1. Add versioned project-local scene resources with stable resource IDs and one
   rooted subtree. Resources contain ordinary entities, not nested links. Reject
   nested/recursive references explicitly before construction. Resource paths must
   remain inside the project, including missing destinations beneath symlinks.
2. Scene documents store instance references, source-to-instance identity maps and
   explicit property overrides. Expanded entities exist in memory only; saved scenes
   retain ordinary entities plus references and root-order metadata. Old scenes remain valid. Resolve and
   validate the whole scene before replacing live/editor state.
3. Instantiate with fresh identities. Set records per-property overrides; reset
   restores source values. Refresh adopts source changes while preserving overrides
   and existing member IDs. Removed overridden members are conflicts, never silently
   discarded. Duplicate/delete operate on complete instances; structural member
   edits require explicit unpack. Physics roots remain roots.
4. Extract selected ordinary subtrees into new resources without overwriting files.
   Provide a keyboard resource browser, source editing/save, return to entry scene,
   link refresh, override reset, and unpack. Dirty document switches require the
   existing discard guard. Play always uses a resolved copy.
5. Test identity, semantic round trips, independent instances, source propagation,
   overrides/conflicts, failed loads/refreshes, missing resources, path containment,
   history, runtime isolation, focus and terminal workflows. Run race tests, vet,
   Staticcheck, PTY flows, and Windows/macOS cross-builds; commit and push.

## Precise semantics

Resources live directly in `resources/*.yaml`; the filename is a locator and the
resource ID is identity. Renamed/missing files produce a contextual error; automatic
asset relocation is deferred. Instance roots are scene roots. Child instances,
linked resources inside resources, inherited variants, and arbitrary hierarchy
patches are deferred. Each source has one root; extracting a child detaches that
root while retaining local pose. Instances preserve the source root's pose unless
explicitly overridden. Root-only physics validation still applies.

Source edits are disk edits with their own undo history; document undo cannot
undo a resource file write. Refresh is explicit and undoable in an open scene.
Reopening resolves the current source. Saving checks references but does not
silently refresh the open scene; creators refresh explicitly to view new source
content. Deleted overridden source members block refresh/reopen until restored or
the instance is unpacked before saving. Unpack retains current expanded values.

Acceptance: extract a multi-object subtree, instantiate twice, override one copy,
edit and save the source, return/refresh, verify propagation without losing the
other copy's override, save/reopen, Play/Stop, duplicate/delete/undo, then unpack.
The milestone supplies basic reusable scenes, not the complete creator alpha.

## Validation results

- Full Go race suite, vet, and Staticcheck passed. Regression tests cover source
  identities, deep-copy undo history, overrides/conflicts, mixed root ordering,
  add/remove source members, invalid/recursive resources, and symlink containment.
- Headless runtime tests verify instance destruction and restart isolation.
- Editor PTY workflow covers extraction, two instances, source editing, overrides,
  reset, unpack, Play/Stop, and reopen in a separate process. Existing demo/settings
  terminal workflows passed. Focus resets to hierarchy on document switches.
- Windows amd64 and macOS arm64 cross-builds passed; native interaction on those
  platforms remains unverified.

Next: semantic input actions and selection/query tools from roadmap stage 1.
See the [resource workflow guide](editor-scene-resources.md) for controls and limits.
