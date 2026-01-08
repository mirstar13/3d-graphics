package main

// Rendering constants
const (
	FOV_X = 60.0
	FOV_Y = 30.0

	DEFAULT_CAMERA_Z = -200.0
	DEFAULT_DZ       = 0.0

	FILLED_CHAR           = '\u2588'
	SHADING_RAMP          = ".:-=+*#%@"
	EDGE_DETECT_THRESHOLD = 2.0

	ATTENUATION_CONSTANT  = 1.0
	ATTENUATION_LINEAR    = 0.01
	ATTENUATION_QUADRATIC = 0.001

	AO_MIN = 0.4
	AO_MAX = 1.0

	ASPECT_RATIO = 1
)

// Default charset for ASCII rendering (intensity levels)
var DefaultCharset = [...]rune{' ', '`', '"', '-', '~', 'o', 'x', 'O', 'X'}
