package physics

// These tolerances target metre-scale scenes, not arbitrary unit systems.
const (
	faceAxisBias               = .0001 // Metres: prefer the first nearly equivalent reference face.
	edgeAxisBias               = .001  // Metres: require a meaningful edge-axis improvement.
	contactSlop                = .005  // Metres: narrow-phase contact margin.
	penetrationSlop            = .002  // Metres: allowed resting penetration.
	restitutionSpeedThreshold  = 1.0   // Metres/second: suppress resting bounce.
	maxPositionCorrection      = .2    // Metres per position iteration.
	positionCorrectionFraction = .4
	cacheNormalAlignment       = .95 // Cosine of maximum normal change.
	cachePointDistance         = .1  // Metres: maximum warm-start point displacement.
)
