// Package engine provides scene geometry, transforms, materials, visibility,
// asset loading, and the contracts implemented by rendering backends.
//
// Coordinates are right-handed with +X right, +Y up, and camera-local +Z
// forward. Camera.FOV.X and Camera.FOV.Y are full angles in degrees.
// Transform rotation methods take radians. Projection returns terminal-cell
// coordinates with Y increasing downward. A zero projected depth indicates an
// invalid projection; negative screen coordinates alone do not.
//
// A SceneNode owns an object and a transform. Mesh.Position is a local offset
// applied before the node's world transform. ComputeNodeBounds bounds the
// object's geometry only; child objects must be tested independently.
//
// Transform caches are lazy. Use the transform setters, or call MarkDirty after
// changing exported position, rotation, or scale fields. Scene mutation and
// cache preparation belong on the update goroutine. Render workers may read
// prepared scene state until the next update.
//
// Renderer is the backend strategy contract. Backends own their drawing and
// input implementations; decorators can add scheduling or post-processing.
// This package has no dependency on terminal control or application menus.
package engine
