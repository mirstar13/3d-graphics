package project

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"strconv"
)

// Property supplies inspector metadata and conversion in one place.
type Property struct {
	Key, Label, Units string
	Choices           []string
	Get               func(Entity) string
	Set               func(*Entity, string) error
}

var Properties = properties()

func properties() []Property {
	text := func(key, label string, get func(*Entity) *string, choices ...string) Property {
		return Property{Key: key, Label: label, Choices: choices, Get: func(e Entity) string { return *get(&e) }, Set: func(e *Entity, v string) error { *get(e) = v; return nil }}
	}
	number := func(key, label, units string, get func(*Entity) *float64) Property {
		return Property{Key: key, Label: label, Units: units, Get: func(e Entity) string { return strconv.FormatFloat(*get(&e), 'f', -1, 64) }, Set: func(e *Entity, v string) error {
			x, err := strconv.ParseFloat(v, 64)
			if err == nil {
				*get(e) = x
			}
			return err
		}}
	}
	result := []Property{
		text("name", "Name", func(e *Entity) *string { return &e.Name }),
		text("shape", "Shape", func(e *Entity) *string { return &e.Shape }, "box", "sphere"),
		text("body", "Body", func(e *Entity) *string { return &e.Body }, "none", "static", "dynamic"),
		number("position.x", "Position X", "m", func(e *Entity) *float64 { return &e.Position.X }),
		number("position.y", "Position Y", "m", func(e *Entity) *float64 { return &e.Position.Y }),
		number("position.z", "Position Z", "m", func(e *Entity) *float64 { return &e.Position.Z }),
		number("rotation.x", "Pitch", "deg", func(e *Entity) *float64 { return &e.Rotation.X }),
		number("rotation.y", "Yaw", "deg", func(e *Entity) *float64 { return &e.Rotation.Y }),
		number("rotation.z", "Roll", "deg", func(e *Entity) *float64 { return &e.Rotation.Z }),
		number("size.x", "Half X / radius", "m", func(e *Entity) *float64 { return &e.Size.X }),
		number("size.y", "Half Y (box)", "m", func(e *Entity) *float64 { return &e.Size.Y }),
		number("size.z", "Half Z (box)", "m", func(e *Entity) *float64 { return &e.Size.Z }),
		number("mass", "Mass", "kg", func(e *Entity) *float64 { return &e.Mass }),
	}
	for _, channel := range []string{"red", "green", "blue"} {
		channel := channel
		get := func(e *Entity) *uint8 {
			switch channel {
			case "red":
				return &e.Color.R
			case "green":
				return &e.Color.G
			default:
				return &e.Color.B
			}
		}
		result = append(result, Property{Key: channel, Label: channel, Get: func(e Entity) string { return strconv.Itoa(int(*get(&e))) }, Set: func(e *Entity, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 255 {
				return fmt.Errorf("color must be 0..255")
			}
			*get(e) = uint8(n)
			return nil
		}})
	}
	result = append(result,
		number("velocity.x", "Initial velocity X", "m/s", func(e *Entity) *float64 { return &e.Velocity.X }),
		number("velocity.y", "Initial velocity Y", "m/s", func(e *Entity) *float64 { return &e.Velocity.Y }),
		number("velocity.z", "Initial velocity Z", "m/s", func(e *Entity) *float64 { return &e.Velocity.Z }),
		number("angular_velocity.x", "Initial angular X", "rad/s", func(e *Entity) *float64 { return &e.AngularVelocity.X }),
		number("angular_velocity.y", "Initial angular Y", "rad/s", func(e *Entity) *float64 { return &e.AngularVelocity.Y }),
		number("angular_velocity.z", "Initial angular Z", "rad/s", func(e *Entity) *float64 { return &e.AngularVelocity.Z }),
		Property{Key: "sensor", Label: "Sensor", Choices: []string{"false", "true"}, Get: func(e Entity) string { return strconv.FormatBool(e.Sensor) }, Set: func(e *Entity, v string) error {
			value, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("sensor must be true or false")
			}
			e.Sensor = value
			return nil
		}})
	return append(result,
		coefficientProperty("friction", "Friction", func(e *Entity) *Coefficient { return &e.Friction }),
		coefficientProperty("restitution", "Restitution", func(e *Entity) *Coefficient { return &e.Restitution }),
		bitsProperty("collision_layer", "Collision layer", func(e *Entity) *CollisionBits { return &e.CollisionLayer }),
		bitsProperty("collision_mask", "Collision mask", func(e *Entity) *CollisionBits { return &e.CollisionMask }),
		coefficientProperty("linear_damping", "Linear damping", func(e *Entity) *Coefficient { return &e.LinearDamping }),
		coefficientProperty("angular_damping", "Angular damping", func(e *Entity) *Coefficient { return &e.AngularDamping }),
		Property{Key: "sphere_ccd", Label: "Sphere CCD", Choices: []string{"default", "on", "off"},
			Get: func(e Entity) string {
				if e.SphereCCD == "" {
					return "default"
				}
				return e.SphereCCD
			},
			Set: func(e *Entity, v string) error {
				if v == "default" {
					v = ""
				}
				e.SphereCCD = v
				return nil
			}},
	)
}

// Editor owns authored snapshots and a bounded history. Runtime sessions receive copies.
type Editor struct {
	document, saved Document
	past, future    []Document
}

func NewEditor(d Document) (*Editor, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &Editor{document: d.Clone(), saved: d.Clone()}, nil
}
func (e *Editor) Document() Document { return e.document.Clone() }
func (e *Editor) Dirty() bool        { return !reflect.DeepEqual(e.document, e.saved) }
func (e *Editor) MarkSaved()         { e.saved = e.document.Clone() }
func (e *Editor) change(next Document) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if reflect.DeepEqual(e.document, next) {
		return nil
	}
	e.past = append(e.past, e.document.Clone())
	if len(e.past) > 64 {
		e.past = e.past[1:]
	}
	e.future = nil
	e.document = next.Clone()
	return nil
}
func (e *Editor) Set(id, key, value string) error {
	next := e.Document()
	for i := range next.Entities {
		if next.Entities[i].ID == id {
			for _, p := range Properties {
				if p.Key == key {
					if err := p.Set(&next.Entities[i], value); err != nil {
						return err
					}
					next.recordOverride(id, key, p.Get(next.Entities[i]))
					return e.change(next)
				}
			}
			return fmt.Errorf("unknown property %s", key)
		}
	}
	return fmt.Errorf("entity no longer exists")
}
func newID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("entity-%x", b), nil
}
func (e *Editor) Add(shape string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	next := e.Document()
	entity := NewEntity(id, "New "+shape)
	entity.Shape = shape
	next.Entities = append(next.Entities, entity)
	return id, e.change(next)
}

// Delete removes a complete subtree as one undoable edit.
func (e *Editor) Delete(id string) error {
	next := e.Document()
	if i, _, ok := next.Link(id); ok {
		if next.Instances[i].Root != id {
			return fmt.Errorf("delete the instance root or unpack before deleting a member")
		}
		next.Instances = append(next.Instances[:i], next.Instances[i+1:]...)
		if len(next.Instances) == 0 {
			next.Instances = nil
		}
	}
	removed := map[string]bool{}
	for _, id := range next.SubtreeIDs(id) {
		removed[id] = true
	}
	if len(removed) == 0 {
		return fmt.Errorf("entity no longer exists")
	}
	kept := next.Entities[:0]
	for _, v := range next.Entities {
		if !removed[v.ID] {
			kept = append(kept, v)
		}
	}
	next.Entities = kept
	return e.change(next)
}

// Duplicate copies a subtree, remapping internal links and retaining its external
// parent. Only the new root receives a +1 local-X offset.
func (e *Editor) Duplicate(id string) (string, error) {
	next := e.Document()
	if i, _, ok := next.Link(id); ok && next.Instances[i].Root != id {
		return "", fmt.Errorf("duplicate the instance root or unpack first")
	}
	ids := next.SubtreeIDs(id)
	if len(ids) == 0 {
		return "", fmt.Errorf("entity no longer exists")
	}
	if len(next.Entities)+len(ids) > MaxEntities {
		return "", fmt.Errorf("scene exceeds %d entities", MaxEntities)
	}
	remap := map[string]string{}
	for _, old := range ids {
		fresh, err := newID()
		if err != nil {
			return "", err
		}
		remap[old] = fresh
	}
	for _, v := range e.document.Entities {
		if fresh, ok := remap[v.ID]; ok {
			if v.ID == id {
				name := []rune(v.Name)
				v.Name = string(name[:min(len(name), 75)]) + " copy"
				v.Position.X += 1
			}
			v.ID = fresh
			if parent, ok := remap[v.Parent]; ok {
				v.Parent = parent
			}
			next.Entities = append(next.Entities, v)
		}
	}
	for _, in := range e.document.Instances {
		if fresh, ok := remap[in.Root]; ok {
			copy := cloneInstance(in)
			copy.Root = fresh
			for source, target := range copy.Members {
				copy.Members[source] = remap[target]
			}
			next.Instances = append(next.Instances, copy)
		}
	}
	for _, v := range next.Entities {
		if v.ID == remap[id] {
			next.recordOverride(v.ID, "name", v.Name)
			prop, _ := property("position.x")
			next.recordOverride(v.ID, "position.x", prop.Get(v))
		}
	}
	if err := e.change(next); err != nil {
		return "", err
	}
	return remap[id], nil
}
func (e *Editor) Undo() bool {
	if len(e.past) == 0 {
		return false
	}
	e.future = append(e.future, e.document)
	e.document = e.past[len(e.past)-1]
	e.past = e.past[:len(e.past)-1]
	return true
}
func (e *Editor) Redo() bool {
	if len(e.future) == 0 {
		return false
	}
	e.past = append(e.past, e.document)
	e.document = e.future[len(e.future)-1]
	e.future = e.future[:len(e.future)-1]
	return true
}
