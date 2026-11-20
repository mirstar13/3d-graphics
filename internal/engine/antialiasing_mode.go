package engine

// AAMode specifies anti-aliasing mode
type AAMode int

const (
	AANone   AAMode = iota
	AAFXAA          // Fast Approximate Anti-Aliasing
	AAMSAA2x        // 2x Multi-Sample Anti-Aliasing
	AAMSAA4x        // 4x Multi-Sample Anti-Aliasing
	AASSAA          // Super-Sample Anti-Aliasing
)
