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
//
// # Project editor
//
// Launch with -editor or open a project using -project DIRECTORY. N creates a
// project in a new directory. O opens an existing one. Tab cycles hierarchy,
// inspector and viewport focus. B and G add primitives. H reparents a subtree.
// Ctrl+D duplicates it. Delete removes it. E edits an inspector property.
// Ctrl+Z and Ctrl+Y navigate history. Ctrl+S saves authored data during Edit or Play.
//
// R extracts the selection as a resource. A browses resources. Enter instantiates
// a resource and E opens its source. Ctrl+R refreshes linked instances. Ctrl+T
// clears a property override and Ctrl+U unpacks an instance. K edits project key
// bindings. In viewport focus use arrows to aim and F to pick.
//
// Ctrl+E opens scene settings. Choose Camera, Ambient, Simulation or a point light
// with the arrows. Enter edits the displayed JSON, Ctrl+U clears the input and
// Escape cancels. Null simulation fields inherit the application setting. L adds a
// point light, D removes the selected light, and C, R and S restore the camera,
// lighting and simulation defaults. Brightness appears as glyph density in the
// monochrome preview. The view presets and zoom are temporary.
//
// P or F5 starts a runtime copy. Space pauses, N steps, and F8 stops it. Ctrl+L
// opens Play tools and pauses time. Configure the world-space impulse and choose
// center or hit. Pick a collider to apply at a body-local anchor. G previews a
// spawn along the reticle ray. Enter queues it for the next tick, which checks
// placement and budget again. D deletes a runtime spawn. R resets the session.
package app
