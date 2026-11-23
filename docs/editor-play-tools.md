# Interactive Play tools

Start Play with P (or your configured binding), then press **Ctrl+L**. The tools
panel pauses the simulation. Closing it with Escape keeps Play paused; N steps,
Space resumes, and Escape/F8 outside the panel stops Play.

## Configure and apply impulses

Up/Down chooses a field; Enter edits it. Enter validates/applies the value, while
Escape cancels the field edit. Configure impulse X/Y/Z in world coordinates and
choose `center` or `hit`. The panel shows direction components and magnitude before
application; the workspace footer shows the current vector and mode.

**I** in the panel queues the impulse. The configured workspace Impulse action
(default I) uses the same settings. In `center` mode the impulse acts at the centre
of mass. For `hit`, close the panel, aim the reticle, and pick a dynamic collider
with F. Reopen the panel and apply the impulse. The picked point is stored in the
body's local frame, so it follows translation/rotation until the next tick.
Off-centre impulses can produce angular velocity.

Changing selection clears the picked anchor. Static/visual-only objects and stale
or mismatched picks produce an error. Queueing an impulse does not advance or wake
the simulation. Close tools and press N to apply pending commands once; Space
resumes ordinary ticking. Default settings retain the original upward impulse.

## Preview and spawn

Configure shape (`box`/`sphere`), size, mass, ray distance, and total entity budget.
Size is a box's half extent on every axis or a sphere's radius. **G** opens a text
preview showing shape, world position, size and mass. Position is the chosen
distance from the camera near plane along the reticle ray. Enter queues the spawn;
Escape cancels without creating an object. This preview is not a drawn ghost mesh.

Placement checks enabled colliders, including sensors, before preview and again at
the execution tick. Boxes use a conservative enclosing sphere, so tight gaps can
be rejected even if a box would fit. Visual-only objects do not block placement.
Commands are processed in submission order: if two pending spawns compete for the
same space or the remaining budget, the first can succeed and the second reports
an error. The command batch is not an all-or-nothing transaction.

Once activated, a spawn appears in the live hierarchy and is selected. It has its
own runtime identity and an explicit runtime-ownership flag. Selection, picking,
impulses and inspection work on spawned objects exactly as on authored ones.

## Delete, reset, inspect

**D** in tools queues deletion of the selected runtime-spawned subtree. Authored
objects are protected by ownership, regardless of their names. At execution,
visuals, bodies and attached physics joints are removed, and stale selection/pick
handles are cleared. Runtime deletion is not recorded in authoring undo history.

**R** in tools resets Play from the unchanged authored document. It clears pending
commands, spawned objects, runtime selection and picked anchors, restarts tick zero,
and replaces session identities. Pause state and tool settings are retained.

The Play inspector is read-only and shows name, body type, runtime ownership,
world position/rotation, mass, linear/angular velocity and sleeping state. Tab
focuses it; Up/Down scrolls fields in short terminals. Stop returns to authored
selection and content. Ctrl+S during Play still saves only the authored document.

## Limits and persistence

| Setting | Range |
| --- | --- |
| Impulse component | -1000..1000 N s |
| Size/radius | .05..10 m |
| Mass | .01..10000 kg |
| Ray distance | .5..100 m |
| Entity budget | Integer 1..128, including authored and visual-only entities |

Tool settings are workspace-local: they survive Play reset/Stop but are not written
to the scene or project manifest. The panel has fixed keys; the existing global
Impulse action remains rebindable through K. No simulation command is dispatched
through a text-entry field. Runtime poses and modifications are not captured back
to authored content in this milestone.

See the [implementation plan](editor-manipulation-plan.md) and
[remaining creator-alpha work](creator-alpha-remaining.md).
