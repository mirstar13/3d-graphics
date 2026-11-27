// Package terminal implements the ANSI terminal renderer and keyboard adapter.
//
// TerminalRenderer owns character, color, and depth buffers. RenderScene
// prepares camera and transform caches, updates lighting and shadows, then
// rasterizes enabled objects. Present writes a completed frame. Initialize
// and Shutdown acquire and restore terminal output state.
//
// ComposeRenderer uses decorators to apply CPU scheduling before the
// anti-aliasing pass. Tile workers share buffers but have disjoint clip
// rectangles and local renderer copies. The job renderer serializes framebuffer
// writes. Unsupported renderer implementations fall back to their own
// RenderScene method instead of sharing mutable clipping state.
//
// Keyboard input has an explicit cancellation channel. Quit and settings
// requests persist across ClearKeys so a frame cannot lose a navigation event.
// Simulation commands use a separate bounded queue, drained atomically without
// ClearKeys erasing events. The application must stop keyboard input before
// handing the console to a TUI. Paused help is visible without debug statistics.
//
// FXAA filters detected color edges. MSAA stores independent subpixel coverage
// and depth, shading once per visible triangle/cell. Points and wireframe lines
// retain full-cell coverage. SSAA shades at twice each dimension and downsamples
// colors, glyphs, and nearest depth. Scratch buffers resize with the viewport.
// LOD fades partition screen cells with complementary ordered dithering.
//
// # Terminal compatibility checks
//
// From the repository root, run the native export check on each target:
//
//	go test -v -count=1 ./internal/project -run '^TestNativePulseDockRelease$'
//
// It exports Pulse Dock, extracts the ZIP into a path containing spaces,
// moves the source project away and runs play.sh or play.cmd from a different
// directory. Scripted turns exercise timeout, restart and docking. The release
// validation workflow runs this test on Linux, Windows and macOS. Linux and
// macOS also exercise interactive editor and renderer flows through PTYs.
//
// For an interactive native check, start the editor with -project
// examples/pulse-dock, enter Play, resize the window, stop, export with F9
// and quit. Launch the extracted game, burn east until timeout, restart, and
// steer north around the pylon to dock. Check cursor visibility, input echo,
// colors and scrolling after returning to the shell. Record OS, architecture,
// terminal application/version, dimensions, build revision and result.
//
// On Windows 10 in VS Code with PowerShell, disabling
// terminal.integrated.gpuAcceleration prevented recurring corruption during
// one follow-up observation. Repeat extended rendering in the same terminal
// session to investigate recurrence. The cause remains unconfirmed.
//
// Font size is controlled by the terminal, not by a portable escape sequence.
// Automatic demo and project-player viewports are bounded at 160x60 cells;
// explicit width/height settings override the automatic limits. Smaller fonts
// should not be used as an implicit rendering-quality setting. See README.md
// for font controls in common terminals. Native console behavior still needs
// testing on each platform; cross-compilation alone does not validate it.
//
// # Pulse Dock frame benchmark
//
//	go test ./internal/app -run '^$' -bench BenchmarkPulseDockView -benchmem -count=3 -cpu=1
//
// This measures a monochrome frame and HUD at two terminal sizes, after loading
// the sample. Simulation, terminal writes and display latency are outside the
// timed loop. On Linux amd64 with Go 1.24.6 and an AMD EPYC 9V74, median
// times across three runs were 0.327 ms at 80x24 and 0.729 ms at 120x40.
// Reusing the framebuffer reduced allocations from about 46 to 16 kB/frame
// and from 110 to 35 kB/frame respectively.
package terminal
