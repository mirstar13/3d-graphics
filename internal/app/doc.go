// Package app composes backends, loads settings, and runs the terminal UI.
//
// Run opens the start menu. New scene selects a demo, Settings edits the
// current configuration, and Exit closes the application. Rendering and
// Bubble Tea never own keyboard input simultaneously. Returning to settings
// shuts down the renderer and its keyboard manager before opening the editor.
//
// Command-line values explicitly supplied by the user override YAML settings.
// Editor changes take effect when applied; saving writes the complete current
// configuration. Re-entering a scene resets its animation and camera state.
//
// Each scene session owns a fixed-step physics world. Scene animation, lighting,
// and node updates run on simulation ticks; camera and rendering use frame time.
// Simulation commands are consumed before updates, independently of movement keys.
//
// BackendAdapter is the application-side factory result that joins a renderer,
// an input factory, and camera conventions. Factories are registered during
// startup; registration must finish before rendering begins.
package app
