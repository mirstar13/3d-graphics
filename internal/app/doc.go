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
// # Building gameplay code
//
// New projects include go.mod and a root main.go with a creator.Behavior.
// Supply those files manually for older scene-only projects. Install Go 1.24.6
// or later and press Ctrl+B in scene Edit mode to compile for the host platform.
// The first build downloads the pinned engine module and dependencies. To build
// against a local engine checkout, replace the module in the project directory:
//
//	go mod edit -replace=github.com/mirstar13/go-3d-graphics=/absolute/path/to/engine
//
// Scroll compiler diagnostics with Up/Down, fix the source, then press Ctrl+B
// to rebuild. Escape cancels a running build or closes its results. Successful
// builds produce build/game (build/game.exe on Windows). Launch from the project
// directory so the starter finds project.yaml. Its main function runs one tick.
//
// F6 runs the last successful build as a child process from the project
// directory. The game reads saved files. Editor edits stay in memory. Escape
// stops the child, F6 restarts it, and Ctrl+B rebuilds after exit. The results
// panel shows the exit status and captured output. The editor owns keyboard
// input throughout. P still opens the interactive scene simulation.
//
// # Objectives and game flow
//
// Add a static sensor with a Plate signal. In Ctrl+E, navigate past the lights
// to Objective, press Enter, and set Text, Signal and TickLimit. A dynamic
// body reaching a plate with the matching signal wins. The deadline counts
// fixed simulation ticks. Contact on the last tick wins, and zero disables the
// deadline. O in scene settings clears the objective.
//
// Play displays the objective and remaining ticks. R restarts a finished round.
// To play the saved scene directly, run:
//
//	go run . -play-project DIRECTORY -player ENTITY_ID
//
// Arrows apply horizontal impulses and Space jumps. Omit -player to watch the
// simulation. Q exits. A restart restores the saved positions and timer.
//
// # Standalone export
//
// Save scene edits and press F9 in Edit mode. Export compiles the root Go
// package and packages its executable, project manifest, entry scene and the
// scenes, resources and assets directories into build/game-OS-ARCH.zip.
// A project-root renderer.yaml is included when present.
// Store additional runtime data under assets. Use ordinary files in these
// directories. An OBJ's material libraries must also be included in the export.
// On a matching platform, extract the complete ZIP and run play.sh (Unix)
// or play.cmd (Windows). The launcher sets the working directory for the game.
// Players need the runtime libraries required by their executable. Building
// requires Go; launching the package uses the compiled program.
//
// # Visual assets
//
// Put Wavefront files in the project assets directory (create it in existing
// projects). In the entity inspector set Mesh OBJ to assets/model.obj. Set
// Material MTL#name to assets/palette.mtl#Copper to override every face or
// leave it empty to use the OBJ face assignments and entity color fallback.
// Clearing Mesh OBJ restores the primitive. The size axes scale OBJ vertices
// about the model origin. Visual-only picking uses the authored primitive proxy.
//
// Collider shape accepts box or sphere. An empty value follows the primitive
// shape. Collider dimensions are box half extents or sphere radius in X. A
// zero component follows the corresponding visual size. Select Body none for
// a visual-only mesh. Asset references also work inside linked resources.
//
// The entity inspector accepts initial linear velocity in m/s and angular
// velocity in rad/s on each world axis. Play applies these to dynamic bodies.
// Enable Sensor on a collider to detect contact without a collision response.
// Friction controls sliding resistance. Restitution ranges from 0 to 1.
// Enter default to inherit the physics material (0.6 friction, 0.1 restitution).
// Collision layer and mask accept 32-bit decimal or 0x-prefixed hexadecimal
// values. Each body's mask must include at least one layer of the other body
// for contact. A zero mask suppresses all contacts. Layer zero also excludes
// the collider from layer-based spatial queries. Enter default for layer 1 or
// mask 0xFFFFFFFF. Sensor contacts follow the same filtering rules.
// Linear and angular damping control how quickly motion and spin decay.
// Zero disables that damping term. Enter default to restore the physics value.
// Set Sphere CCD to on for a fast dynamic sphere to sweep against supported
// static colliders, off to disable it, or default to inherit the scene policy.
// To make a pressure plate, enable Sensor on a static box and set Plate signal
// to a name such as entrance. On a separate solid static body, set Door signal
// to entrance. Matching is case-sensitive and scene-wide, including instances.
// Any dynamic body holding any matching plate keeps all matching doors open.
// A door lifts along world Y by Door lift metres (zero uses its collider height).
// Contact changes take effect at the next fixed tick. After the last occupant
// leaves, the door returns to its authored position. Reset starts it closed.
//
// To build a launch pad, enable Sensor on a static box and set Sensor enter
// impulse Y to 5 N·s. When a dynamic body enters, the pad applies that impulse
// on the next fixed tick. Remaining inside does not retrigger it. Leaving and
// entering again activates it again. The vector uses world axes and a larger
// mass gains less speed. A zero vector disables the action.
// Enable One-shot sensor impulse to launch only the first dynamic entrant.
// With simultaneous entries, physics contact order chooses the recipient.
// Select a sensor in the Play hierarchy to inspect its impulse status. A spent
// one-shot pad has already submitted its impulse. Disabled means its vector is zero.
// Ctrl+L shows the last three sensor impulses, with the contact tick, source
// and recipient IDs, and world-space impulse. Resetting Play clears this history.
// Resetting Play rearms each pad. Existing pads repeat by default.
// These properties support linked-instance overrides and undo like mass.
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
