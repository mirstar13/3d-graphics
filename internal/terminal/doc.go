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
package terminal
