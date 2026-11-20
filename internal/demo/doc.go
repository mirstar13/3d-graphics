// Package demo provides the scene presets shown in the application's picker.
//
// Names and the Demo constants use the same ordering. BuildScene populates a
// scene, ConfigureCamera and ConfigureCameraController set its initial view,
// and AnimateSceneDemo advances its example animation. Application FOV and
// orbit settings are applied after these preset defaults.
//
// Demo functions are examples of the engine API and may print a short scene
// description during startup. They do not own terminal input or renderer
// lifecycle.
package demo
