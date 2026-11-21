// Package physics implements fixed-step rigid-body simulation with +Y up.
// Positions use metres, time uses seconds, and angular velocity uses radians
// per second. Dynamic spheres and oriented boxes collide with each other,
// static planes, and immutable static triangle meshes. Kinematic bodies supply
// target poses. Contacts support friction, restitution, sensors, and layer masks.
//
// Worlds and their mutations belong to one simulation goroutine. Generation-
// checked handles identify bodies; snapshots never expose mutable world state.
// BeforeStep prepares forces and targets; AfterStep copies authoritative poses
// to scene nodes. Lifecycle changes inside hooks take effect next tick.
//
// The clock bounds catch-up work and never enlarges its integration step.
// Collision detection is discrete; fast bodies can tunnel. Sleeping, joints,
// continuous collision detection, and dynamic triangle meshes are not supported.
// Diagnostic wall-clock timings do not influence simulation results.
package physics
