# 3D graphics

The active development branch focuses on the terminal renderer. Go 1.24.6 or
newer is required. The terminal build uses pure Go and can run with CGO disabled.

Run from the repository directory in an interactive terminal:

```sh
go run .
go run . -settings
go run . -config renderer.yaml -mode tiles -aa fxaa
go test ./...
```

Startup loads `renderer.yaml` and opens the start menu. Defaults apply
to omitted fields. Explicit CLI flags override file values. A missing default
file uses built-in defaults, while a missing explicitly requested file or invalid
YAML reports an error. Unknown keys and invalid values are rejected.

## Start menu

`go run .` opens New scene, Settings, and Exit. Use Up/Down and Enter.
New scene opens seven curated terminal scenes, with descriptions and the configured scene selected.
Enter starts that scene. Escape returns from the picker to the start menu.
Leaving a running scene with X, Escape, or Ctrl+C returns to the start menu.
Choose Exit, or press Escape or Ctrl+C in the start menu, to close the application.

## Simulation controls

| Key | Action while rendering |
| --- | --- |
| Space | Pause scene simulation. |
| Enter | Resume simulation. |
| N | Advance one simulation tick while paused. |
| C | Toggle collider outlines and contact markers. |
| P | Open settings, ending the current scene session. |

Camera controls continue working while paused. Animation, lights, node updates,
and LOD transitions advance at a fixed 60 Hz, independently of rendering FPS.
Pause help remains visible when debug output is disabled. Restarting a scene
resets simulation time; resuming never catches up the time spent paused.

The clock accepts at most 250 ms of elapsed time and executes at most eight
simulation ticks per rendered frame. Excess time is discarded to keep the
application responsive; sustained low frame rates can therefore slow simulation.
Debug status shows discarded seconds and dropped input commands. Each input and
pending-step queue holds at most 64 requests; further requests are ignored and
counted. Holding N may repeat steps according to terminal key-repeat settings.
Resume cancels pending steps. Running physics bodies interpolate their previous and
current poses for presentation, adding up to one tick of visual latency. Paused,
single-stepped, and collider-overlay views show the latest authoritative pose.

`internal/physics` also provides tested semi-implicit Euler integration with
default gravity `(0, -9.81, 0)` in metres/second squared. Gravity is available to
simulation clients; it does not automatically make existing demo objects fall.
The Physics playground opts into rigid-body simulation: dynamic spheres and boxes,
static planes and triangle meshes, and a moving kinematic platform. Contacts use
friction, restitution, and a warm-started sequential impulse solver. C displays
colliders and contact normals over the image; debug status reports body, pair,
contact counts, and physics tick cost. Pause and single-step also freeze and step bodies.

Resting dynamic bodies sleep in connected groups, waking on forces, impacts, and
changes to their support. Debug status includes the sleeping-body count.

Dynamic spheres can opt into continuous collision detection (`BodyDesc.CCD`) against
static planes, boxes, and triangle meshes. Other pairs and sensors remain discrete;
fast bodies can tunnel in those cases. CCD has a bounded impact budget and stops
CCD motion conservatively if exhausted. Distance, ball-and-socket, and hinge joints
support local/world anchors and connected-body collision filtering; hinges also
support limits and torque-limited motors. Dynamic triangle meshes remain future work.
Collision shapes are independent of render meshes, LOD, and anti-aliasing.

## Settings screen

Choose Settings in the start menu, use `-settings` at startup, or press P while rendering. Up/Down selects a field,
Left/Right changes its value, and E edits a numeric value directly.
From the start menu, Enter applies settings and returns to the menu.
From a running scene, Enter applies settings and restarts that scene.
S saves settings to the selected YAML file. Escape returns to the start menu,
discarding edits made since the last save or apply. Restarting a scene resets
its camera and animation state.

The editor exposes the demo, CPU mode, anti-aliasing, worker count, tile size,
FPS, viewport size, color depth, horizontal and vertical FOV, auto orbit,
debug display, profiling, and physics controls. Physics settings expose gravity,
solver iterations, sleeping, sphere CCD, and interpolation. Applying changes
restarts the scene. The nested `physics` YAML group also accepts sleep thresholds,
sleep delay, and `ccd_max_impacts` (distinct from clock catch-up steps). Old YAML
files retain defaults. Fixed simulation stays at 60 Hz. Sphere CCD defaults apply
only to dynamic spheres; `BodyDesc.CCDPolicy` can explicitly enable/disable it per
body. Unsupported pairs remain discrete.

The checked-in `renderer.yaml` supplies the default profile. CPU modes are
`single`, `tiles`, `jobs`, and `batched`. Anti-aliasing options are `none`,
`fxaa`, `msaa2`, `msaa4`, and `ssaa`. MSAA uses 2 or 4 subpixel coverage/depth
samples for filled triangles, with one lighting evaluation per triangle/cell.
SSAA renders at twice each dimension and averages each 2x2 block. Both resolve
color and monochrome glyphs. MSAA retains full-cell points and wireframe lines.
FXAA is a color-edge filter; compare the modes on your scenes before selecting
a default. SSAA has higher shading and memory costs.

Horizontal and vertical FOV are full angles in degrees. Increasing an angle
widens the visible scene. Projection, picking rays, and culling share this
convention. Rotation methods use radians with pitch about X, yaw about Y,
and roll about Z.

CLI overrides include `-demo`, `-mode`, `-aa`, `-workers`, `-tile-size`,
`-fps`, `-width`, `-height`, `-color`, `-fov-x`, `-fov-y`,
`-auto-orbit`, `-debug`, and `-profiling`. Boolean flags accept
`=false`. Overrides remain in memory until S saves them in the editor.

## Terminal compatibility

The viewport fits the terminal size at startup, reserving space for the status
line and avoiding the last column's automatic wrapping. Buffers follow window
resizing between completed frames. When size detection is unavailable, the fallback is 80 by 24.

Color detection uses COLORTERM and TERM. NO_COLOR disables automatic colors.
Override detection with `-color none`, `-color 16`, `-color 256`, or
`-color truecolor`. Monochrome output still uses cursor positioning.
Redirected input/output and TERM=dumb produce a diagnostic because the
interactive renderer requires a cursor-addressable terminal.

The default target is 30 FPS. For a smaller workload, use:

```sh
go run . -width 60 -height 20 -fps 15 -workers 1 -color 16
```

Explicit dimensions are capped to the terminal size. Worker count uses the YAML value or Go's available CPU parallelism when omitted.
Single-threaded rendering remains the default.

Windows consoles require virtual terminal processing, which is enabled and
restored for the rendering session. Linux runtime tests exercise pseudo
terminals with xterm-256color, xterm, and linux settings. Windows amd64 and
macOS arm64 are cross-compiled; interactive behavior on those systems still
needs native validation.

## Architecture

The root entry point calls `internal/app.Run`, preserving `go run .`.

| Package | Responsibility |
| --- | --- |
| `internal/engine` | Geometry, transforms, scene graph, lighting, assets, camera control, and renderer/input contracts |
| `internal/terminal` | Terminal rasterization, keyboard input, console capabilities, anti-aliasing, and CPU rendering wrappers |
| `internal/demo` | Example scenes, lighting presets, animation, and demo camera setup |
| `internal/app` | CLI, settings TUI, backend registration, simulation scheduling, and application lifecycle |
| `internal/physics` | Fixed-step rigid bodies, primitive/static-mesh collisions, and contact solving |
| `internal/config` | YAML loading, defaults, validation, and saving |

The application imports settings, the engine, terminal adapter, and demos. The terminal
package depends on the engine; demos also describe physics bodies. The engine has no terminal, keyboard,
or application dependencies.

Anti-aliasing and CPU rendering wrappers live with the terminal adapter because
their existing implementations use its raster buffers. Tests live beside the
packages they exercise. Run the complete suite with `go test ./...`.

## Renderer adapters

Implement `engine.Renderer` and `engine.InputManager` in a backend package.
Wire its factory into the application using `app.RegisterBackend` during
application setup. The factory returns an `app.BackendAdapter` containing
the renderer, input factory, camera orientation, and terminal-output flag.
Backend packages depend on engine contracts. Registration belongs to application
composition so backend packages do not need to import the application.

Input is constructed after renderer initialization. The application owns
initialization and shutdown. An adapter named example can be selected with
`-renderer example`. Adapters are compiled into the executable.
A renderer that needs drawing calls on the calling thread implements
`engine.MainThreadRenderer`.

Local branches `renderers/opengl` and `renderers/vulkan` preserve the original
combined implementation at the pre-refactor commit. They are starting points
for extracting those adapters onto this interface. Their historical code still
contains both graphics backends and the original native dependencies.

## Development and API documentation

Package and API comments are available through Go's documentation tool:

    go doc ./internal/engine
    go doc ./internal/engine.Camera
    go doc ./internal/terminal
    go doc ./internal/config
    go test -race ./...
    go vet ./...

See [the code audit](docs/audit.md) for corrected bugs, regression coverage,
removed code, and remaining improvement areas.

OBJ imports preserve `usemtl` assignments per triangulated face, including faces
before the first material directive. Missing libraries or material names are
reported as errors. MTL diffuse/specular properties are supported; texture-map
loading and material-preserving OBJ export are not yet implemented.

Detail runway uses timed, dithered cross-fades. Morph mode interpolates meshes
with matching vertex/index topology and falls back to a cross-fade otherwise.
Shadow maps select one of the transitioning meshes at the midpoint.

See [terminal compatibility checks](docs/terminal-compatibility.md) for the
repeatable PTY test and native Windows/macOS test procedure.

## Scene catalog

| Number | Scene | What to look for |
| --- | --- | --- |
| 1 | Kinetic gallery | A colored prism, floating sphere, and tumbling ring on plinths. |
| 2 | Orbital clock | Three nested orbits and a moon; a transform animation, not a gravity simulation. |
| 3 | Texture workshop | Broad checker patterns, UV seams, and a gradient ring. |
| 4 | Lantern courtyard | A moving overhead light and visible cast shadows. |
| 5 | Detail runway | Spheres approach and recede through three complete meshes with timed LOD fades. |
| 6 | Wave field | 81 instanced columns animate as a wave; useful for comparing CPU modes and AA. |
| 7 | Physics playground | Falling spheres, stacked crates, a moving platform, a triangle-mesh ramp, and a joint-driven pendulum. |

These replace the former 13 technical examples. Geometry/material examples were
combined into the gallery; hierarchy, textures, shadows, LOD, and performance
examples were rebuilt. Spatial partitioning, collision/physics, and the broad
advanced-feature demos were removed because they did not clearly demonstrate
their advertised behavior in the terminal.

Scene numbers now refer to this catalog. Existing `demo: 1` remains the default;
values 2–7 select the new scenes, and 8–13 must be changed to a valid number before
startup. Use New scene to choose again. All scenes work without auto orbit and
in monochrome. Only Lantern courtyard generates shadow maps. Builders use no
external assets or randomness, and each preset stays below 5,000 triangles,
counting all LOD meshes and rendered instances.

To export a framebuffer contact sheet for review:

    DEMO_PREVIEW=/tmp/demo-preview.png go test ./internal/demo -run TestScenePreview -count=1

Panels follow the picker order from left to right, top to bottom.
