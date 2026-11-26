package game

import (
	"fmt"
	"math"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

// Inspection is a read-only live snapshot; Spawned distinguishes disposable
// runtime objects from authored scaffolding without relying on names.
type Inspection struct {
	// EnterImpulseArmed reports whether this sensor can apply its configured impulse on entry.
	EnterImpulseArmed bool
	Entity            project.Entity
	Spawned           bool
	Body              *physics.BodyState
}

// Inspect returns live pose, ownership and an independent body-state snapshot.
func (s *Session) Inspect(id EntityID) (Inspection, error) {
	e, err := s.lookup(id)
	if err != nil {
		return Inspection{}, err
	}
	state, err := s.State(id)
	if err != nil {
		return Inspection{}, err
	}
	out := Inspection{Entity: state, Spawned: e.spawned}
	if e.description.Body != "none" {
		b, err := s.World.Body(e.body)
		if err != nil {
			return Inspection{}, err
		}
		_, out.EnterImpulseArmed = s.enterImpulses[e.body]
		out.Body = &b
		out.Entity.Position = b.Position
		pitch, yaw, roll := b.Rotation.ToEuler()
		out.Entity.Rotation = engine.Point{X: pitch * 180 / math.Pi, Y: yaw * 180 / math.Pi, Z: roll * 180 / math.Pi}
	}
	return out, nil
}

// Hierarchy returns current members, including queued spawns only after activation.
func (s *Session) Hierarchy() []project.TreeEntry { return s.document().Hierarchy() }

// Anchor is a generation-checked point in a body's local frame, captured at pick time.
type Anchor struct {
	Entity     EntityID
	LocalPoint engine.Point
}

// AnchorAt captures a world hit point in a dynamic body's local frame.
func (s *Session) AnchorAt(id EntityID, point engine.Point) (Anchor, error) {
	e, err := s.lookup(id)
	if err != nil {
		return Anchor{}, err
	}
	if e.description.Body != "dynamic" {
		return Anchor{}, fmt.Errorf("pick a dynamic collider")
	}
	b, err := s.World.Body(e.body)
	if err != nil {
		return Anchor{}, err
	}
	for _, v := range []float64{point.X, point.Y, point.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return Anchor{}, fmt.Errorf("invalid hit point")
		}
	}
	delta := engine.Point{X: point.X - b.Position.X, Y: point.Y - b.Position.Y, Z: point.Z - b.Position.Z}
	q := b.Rotation
	q.X = -q.X
	q.Y = -q.Y
	q.Z = -q.Z
	return Anchor{Entity: id, LocalPoint: q.RotateVector(delta)}, nil
}

// QueueImpulseAtAnchor follows a picked body's local point at the execution tick.
func (s *Session) QueueImpulseAtAnchor(anchor Anchor, impulse engine.Point) error {
	return s.queueImpulse(anchor.Entity, impulse, &anchor.LocalPoint)
}

// CheckSpawn validates a root primitive and a conservative collider-free placement.
// Boxes use enclosing spheres. Visual-only geometry does not block placement.
func (s *Session) CheckSpawn(d project.Entity, budget int) error {
	if budget < 1 || budget > project.MaxEntities {
		return fmt.Errorf("entity budget must be 1..%d", project.MaxEntities)
	}
	if d.Parent != "" || d.Body != "dynamic" {
		return fmt.Errorf("interactive spawn requires a dynamic root")
	}
	if err := s.validateSpawn(d); err != nil {
		return err
	}
	if len(s.Hierarchy()) >= budget {
		return fmt.Errorf("entity budget reached (%d)", budget)
	}
	radius := d.Size.X
	if d.Shape == "box" {
		radius = math.Sqrt(d.Size.X*d.Size.X + d.Size.Y*d.Size.Y + d.Size.Z*d.Size.Z)
	}
	hits, err := s.World.OverlapSphere(d.Position, radius, physics.QueryFilter{LayerMask: ^uint32(0), IncludeSensors: true})
	if err != nil {
		return err
	}
	if len(hits) > 0 {
		return fmt.Errorf("spawn blocked by collider (conservative bounds)")
	}
	return nil
}

// QueueCheckedSpawn rechecks placement when applied, including preceding spawns.
func (s *Session) QueueCheckedSpawn(d project.Entity, budget int) error {
	if err := s.CheckSpawn(d, budget); err != nil {
		return err
	}
	return s.enqueue(func() error {
		if err := s.CheckSpawn(d, budget); err != nil {
			return err
		}
		return s.spawnRuntime(d)
	})
}
func (s *Session) spawnRuntime(d project.Entity) error {
	id, err := s.spawn(d)
	if err != nil {
		return err
	}
	e, _ := s.lookup(id)
	e.spawned = true
	return nil
}

// QueueDeleteSpawned protects authored objects; all descendant handles are invalidated.
func (s *Session) QueueDeleteSpawned(id EntityID) error {
	e, err := s.lookup(id)
	if err != nil {
		return err
	}
	if !e.spawned {
		return fmt.Errorf("authored objects are protected; select a runtime spawn")
	}
	return s.QueueDestroy(id)
}

// SensorEvent records a successful impulse submission at the end of a physics tick.
// Sensor and Recipient are snapshot entity IDs and remain readable after deletion.
type SensorEvent struct {
	Tick              uint64
	Sensor, Recipient string
	Impulse           engine.Point
}

// SensorEvents returns the three most recent sensor actions in chronological order.
func (s *Session) SensorEvents() []SensorEvent { return append([]SensorEvent(nil), s.sensorEvents...) }
