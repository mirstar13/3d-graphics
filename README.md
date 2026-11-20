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
New scene opens the existing scene presets, with the configured demo selected.
Enter starts that scene. Escape returns from the picker to the start menu.
Leaving a running scene with X, Escape, or Ctrl+C returns to the start menu.
Choose Exit, or press Escape or Ctrl+C in the start menu, to close the application.

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
debug display, and profiling.

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
line and avoiding the last column's automatic wrapping. Restart after resizing
the window. When size detection is unavailable, the fallback is 80 by 24.

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
| `internal/app` | CLI, settings TUI, backend registration, and application lifecycle |
| `internal/config` | YAML loading, defaults, validation, and saving |

The application imports settings, the engine, terminal adapter, and demos. The terminal
and demo packages depend on the engine. The engine has no terminal, keyboard,
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

The LOD demo uses timed, dithered cross-fades. Morph mode interpolates meshes
with matching vertex/index topology and falls back to a cross-fade otherwise.
Shadow maps select one of the transitioning meshes at the midpoint.

See [terminal compatibility checks](docs/terminal-compatibility.md) for the
repeatable PTY test and native Windows/macOS test procedure.
