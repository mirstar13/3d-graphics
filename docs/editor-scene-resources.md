# Reusable linked scenes

A scene resource is a reusable subtree. Instances retain a link to that source,
with their own entity identities and explicit property overrides. The source and
scene are different documents; saving one never silently writes the other.

## Create, reuse, and edit

1. Assemble an ordinary subtree in the entry scene. Select its root and press **R**.
   Enter a resource name. Extraction creates a new file under `resources/` and
   leaves the original objects unchanged. Extracting a child detaches the copied
   root and keeps its local pose. Linked objects must be unpacked before extraction.
2. Press **A** to browse sources. Up/Down selects a resource; **Enter** instantiates
   it at the scene root. Placement initially follows the source. Use the inspector
   to reposition it. Instantiate again for another independent copy.
3. Edit an instance property as usual with **E/Enter**. The inspector marks explicit
   overrides with `*`; hierarchy members show `[linked]`. Even a value equal to the
   current source is an explicit override and remains fixed when the source changes.
4. **Ctrl+S** saves the entry scene. In the browser, select the resource and press
   **E** to edit its source. A `[SOURCE]` label identifies this document. Modify it
   and save with **Ctrl+S**. Press **A**, choose **Entry scene**, and press Enter to
   reopen the saved entry scene with the new source values. Unsaved switches offer
   **D** to discard or **Escape** to cancel; save before switching to retain edits.
5. **Ctrl+R** refreshes all links in an already open entry scene, preserving
   overrides. Refresh is one undoable command. Opening a scene also resolves the
   current source. Source saves are disk operations; scene undo does not undo them.
6. Select an overridden property and press **Ctrl+T** to reset it to its current
   source value. This changes only that property, not other source changes.
7. **Ctrl+U** unpacks the selected member's entire instance into ordinary entities,
   preserving current values and IDs. This removes the source dependency and makes
   structural edits available. Unpack and override reset both support undo/redo.

Delete and Ctrl+D act on an instance **root**, removing/copying the whole instance.
The duplicate keeps its link, remaps every member ID, and records the copied root's
name and +1 local-X placement as overrides. Individual member deletion, member
reparenting, and parenting ordinary objects into an instance require unpacking.

Play receives the expanded document as a separate runtime copy. Impulses and runtime
removal cannot change the source, another instance, or the authored scene.

## Source updates and conflicts

Existing members keep their instance IDs. New source members receive fresh IDs,
which persist when the refreshed scene is saved. Removing a member with no overrides
removes it on refresh. Removing an overridden member is a conflict: refresh/open/save
fails with a contextual error, leaving the open document and existing file intact.
Restore that source member, or unpack the still-open instance before saving.
Changing a source's root identity is also a conflict; create a new resource instead.

Missing files, replaced resource IDs, invalid overrides, unsupported fields/versions,
invalid hierarchy/physics combinations, and budgets fail before replacing the
current workspace. Saving validates links against disk but does not automatically
refresh the displayed scene. Refresh explicitly to preview source changes.

Each source editor has independent undo history, reset when switching documents.
Returning to the entry scene reloads it from disk. There is no unsaved multi-tab
workspace or filesystem watcher in this milestone.

## Schema and storage

`resources/<generated-id>.yaml` contains `version`, stable `id`, and `scene`.
The embedded scene uses the normal primitive hierarchy schema and has exactly one
root. Resource names are labels; names need not be unique. The browser also displays
resource IDs, and invalid files stay visible with a diagnostic.

A saved entry scene stores ordinary `entities`, plus `instances` containing:

- `root`: the instance root's persistent entity ID.
- `resource`: the expected source resource ID.
- `path`: a locator directly inside `resources/`, ending in `.yaml`.
- `members`: source entity ID → instance entity ID mappings.
- `overrides`: source entity ID → property key → string value mappings.

`root_order` preserves hierarchy root ordering when ordinary and linked objects
are mixed. Expanded instance entities are in-memory data, not duplicated payloads
in the scene file. Resource and override maps use stable YAML serialization; exact
formatting and comments are not preserved. Entity IDs, including new mappings,
are independent of names and runtime handles.

Existing version-1 scenes without these optional fields load unchanged. Older
engine versions reject the new fields instead of ignoring linkage. Source and
scene writes use validated temporary files and atomic replacement. Reads/writes
are capped at 1 MiB; expanded scenes at 128 entities; the browser at 256 files.
Paths are project-relative and checked through existing ancestor symlinks, including
new-file destinations. Automatic relocation after resource renaming is not supported.

## Boundaries and validation

Resources cannot contain linked instances: nested links, recursive dependencies,
and inherited variants are rejected. Instance roots are scene roots, so the existing
root-only physics rule remains intact. General asset import, materials as resources,
script components, export, and nested override merging are later work.

The [implementation plan](editor-resources-plan.md) records this milestone's scope.
Tests cover independent copies, source propagation, override conflicts, deep-copy
history, identity remapping, mixed-root ordering, malformed/missing references,
symlink containment, runtime isolation, failed switches, and focused input. The
editor PTY script exercises extraction, two instances, source edits, overrides,
reset, unpack, Play/Stop, and reopening in another process.
