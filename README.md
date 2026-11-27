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

## Example games

Open **Load scene / project** to play these games. Select **Play** with the mouse
or press **Enter**. Win and game-over screens let you replay, return to the game
menu, or quit to the launcher.

| Example | Play |
| --- | --- |
| [Switch Room](examples/switch-room) | Walk through a first-person room, open the gate and reach the exit. |
| [Pulse Dock](examples/pulse-dock) | Use the arrow keys to steer a tug around a pylon and dock before time runs out. |
| [Bank Shot](examples/bank-shot) | Choose a rail with Left or Right, or aim straight with Down. Press Space to launch one puck. |
| [Lift Run](examples/lift-run) | Steer with Left/Right and brake with Space. Use the launch pad to reach the raised landing. |

[Relay Rooms](examples/relay-rooms) and the [Behavior Workshop](examples/behavior-workshop)
also run from their example directories with `go run .`. The workshop requires
a compiled worker. Pulse Dock supports a turn-based cockpit with `go run . -text`.
Relay Rooms supports its line-input demo with the same flag.

Open an example for editing with `go run . -project examples/bank-shot`, replacing
the folder as needed. Use **Actions → Export game** to build a ZIP, then extract
it and run the included launcher. On Linux, use an audio-enabled build to hear
the examples.

## Project audio

In the editor, press **Ctrl+K** to open audio cues. Use **Import** to add a 16-bit
PCM WAV file in mono or stereo at 44.1 or 48 kHz. Give the file a name such as
`plate_down.wav` using letters, numbers, underscores or hyphens. Select the cue
and use **Audition** or **Stop**. To assign a cue, first select an entity with a
**Plate signal**, open the cue browser and press **Enter** on the cue. Save the
scene with **Ctrl+S**. To clear the assignment, empty **Plate sound** in the
inspector.

Use **Ctrl+Y** during Play, at the audio cue browser, or in an interactive game
to toggle mute. A pressure plate plays once when it becomes occupied. Go game
behaviors can call `world.PlaySound("plate_down")` during a tick. Set
`win_sound` and `loss_sound` in `project.yaml` or use **Use on win** and
**Use on loss** in the editor cue browser to select round result sounds. Go behaviors can call `world.FinishRound(true)` to win or
`world.FinishRound(false)` to lose after the current tick succeeds. Export
includes the project's WAV assets.

Windows and macOS builds include the native audio backend. On Linux, install
the ALSA development files and enable the backend with `go run -tags audio .`.
An editor built this way also enables audio in its project builds and exports.
Use `-tags audio_disabled` for a device-free build. Audio comes from the machine
running the game.

## First-person example

Run `go run .`, choose **Load scene / project**, and select **Switch Room**.
Walk with **WASD**. Hold the right mouse button and drag to turn and look up
or down. Press **M** for free look and **Esc** to release it. The arrow keys
also turn the camera. Aim at the crate and press **F** to push or **G** to
grab or release. Approach the green button and press **E** to open the gate.
Reach the gold pad before time runs out.

Run in any terminal with ANSI cursor positioning and interactive input. Font size
belongs to the terminal's preferences: the engine does not require VS Code or
change your font. See [font size and performance](#font-size-and-performance).

## Project editor (prototype)

Run `go run . -editor`, select **New scene / project** (or press **E**) in the start menu, or open an existing project
with `go run . -project /path/to/MyGame`. Create a project, add boxes/spheres, edit
properties, save, and Play/Stop without changing the authored scene. The workspace
has clickable New, Open, Save, Play, Pause, Step, and Stop controls. Use
**Actions** for entity creation, parenting, undo/redo, resources, scene
settings, build and export. Click hierarchy rows and inspector properties,
and scroll these panels with the mouse wheel. **Browse** selects directories
or import files, and **Keys** opens an on-screen keyboard for text fields.
During Play, drag with the right mouse button or use Shift+arrow keys to turn the
camera. Press **Ctrl+G** to give the last built game the terminal, with any
`terminal_args` from its manifest. The editor regains focus when the child exits.
Press **R** to extract a subtree and **A** to browse resources.
Physics bodies remain root-level. Press **K** to edit project key bindings;
**Tab** to focus the viewport, arrows to aim the reticle, and **F** to pick.
Press **Ctrl+E** to author the scene camera, lighting and simulation settings.
During Play, **Ctrl+L** opens tools for configurable impulses, checked spawning,
runtime deletion, reset and physics inspection. Run `go doc ./internal/app` for
the editor controls and `go doc ./internal/project` for scene storage and links.

Press **Ctrl+X** to browse OBJ and GLB meshes under `assets/`. Use **I** in
the browser to copy in a self-contained Blender-exported GLB, then **Enter**
to assign it to the selected entity. Static triangle meshes use the default
glTF scene's transforms, per-face materials and embedded PNG/JPEG base color
textures. The inspector still controls the physics collider. Animated or
skinned models, transparency, external glTF dependencies and native
`.blend` files are outside this importer.

The **Behavior** and **Behavior parameters (JSON)** inspector fields attach a
named behavior to an entity. Set `behavior_worker` in `project.yaml` to a
project executable under `assets/`, then use **Ctrl+W** for Scripted Play.
The [Behavior Workshop](examples/behavior-workshop/README.md) demonstrates
two crates with different settings and includes C# and C++ implementations
of the same worker protocol. In newly created projects, add
`terminal_args: [-view]` to `project.yaml` to open the interactive terminal
view through **Ctrl+G** after building.


## Start menu

`go run .` opens the launcher. Click an item or use Up/Down and Enter.
Load scene / project lists the demo scenes and games. Choose Open project directory to edit an existing project, including the example. New scene / project opens the editor’s project creation form. Click geometry in the viewport to select it and use the mouse wheel to zoom.
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

Compound colliders combine up to 32 locally positioned/rotated spheres, boxes, or convex hulls
with explicit relative mass weights. They use a full inertia tensor and expose
the construction-origin-to-centre-of-mass offset for visual bindings. Compounds
use discrete collision detection. Collision shapes are independent of render
meshes, LOD, and anti-aliasing.

`physics.ConvexHull` validates an authored closed convex triangle surface (4–64
vertices, up to 128 triangles). It computes mass properties, merges coplanar faces,
and generates multi-point SAT contacts. Hulls interact with spheres, boxes, other
hulls, planes, and static triangle meshes. Automatic hull generation, convex
decomposition, hull CCD, and direct dynamic concave meshes remain future work.

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

The viewport fits within the terminal size at startup, reserving space for the status
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

### Font size and performance

There is no portable API for an application to change a terminal's font size.
Use your terminal's font preferences or zoom controls. Font sizes in pixels and
points differ between terminals and displays, so 11 is not a universal target.

| Terminal | Font setting |
| --- | --- |
| Windows Terminal | Settings → profile → Appearance → Font size |
| macOS Terminal | Settings → Profiles → Text → Font → Change |
| iTerm2 | Settings → Profiles → Text → Font |
| GNOME Terminal | Preferences → profile → Text → Custom font |
| Konsole | Settings → Edit Current Profile → Appearance → Font |
| kitty | `font_size` in `kitty.conf` |
| Alacritty | `[font] size` in `alacritty.toml` |
| VS Code | `terminal.integrated.fontSize` in user settings |

Smaller fonts fit more cells into a window. Rendering and output costs grow with
the cell count, while SSAA multiplies the raster workload by four. Automatic
viewports (`width: 0`, `height: 0`) therefore scale the available grid to at most
160 columns and 60 rows, preserving its proportions. This applies to demos and
the standalone project player, including Switch Room. The rendered area can be
smaller than the window and is centred horizontally and vertically. Demos reserve
a row for status; the project player centres its frame below the HUD. Resizing
or zooming recomputes that area and its margins between frames.
An individually automatic axis is capped at 160 columns or 60 rows.

Use Settings or explicit dimensions to choose the workload independently of font
size, including a larger viewport when your machine and terminal can handle it:

```sh
go run . -width 120 -height 40 -fps 30 -aa none
go run . -width 240 -height 80 -mode tiles -workers 4 -aa none
```

These dimensions also apply when launching Switch Room or Pulse Dock from the
launcher. Explicit dimensions still fit within the available terminal grid.
The editor's authoring viewport follows its panel layout. For lag spikes, first
try a smaller viewport with AA disabled; compare `single` and `tiles` on your
scene. A 30 FPS target cannot guarantee 30 FPS on every machine or terminal.
The workspace file preserves your own font preference.

Windows consoles require virtual terminal processing, which is enabled and
restored for the rendering session. Linux runtime tests exercise pseudo
terminals with xterm-256color, xterm, and linux settings. Windows amd64 and
macOS arm64 are cross-compiled; interactive behavior on those systems still
needs native validation.

### Renderer performance measurements

Run the raster benchmark independently of terminal output and simulation:

```sh
go test ./internal/demo -run '^$' -bench BenchmarkSceneRaster -benchmem -count=3
```

On Linux amd64, Go 1.24.6, AMD EPYC 9V45, two 300 ms runs per case measured
the following mean frame costs before and after optimizing cached transform
access and common specular exponents. Both runs used single-threaded rendering,
no AA, and the same initial scene state. These are raster times, not end-to-end
FPS; physics simulation, terminal output and terminal display latency are excluded.

| Scene | Grid | Before | After |
| --- | --- | --- | --- |
| Kinetic gallery | 160×60 | 0.821 ms | 0.484 ms |
| Kinetic gallery | 320×120 | 2.542 ms | 1.427 ms |
| Wave field | 160×60 | 0.680 ms | 0.426 ms |
| Wave field | 320×120 | 1.735 ms | 1.065 ms |
| Physics playground | 160×60 | 1.810 ms | 0.971 ms |
| Physics playground | 320×120 | 2.964 ms | 1.697 ms |

An SSE2 prototype computed pairs of edge weights. Scene-level measurements
overlapped the scalar baseline, so that prototype was removed. The retained
optimizations use portable Go and preserve support for arbitrary shininess
values; powers of two from 2 through 256 use repeated squaring. Floating-point
results can differ by roundoff from `math.Pow`.

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

### Benchmark OpenGL on your machine

On Linux amd64 or arm64 with Python 3, Go 1.24.6 or newer, and surfaceless EGL
with OpenGL 3.3, run the headless comparison with:

```sh
python3 scripts/benchmark_opengl.py
```

The script reports the active EGL renderers and checks CPU/OpenGL image
agreement before timing. It compares scene rasterization with single and
batched CPU modes and compares the CPU and OpenGL FXAA filters separately.
It writes raw logs and a `summary.csv` under the ignored `.cache/` directory.
Use `--cpu` to set the Go CPU budget and batched worker count, `--output` for
a new results directory, or `--help` for timing options. Recognized software
renderers require `--allow-software`. On a multi-GPU Mesa system, you can use
`DRI_PRIME=1` and check the reported renderer to confirm which GPU was used.

### Experimental offscreen triangle rendering

On Linux amd64 or arm64 with surfaceless EGL and OpenGL 3.3, try:

```sh
go run -tags terminalgl . -renderer terminal-gl-raster -demo 1
```

OpenGL determines triangle coverage and visibility. The engine shades the
winning triangles with its existing CPU materials, textures, and lighting,
then presents the same terminal framebuffer. The debug line shows **GL raster**
or **CPU raster**. The selected CPU mode handles frames with unsupported
drivers, MSAA, wireframes, interleaved points or lines, or screen-door LOD
transitions. Trailing points and lines use the CPU depth test.
OpenGL coverage and depth precision may differ at triangle edges and at
nearly coincident surfaces.

### Experimental offscreen OpenGL FXAA

On Linux amd64 or arm64 with a surfaceless EGL driver that supports OpenGL 3.3,
build and run with:

```sh
go run -tags terminalgl . -renderer terminal-gl -aa fxaa
```

The terminal backend still rasterizes geometry on the CPU. OpenGL filters the
color cells, and the terminal presenter writes ANSI output after color readback.
Use any existing CPU render mode and press **T** for timings. The renderer
reports the active OpenGL device. Builds without the tag, unavailable devices,
and filter errors fall back to CPU FXAA. Other AA modes use their normal CPU
paths. The default renderer is `terminal`.


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

OBJ imports preserve `usemtl` assignments per triangulated face, including faces
before the first material directive. Missing libraries or material names are
reported as errors. MTL diffuse/specular properties are supported; texture-map
loading and material-preserving OBJ export are not yet implemented.

Detail runway uses timed, dithered cross-fades. Morph mode interpolates meshes
with matching vertex/index topology and falls back to a cross-fade otherwise.
Shadow maps select one of the transitioning meshes at the midpoint.

Run `go doc ./internal/terminal` for terminal compatibility checks.

## Scene catalog

| Number | Scene | What to look for |
| --- | --- | --- |
| 1 | Kinetic gallery | A colored prism, floating sphere, and tumbling ring on plinths. |
| 2 | Orbital clock | Three nested orbits and a moon; a transform animation, not a gravity simulation. |
| 3 | Texture workshop | Broad checker patterns, UV seams, and a gradient ring. |
| 4 | Lantern courtyard | A moving overhead light and visible cast shadows. |
| 5 | Detail runway | Spheres approach and recede through three complete meshes with timed LOD fades. |
| 6 | Wave field | 81 instanced columns animate as a wave; useful for comparing CPU modes and AA. |
| 7 | Physics playground | Falling spheres, stacked crates, a moving platform, a triangle-mesh ramp, a pendulum, a compound dumbbell, and a convex wedge. |

These replace the former 13 technical examples. Geometry/material examples were
combined into the gallery; hierarchy, textures, shadows, LOD, and performance
examples were rebuilt. Spatial partitioning, collision/physics, and the broad
advanced-feature demos were removed because they did not clearly demonstrate
their advertised behavior in the terminal.

Scene numbers now refer to this catalog. Existing `demo: 1` remains the default;
values 2–7 select the new scenes, and 8–13 must be changed to a valid number before
startup. Use Load scene / project to choose again. All scenes work without auto orbit and
in monochrome. Only Lantern courtyard generates shadow maps. Builders use no
external assets or randomness, and each preset stays below 5,000 triangles,
counting all LOD meshes and rendered instances.

To export a framebuffer contact sheet for review:

    DEMO_PREVIEW=/tmp/demo-preview.png go test ./internal/demo -run TestScenePreview -count=1

Panels follow the picker order from left to right, top to bottom.

## Creator release validation

The creator workflow covers authoring, project-local builds, supervised runs and
standalone export. Run the engine and sample tests separately because Pulse Dock
has its own Go module:

    go test -race ./...
    cd examples/pulse-dock
    go test -race ./...

For the native export check, frame benchmark and terminal procedures, run
`go doc ./internal/terminal`. For the project build and export controls, run
`go doc ./internal/app`. The public gameplay API is documented by
`go doc ./creator`.

### Demo frame timing

Press T during a demo to toggle update, raster and presentation timings for
the previous frame. Start with `-frame-trace /tmp/playground.frame-trace.csv`
to capture up to 600 completed frames from the first scene you exit. The
trace separates framebuffer write time from total presentation time and records
simulation ticks advanced, encoded framebuffer
bytes before status output, and viewport dimensions. Leaving the scene prints
p50, p95 and p99 frame intervals. Choose a new path for each capture.

### Relay Rooms

Run `go run .` in `examples/relay-rooms` for a turn-based example of project
actions and room transitions. Enter U to use the relay, U again to activate
the exit beacon, R to reload the current room, or Q to quit. Change the action
key in `project.yaml`. To export from the engine checkout, open the example
with `go run . -project examples/relay-rooms` and press F9.

Run `go run . -view` in the same directory for the interactive game. Walk with
WASD and look with the arrow keys or press **M** for mouse look. Get close to
each relay or beacon, aim the reticle and press **U**. Press **R** to restart
from the first room or **Q** to quit. **Esc** releases mouse capture. Terminal
mouse motion has cell resolution and stops at the edge of the window. The
exported launcher accepts the same `-view` flag.

In a standalone Go project, `game.RunTerminal(help)` runs a centred monochrome
view and delivers key presses and accumulated mouse motion to fixed ticks.
A behavior can request capture with `world.CaptureMouse(true)`, read
`dx, dy := world.MouseDelta()`, and call `world.TurnCamera` with its own
sensitivity in radians. Use `world.MouseLook()` to read the project's
`mouse_look` settings. Its optional keys are `horizontal`, `vertical`,
`invert_y`, and `drag_only` (hold the right mouse button after capture).
Positive camera angles turn right and up. Mouse Y
increases downward. A custom host can use `game.MouseCaptured()` and
`game.StepWithInput` to supply the same input.

The sample builds against the engine checkout via its relative Go module
replacement. If you copy the project elsewhere for development, point that
replacement at your engine checkout. The extracted export runs independently
of both source directories.
