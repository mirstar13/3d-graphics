package input

import "math"

// LookSettings converts terminal cell motion to camera angles in radians.
// Zero sensitivity uses the default for that axis.
type LookSettings struct {
	Horizontal float64 `yaml:"horizontal,omitempty"`
	Vertical   float64 `yaml:"vertical,omitempty"`
	DragOnly   bool    `yaml:"drag_only,omitempty"`
	InvertY    bool    `yaml:"invert_y,omitempty"`
}

func (s LookSettings) Valid() bool {
	return s.Horizontal >= 0 && s.Vertical >= 0 && !math.IsInf(s.Horizontal, 0) && !math.IsInf(s.Vertical, 0)
}

func (s LookSettings) Angles(x, y int) (right, up float64) {
	if s.Horizontal == 0 {
		s.Horizontal = .035
	}
	if s.Vertical == 0 {
		s.Vertical = .06
	}
	if !s.InvertY {
		s.Vertical = -s.Vertical
	}
	return float64(x) * s.Horizontal, float64(y) * s.Vertical
}
