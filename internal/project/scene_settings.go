package project

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"slices"
)

// SceneSettings describes global state inherited by each new runtime session.
type SceneSettings struct {
	Objective  *ObjectiveSettings  `yaml:"objective,omitempty"`
	Camera     *CameraSettings     `yaml:"camera,omitempty"`
	Lighting   *LightingSettings   `yaml:"lighting,omitempty"`
	Simulation *SimulationSettings `yaml:"simulation,omitempty"`
}

// ObjectiveSettings challenges a dynamic body to activate a named pressure plate
// by the final tick. Zero TickLimit allows unlimited time.
type ObjectiveSettings struct {
	Text      string `yaml:"text"`
	Signal    string `yaml:"signal"`
	TickLimit uint64 `yaml:"tick_limit,omitempty"`
}

// CameraSettings specifies the initial world-space view and clipping planes.
type CameraSettings struct {
	Position, Target      engine.Point
	FOVX, FOVY, Near, Far float64
}

// LightingSettings replaces the complete legacy light rig when present.
type LightingSettings struct {
	Ambient   engine.Color
	Intensity float64
	Lights    []PointLight
}

// PointLight describes an individually switchable scene light.
type PointLight struct {
	ID, Name  string
	Position  engine.Point
	Color     engine.Color
	Intensity float64
	Enabled   bool
}

// SimulationSettings inherits application values for omitted fields.
type SimulationSettings struct {
	GravityX           *float64 `yaml:"gravityx,omitempty"`
	GravityY           *float64 `yaml:"gravityy,omitempty"`
	GravityZ           *float64 `yaml:"gravityz,omitempty"`
	Sleeping           *bool    `yaml:"sleeping,omitempty"`
	DefaultCCD         *bool    `yaml:"defaultccd,omitempty"`
	VelocityIterations *int     `yaml:"velocityiterations,omitempty"`
	PositionIterations *int     `yaml:"positioniterations,omitempty"`
}

// DefaultCamera returns the legacy project view.
func DefaultCamera() CameraSettings {
	return CameraSettings{engine.Point{X: 14, Y: 11, Z: -20}, engine.Point{Y: 1}, 60, 40, .2, 300}
}

// DefaultLighting returns the legacy project light rig.
func DefaultLighting() LightingSettings {
	return LightingSettings{engine.Color{R: 30, G: 30, B: 40}, .5, []PointLight{{"light1", "Key", engine.Point{X: -5, Y: 12, Z: -8}, engine.ColorWhite, 1, true}}}
}
func copyValue[T any](v *T) *T {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
func (s SceneSettings) clone() SceneSettings {
	s.Objective = copyValue(s.Objective)
	s.Camera = copyValue(s.Camera)
	if s.Lighting != nil {
		s.Lighting = copyValue(s.Lighting)
		s.Lighting.Lights = slices.Clone(s.Lighting.Lights)
	}
	if v := copyValue(s.Simulation); v != nil {
		s.Simulation = v
		v.GravityX = copyValue(v.GravityX)
		v.GravityY = copyValue(v.GravityY)
		v.GravityZ = copyValue(v.GravityZ)
		v.Sleeping = copyValue(v.Sleeping)
		v.DefaultCCD = copyValue(v.DefaultCCD)
		v.VelocityIterations = copyValue(v.VelocityIterations)
		v.PositionIterations = copyValue(v.PositionIterations)
	}
	return s
}
func validPosition(p engine.Point) bool {
	return finiteRange(p.X, -1e9, 1e9) && finiteRange(p.Y, -1e9, 1e9) && finiteRange(p.Z, -1e9, 1e9)
}
func (s SceneSettings) validate() error {
	if o := s.Objective; o != nil && (!ValidText(o.Text, 100) || !ValidText(o.Signal, 80)) {
		return fmt.Errorf("objective requires printable text and a plate signal")
	}
	if c := s.Camera; c != nil && (!validPosition(c.Position) || !validPosition(c.Target) || c.Position == c.Target || !finiteRange(c.FOVX, .01, 179.99) || !finiteRange(c.FOVY, .01, 179.99) || !finiteRange(c.Near, 1e-8, 1e9) || !finiteRange(c.Far, c.Near, 1e9) || c.Far == c.Near) {
		return fmt.Errorf("invalid camera position, target, FOV or clipping planes")
	}
	if l := s.Lighting; l != nil {
		if !finiteRange(l.Intensity, 0, 1e6) || len(l.Lights) > 8 {
			return fmt.Errorf("invalid ambient intensity or more than eight lights")
		}
		seen := map[string]bool{}
		for _, p := range l.Lights {
			if seen[p.ID] || !ValidText(p.ID, 80) || !ValidText(p.Name, 80) || !validPosition(p.Position) || !finiteRange(p.Intensity, 0, 1e6) {
				return fmt.Errorf("invalid or duplicate point light %q", p.ID)
			}
			seen[p.ID] = true
		}
	}
	if v := s.Simulation; v != nil {
		for _, g := range []*float64{v.GravityX, v.GravityY, v.GravityZ} {
			if g != nil && !finiteRange(*g, -1e6, 1e6) {
				return fmt.Errorf("invalid gravity")
			}
		}
		for _, i := range []*int{v.VelocityIterations, v.PositionIterations} {
			if i != nil && (*i < 1 || *i > 256) {
				return fmt.Errorf("solver iterations must be 1..256")
			}
		}
	}
	return nil
}

// SetSceneSettings records a validated change in the editor history.
func (e *Editor) SetSceneSettings(s SceneSettings) error {
	d := e.Document()
	d.SceneSettings = s
	return e.change(d)
}
