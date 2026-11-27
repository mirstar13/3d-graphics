// Package project owns versioned authored documents independently of runtime state.
//
// # Project files
//
// The manifest selects the initial scene and stores semantic input bindings.
// Scene files store stable entity identities, local transforms, root ordering
// and linked instances. Bodies occupy hierarchy roots. A reusable resource has
// one root and contains ordinary entities. Instance member maps connect each
// source identity to its host identity. Overrides address source IDs and property
// keys. Refresh resolves source changes and reapplies overrides. Unpacking keeps
// the resolved entities and drops their link.
//
// Global camera, lighting and simulation fields belong to the host scene.
// Omitting lighting restores the default rig. Supplying an empty light list
// removes point illumination. Simulation overrides apply independently per field.
package project

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/input"
)

const Version = 1
const MaxEntities = 128

// Manifest identifies a project and its initial scene. Version is the file schema.
type Manifest struct {
	Sounds           map[string]string  `yaml:"sounds,omitempty"`
	BehaviorWorker   string             `yaml:"behavior_worker,omitempty"`
	TerminalArgs     []string           `yaml:"terminal_args,omitempty"`
	MouseLook        input.LookSettings `yaml:"mouse_look,omitempty"`
	GameplayBindings map[string]string  `yaml:"gameplay_bindings,omitempty"`
	Version          int                `yaml:"version"`
	Name             string             `yaml:"name"`
	EntryScene       string             `yaml:"entry_scene"`
	Bindings         input.Bindings     `yaml:"bindings,omitempty"`
}

// Entity is an authored primitive with an optional persistent parent ID.
// Position and Rotation are local to the parent. Size contains
// box half extents or the sphere radius in X; Rotation is Euler degrees.
type Entity struct {
	Behavior     string       `yaml:"behavior,omitempty"`
	BehaviorArgs string       `yaml:"behavior_args,omitempty"`
	ID           string       `yaml:"id"`
	Parent       string       `yaml:"parent,omitempty"`
	Name         string       `yaml:"name"`
	Shape        string       `yaml:"shape"`
	Body         string       `yaml:"body"`
	Position     engine.Point `yaml:"position"`
	Rotation     engine.Point `yaml:"rotation"`
	Size         engine.Point `yaml:"size"`
	Mass         float64      `yaml:"mass"`
	// Velocity and AngularVelocity initialize dynamic bodies in world coordinates
	// (metres per second and radians per second). Other body types retain these
	// authored values for a later switch to dynamic.
	Velocity        engine.Point `yaml:"velocity,omitempty"`
	AngularVelocity engine.Point `yaml:"angular_velocity,omitempty"`
	// MeshAsset selects an OBJ or GLB. MaterialAsset selects a named MTL entry (path#name).
	MeshAsset     string `yaml:"mesh_asset,omitempty"`
	MaterialAsset string `yaml:"material_asset,omitempty"`
	// ColliderShape inherits Shape when empty. Zero collider dimensions inherit Size per axis.
	ColliderShape string       `yaml:"collider_shape,omitempty"`
	ColliderSize  engine.Point `yaml:"collider_size,omitempty"`
	// PlateSignal names the signal held while this sensor overlaps a dynamic body.
	PlateSignal string `yaml:"plate_signal,omitempty"`
	PlateSound  string `yaml:"plate_sound,omitempty"`
	// DoorSignal subscribes a static door to plates with the same project signal.
	DoorSignal string `yaml:"door_signal,omitempty"`
	// DoorLift is the upward opening distance in metres. Zero uses the collider height.
	DoorLift float64 `yaml:"door_lift,omitempty"`
	// EnterOnce consumes the impulse after its first dynamic recipient in a session.
	EnterOnce bool `yaml:"enter_once,omitempty"`
	// EnterImpulse is a world-space impulse delivered to a dynamic body when
	// it enters this sensor. The zero vector disables the action.
	EnterImpulse engine.Point `yaml:"enter_impulse,omitempty"`
	// LinearDamping and AngularDamping control the decay of body velocity.
	LinearDamping  Coefficient `yaml:"linear_damping,omitempty"`
	AngularDamping Coefficient `yaml:"angular_damping,omitempty"`
	// SphereCCD overrides continuous collision detection for a dynamic sphere.
	// The empty policy inherits the scene setting.
	SphereCCD string `yaml:"sphere_ccd,omitempty"`
	// CollisionLayer defines membership bits. CollisionMask selects accepted layers.
	CollisionLayer CollisionBits `yaml:"collision_layer,omitempty"`
	CollisionMask  CollisionBits `yaml:"collision_mask,omitempty"`
	// Friction controls tangential resistance. Restitution sets bounce from zero to one.
	Friction    Coefficient `yaml:"friction,omitempty"`
	Restitution Coefficient `yaml:"restitution,omitempty"`
	// Sensor colliders report overlapping contacts without applying impulses.
	Sensor bool         `yaml:"sensor,omitempty"`
	Color  engine.Color `yaml:"color"`
}

// Document contains authored state only, never live handles, velocities or caches.
type Document struct {
	SceneSettings `yaml:",inline"`
	Version       int        `yaml:"version"`
	Name          string     `yaml:"name"`
	Entities      []Entity   `yaml:"entities"`
	Instances     []Instance `yaml:"instances,omitempty"`
	// RootOrder is serialization metadata consumed by Project.Resolve.
	RootOrder []string `yaml:"root_order,omitempty"`
}

// Clone returns an independent authored snapshot suitable for editing or history.
func (d Document) Clone() Document {
	d.SceneSettings = d.SceneSettings.clone()
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
	if err := d.SceneSettings.validate(); err != nil {
		return err
	}
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
		if err := e.validateBehavior(); err != nil {
			return fmt.Errorf("entity %s: %w", e.ID, err)
		}
		if err := e.validateAssets(); err != nil {
			return fmt.Errorf("entity %s: %w", e.ID, err)
		}
		if e.ColliderShape != "" && e.ColliderShape != "sphere" && e.ColliderShape != "box" {
			return fmt.Errorf("entity %s: unsupported collider shape", e.ID)
		}
		for _, v := range []float64{e.ColliderSize.X, e.ColliderSize.Y, e.ColliderSize.Z} {
			if v != 0 && !finiteRange(v, .05, 100) {
				return fmt.Errorf("entity %s: collider dimensions must inherit (0) or be 0.05..100 m", e.ID)
			}
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
		if e.PlateSignal != "" && (!ValidText(e.PlateSignal, 80) || !e.Sensor || e.Body == "none") {
			return fmt.Errorf("entity %s: plate signal requires a sensor body and a printable name", e.ID)
		}
		if e.PlateSound != "" && (!validCue(e.PlateSound) || e.PlateSignal == "") {
			return fmt.Errorf("entity %s: plate sound requires a plate signal and valid cue ID", e.ID)
		}
		if e.DoorSignal != "" && (!ValidText(e.DoorSignal, 80) || e.Body != "static" || e.Sensor) {
			return fmt.Errorf("entity %s: door signal requires a solid static body and a printable name", e.ID)
		}
		if !finiteRange(e.DoorLift, 0, 1000) {
			return fmt.Errorf("entity %s: door lift must be 0..1000 m", e.ID)
		}
		if !validPosition(e.EnterImpulse) {
			return fmt.Errorf("entity %s: invalid sensor enter impulse", e.ID)
		}
		for _, v := range []engine.Point{e.Velocity, e.AngularVelocity} {
			if !validPosition(v) {
				return fmt.Errorf("entity %s: initial velocity components must be finite and within ±1e9", e.ID)
			}
		}
		if !finiteRange(e.LinearDamping.Resolve(0), 0, math.MaxFloat64) || !finiteRange(e.AngularDamping.Resolve(0), 0, math.MaxFloat64) {
			return fmt.Errorf("entity %s: damping must be finite and nonnegative", e.ID)
		}
		switch e.SphereCCD {
		case "", "off":
		case "on":
			if e.Body != "dynamic" || e.CollisionShape() != "sphere" {
				return fmt.Errorf("entity %s: CCD requires a dynamic sphere", e.ID)
			}
		default:
			return fmt.Errorf("entity %s: unsupported sphere CCD policy %q", e.ID, e.SphereCCD)
		}
		if !finiteRange(e.Friction.Resolve(0), 0, math.Sqrt(math.MaxFloat64)) {
			return fmt.Errorf("entity %s: friction must be nonnegative and small enough to combine with another collider", e.ID)
		}
		if !finiteRange(e.Restitution.Resolve(0), 0, 1) {
			return fmt.Errorf("entity %s: restitution must be within 0..1", e.ID)
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
