// Package project owns versioned authored documents independently of runtime state.
package project

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

const Version = 1
const MaxEntities = 128

// Manifest identifies a project and its initial scene. Version is the file schema.
type Manifest struct {
	Version    int    `yaml:"version"`
	Name       string `yaml:"name"`
	EntryScene string `yaml:"entry_scene"`
}

// Entity is an authored primitive with an optional persistent parent ID.
// Position and Rotation are local to the parent. Size contains
// box half extents or the sphere radius in X; Rotation is Euler degrees.
type Entity struct {
	ID       string       `yaml:"id"`
	Parent   string       `yaml:"parent,omitempty"`
	Name     string       `yaml:"name"`
	Shape    string       `yaml:"shape"`
	Body     string       `yaml:"body"`
	Position engine.Point `yaml:"position"`
	Rotation engine.Point `yaml:"rotation"`
	Size     engine.Point `yaml:"size"`
	Mass     float64      `yaml:"mass"`
	Color    engine.Color `yaml:"color"`
}

// Document contains authored state only, never live handles, velocities or caches.
type Document struct {
	Version   int        `yaml:"version"`
	Name      string     `yaml:"name"`
	Entities  []Entity   `yaml:"entities"`
	Instances []Instance `yaml:"instances,omitempty"`
	// RootOrder is serialization metadata consumed by Project.Resolve.
	RootOrder []string `yaml:"root_order,omitempty"`
}

func (d Document) Clone() Document {
	d.Entities = append([]Entity(nil), d.Entities...)
	d.RootOrder = append([]string(nil), d.RootOrder...)
	if d.Instances != nil {
		instances := make([]Instance, len(d.Instances))
		for i, in := range d.Instances {
			instances[i] = cloneInstance(in)
		}
		d.Instances = instances
	}
	return d
}

// NewDocument supplies a static floor and a falling box for a complete play loop.
func NewDocument() Document {
	return Document{Version: Version, Name: "Main", Entities: []Entity{
		{ID: "floor", Name: "Floor", Shape: "box", Body: "static", Position: engine.Point{Y: -.25}, Size: engine.Point{X: 8, Y: .25, Z: 8}, Mass: 1, Color: engine.Color{R: 80, G: 100, B: 120}},
		NewEntity("box", "Box"),
	}}
}
func NewEntity(id, name string) Entity {
	return Entity{ID: id, Name: name, Shape: "box", Body: "dynamic", Position: engine.Point{Y: 3}, Size: engine.Point{X: .5, Y: .5, Z: .5}, Mass: 1, Color: engine.Color{R: 240, G: 160, B: 70}}
}
func ValidText(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len([]rune(s)) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func finiteRange(v, lo, hi float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= lo && v <= hi
}
func (d Document) Validate() error {
	if d.Version != Version {
		return fmt.Errorf("unsupported scene version %d", d.Version)
	}
	if !ValidText(d.Name, 80) {
		return fmt.Errorf("scene name must contain 1..80 printable characters")
	}
	if len(d.Entities) > MaxEntities {
		return fmt.Errorf("scene exceeds %d entities", MaxEntities)
	}
	seen := map[string]bool{}
	for _, e := range d.Entities {
		if !ValidText(e.ID, 80) || seen[e.ID] {
			return fmt.Errorf("duplicate or invalid entity ID %q", e.ID)
		}
		seen[e.ID] = true
		if !ValidText(e.Name, 80) {
			return fmt.Errorf("entity %s: invalid name", e.ID)
		}
		if e.Shape != "box" && e.Shape != "sphere" {
			return fmt.Errorf("entity %s: unsupported shape", e.ID)
		}
		if e.Body != "none" && e.Body != "static" && e.Body != "dynamic" {
			return fmt.Errorf("entity %s: unsupported body type", e.ID)
		}
		for _, v := range []float64{e.Position.X, e.Position.Y, e.Position.Z} {
			if !finiteRange(v, -1000, 1000) {
				return fmt.Errorf("entity %s: position outside -1000..1000 m", e.ID)
			}
		}
		for _, v := range []float64{e.Rotation.X, e.Rotation.Y, e.Rotation.Z} {
			if !finiteRange(v, -360, 360) {
				return fmt.Errorf("entity %s: rotation outside -360..360 degrees", e.ID)
			}
		}
		for _, v := range []float64{e.Size.X, e.Size.Y, e.Size.Z} {
			if !finiteRange(v, .05, 100) {
				return fmt.Errorf("entity %s: size outside 0.05..100 m", e.ID)
			}
		}
		if !finiteRange(e.Mass, .01, 10000) {
			return fmt.Errorf("entity %s: mass outside 0.01..10000 kg", e.ID)
		}
	}
	if err := d.validateHierarchy(); err != nil {
		return err
	}
	return d.validateInstances()
}
