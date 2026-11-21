// Package demo provides a curated catalog of deterministic procedural scenes.
// Presets owns names, descriptions, camera framing, shadow policy, builders, and
// animation. BuildScene initializes time zero; AnimateSceneDemo evaluates pose
// from elapsed simulation seconds. Scene.Update advances LOD transitions. The application
// applies user FOV and orbit settings after preset defaults and owns input and
// renderer lifecycle. Builders do not print into the terminal framebuffer.
package demo
