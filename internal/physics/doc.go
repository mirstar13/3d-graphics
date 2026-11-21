// Package physics provides a fixed-step clock and gravity integration primitives.
// The application owns a world per scene session, supplies elapsed durations,
// and runs updates synchronously before rendering. Overload drops whole elapsed
// ticks after the catch-up budget; physics never increases its integration step.
// No rigid bodies, collision detection, or contact solver are implemented yet.
//
// Gravity is in metres/second squared, with +Y up. Future dynamic-body poses must
// be owned by physics and copied to unparented scene transforms before render
// cache preparation. Demo animation must not also write those dynamic poses.
// Previous/current pose interpolation is deferred until bodies are available.
package physics
