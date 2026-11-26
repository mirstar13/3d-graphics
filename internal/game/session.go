package game

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

// EntityID is a session-local, generation-checked live handle, never a file ID.
type EntityID struct {
	session          uint64
	slot, generation uint32
}
type entity struct {
	description project.Entity
	node        *engine.SceneNode
	body        physics.BodyID
	generation  uint32
	active      bool
	spawned     bool
}

var sessionSequence atomic.Uint64

// Session owns live entities and a world; all operations belong to the update goroutine.
type Session struct {
	// InitialCamera holds the authored view used to restore transient editor navigation.
	InitialCamera project.CameraSettings
	Scene         *engine.Scene
	Lighting      *engine.LightingSystem
	World         *physics.World
	id            uint64
	entities      []entity
	pending       []func() error
	lastError     error
}

// NewSession constructs an independent runtime from authored data and application physics defaults.
func NewSession(d project.Document, options physics.Options) (*Session, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if v := d.Simulation; v != nil {
		if v.GravityX != nil {
			options.Gravity.X = *v.GravityX
		}
		if v.GravityY != nil {
			options.Gravity.Y = *v.GravityY
		}
		if v.GravityZ != nil {
			options.Gravity.Z = *v.GravityZ
		}
		if v.Sleeping != nil {
			options.Sleeping = *v.Sleeping
		}
		if v.DefaultCCD != nil {
			options.DefaultCCD = *v.DefaultCCD
		}
		if v.VelocityIterations != nil {
			options.VelocityIterations = *v.VelocityIterations
		}
		if v.PositionIterations != nil {
			options.PositionIterations = *v.PositionIterations
		}
	}
	world, err := physics.NewWorld(options)
	if err != nil {
		return nil, err
	}
	scene := engine.NewScene()
	c := project.DefaultCamera()
	if d.Camera != nil {
		c = *d.Camera
	}
	scene.Camera.SetPosition(c.Position.X, c.Position.Y, c.Position.Z)
	scene.Camera.LookAt(c.Target)
	scene.Camera.SetFOV(c.FOVX, c.FOVY)
	scene.Camera.Near, scene.Camera.Far = c.Near, c.Far
	s := &Session{InitialCamera: c, Scene: scene, World: world, id: sessionSequence.Add(1)}
	s.Lighting = engine.NewLightingSystem(scene.Camera)
	lighting := project.DefaultLighting()
	if d.Lighting != nil {
		lighting = *d.Lighting
	}
	s.Lighting.AmbientLight, s.Lighting.AmbientIntensity = lighting.Ambient, lighting.Intensity
	for _, p := range lighting.Lights {
		light := engine.NewLight(p.Position.X, p.Position.Y, p.Position.Z, p.Color, p.Intensity)
		light.IsEnabled = p.Enabled
		s.Lighting.AddLight(light)
	}
	for _, entry := range d.Hierarchy() {
		if _, err = s.spawn(entry.Entity); err != nil {
			return nil, err
		}
	}
	return s, nil
}
func (s *Session) spawn(d project.Entity) (EntityID, error) {
	for _, e := range s.entities {
		if e.active && e.description.ID == d.ID {
			return EntityID{}, fmt.Errorf("duplicate authored identity")
		}
	}
	var parent *engine.SceneNode
	if d.Parent != "" {
		id, ok := s.Find(d.Parent)
		if !ok {
			return EntityID{}, fmt.Errorf("missing parent %s", d.Parent)
		}
		entry, _ := s.lookup(id)
		parent = entry.node
	}
	m := engine.NewMaterial()
	m.DiffuseColor = d.Color
	m.AmbientStrength = .6
	m.SpecularStrength = .2
	var node *engine.SceneNode
	if d.Shape == "box" {
		node = s.Scene.CreateCube("entity:"+d.ID, 1, &m)
		mesh := node.Object.(*engine.Mesh)
		for i, v := range mesh.Vertices {
			mesh.Vertices[i] = engine.Point{X: v.X * d.Size.X, Y: v.Y * d.Size.Y, Z: v.Z * d.Size.Z}
		}
	} else {
		mesh := engine.GenerateSphere(d.Size.X, 8, 12)
		mesh.Material = &m
		node = engine.NewSceneNodeWithObject("entity:"+d.ID, mesh)
		s.Scene.AddNode(node)
	}
	if parent != nil {
		parent.AddChild(node)
	}
	q := engine.QuaternionFromEuler(d.Rotation.X*math.Pi/180, d.Rotation.Y*math.Pi/180, d.Rotation.Z*math.Pi/180)
	node.Transform.SetPosition(d.Position.X, d.Position.Y, d.Position.Z)
	node.Transform.SetRotationQuaternion(q)
	entry := entity{description: d, node: node, generation: 1, active: true}
	if d.Body != "none" {
		kind := physics.Static
		if d.Body == "dynamic" {
			kind = physics.Dynamic
		}
		shape := physics.Box(d.Size)
		if d.Shape == "sphere" {
			shape = physics.Sphere(d.Size.X)
		}
		desc := physics.NewBodyDesc(kind, shape)
		desc.Position = d.Position
		desc.Rotation = q
		desc.Mass = d.Mass
		id, err := s.World.AddBody(desc)
		if err != nil {
			s.Scene.RemoveNode(node)
			return EntityID{}, err
		}
		entry.body = id
	}
	slot := len(s.entities)
	for i, old := range s.entities {
		if !old.active {
			slot = i
			entry.generation = old.generation
			break
		}
	}
	if slot == len(s.entities) {
		s.entities = append(s.entities, entry)
	} else {
		s.entities[slot] = entry
	}
	return EntityID{s.id, uint32(slot), entry.generation}, nil
}
func (s *Session) lookup(id EntityID) (*entity, error) {
	if id.session != s.id || int(id.slot) >= len(s.entities) {
		return nil, fmt.Errorf("invalid entity handle")
	}
	e := &s.entities[id.slot]
	if !e.active || e.generation != id.generation {
		return nil, fmt.Errorf("stale entity handle")
	}
	return e, nil
}

// Find resolves a persistent document identity to a current live handle.
func (s *Session) Find(fileID string) (EntityID, bool) {
	for i, e := range s.entities {
		if e.active && e.description.ID == fileID {
			return EntityID{s.id, uint32(i), e.generation}, true
		}
	}
	return EntityID{}, false
}

// State returns a description with live world position and Euler rotation (degrees).
// Parent remains the persistent hierarchy link; this snapshot is not a scene file.
func (s *Session) State(id EntityID) (project.Entity, error) {
	e, err := s.lookup(id)
	if err != nil {
		return project.Entity{}, err
	}
	d := e.description
	d.Position = e.node.Transform.GetWorldPosition()
	r := e.node.Transform.GetWorldRotation()
	d.Rotation = engine.Point{X: r.X * 180 / math.Pi, Y: r.Y * 180 / math.Pi, Z: r.Z * 180 / math.Pi}
	return d, nil
}

// QueueImpulse applies a world-space impulse once at the next simulation tick.
func (s *Session) QueueImpulse(id EntityID, impulse engine.Point) error {
	return s.queueImpulse(id, impulse, nil)
}
func (s *Session) queueImpulse(id EntityID, impulse engine.Point, localPoint *engine.Point) error {
	e, err := s.lookup(id)
	if err != nil {
		return err
	}
	if e.description.Body != "dynamic" {
		return fmt.Errorf("select a dynamic body")
	}
	for _, v := range []float64{impulse.X, impulse.Y, impulse.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid impulse")
		}
	}
	var anchor engine.Point
	if localPoint != nil {
		anchor = *localPoint
		for _, v := range []float64{anchor.X, anchor.Y, anchor.Z} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("invalid local anchor")
			}
		}
	}
	return s.enqueue(func() error {
		e, err := s.lookup(id)
		if err != nil {
			return err
		}
		if localPoint != nil {
			b, err := s.World.Body(e.body)
			if err != nil {
				return err
			}
			offset := b.Rotation.RotateVector(anchor)
			point := engine.Point{X: b.Position.X + offset.X, Y: b.Position.Y + offset.Y, Z: b.Position.Z + offset.Z}
			return s.World.AddImpulseAtPoint(e.body, impulse, point)
		}
		return s.World.AddImpulse(e.body, impulse)
	})
}
func (s *Session) enqueue(command func() error) error {
	if len(s.pending) >= 64 {
		return fmt.Errorf("command queue full")
	}
	s.pending = append(s.pending, command)
	return nil
}

// QueueSpawn creates a validated primitive at the next tick; Find resolves it afterward.
func (s *Session) QueueSpawn(d project.Entity) error {
	if err := s.validateSpawn(d); err != nil {
		return err
	}
	return s.enqueue(func() error {
		if err := s.validateSpawn(d); err != nil {
			return err
		}
		return s.spawnRuntime(d)
	})
}

// QueueDestroy removes visual and physics state together at a tick boundary.
func (s *Session) QueueDestroy(id EntityID) error {
	if _, err := s.lookup(id); err != nil {
		return err
	}
	return s.enqueue(func() error {
		e, err := s.lookup(id)
		if err != nil {
			return err
		}
		document := s.document()
		ids := document.SubtreeIDs(e.description.ID)
		// Remove leaves first, clearing scene registry entries and all handles.
		for i := len(ids) - 1; i >= 0; i-- {
			childID, _ := s.Find(ids[i])
			child, _ := s.lookup(childID)
			if child.description.Body != "none" {
				if err := s.World.RemoveBody(child.body); err != nil {
					return err
				}
			}
			s.Scene.RemoveNode(child.node)
			child.active = false
			child.generation++
		}
		return nil
	})
}
func (s *Session) hooks() physics.StepHooks {
	return physics.StepHooks{AfterStep: func(t physics.Tick) {
		for i := range s.entities {
			e := &s.entities[i]
			if !e.active || e.description.Body == "none" {
				continue
			}
			state, err := s.World.Body(e.body)
			if err != nil {
				continue
			}
			e.node.Transform.SetPosition(state.Position.X, state.Position.Y, state.Position.Z)
			e.node.Transform.SetRotationQuaternion(state.Rotation)
		}
		s.Scene.UpdateVisuals(t.Delta)
	}}
}

// Advance runs fixed updates; commands stay pending while paused.
func (s *Session) Advance(elapsed time.Duration) error {
	s.lastError = nil
	stats := s.World.Stats()
	if elapsed > 0 && !stats.Paused && elapsed >= stats.FixedStep-stats.Remainder {
		s.applyCommands()
	}
	if err := s.World.AdvanceWithHooks(elapsed, s.hooks()); err != nil {
		return err
	}
	return s.lastError
}
func (s *Session) Step() error {
	s.lastError = nil
	if s.World.Stats().Paused {
		s.applyCommands()
	}
	if err := s.World.StepOnceWithHooks(s.hooks()); err != nil {
		return err
	}
	return s.lastError
}
func (s *Session) PendingCommands() int { return len(s.pending) }

func (s *Session) applyCommands() {
	pending := s.pending
	s.pending = nil
	for _, command := range pending {
		if err := command(); err != nil {
			s.lastError = err
		}
	}
}

// SelectionLines returns an oriented bounds overlay without changing simulation.
func (s *Session) SelectionLines(id EntityID) []engine.DebugLine {
	e, err := s.lookup(id)
	if err != nil {
		return nil
	}
	half := e.description.Size
	if e.description.Shape == "sphere" {
		half = engine.Point{X: half.X, Y: half.X, Z: half.X}
	}
	var vertices [8]engine.Point
	for i := 0; i < 8; i++ {
		p := engine.Point{X: -half.X, Y: -half.Y, Z: -half.Z}
		if i&1 != 0 {
			p.X = half.X
		}
		if i&2 != 0 {
			p.Y = half.Y
		}
		if i&4 != 0 {
			p.Z = half.Z
		}
		vertices[i] = e.node.Transform.TransformPoint(p)
	}
	var lines []engine.DebugLine
	for i := 0; i < 8; i++ {
		for bit := 0; bit < 3; bit++ {
			j := i ^ (1 << bit)
			if i < j {
				lines = append(lines, engine.DebugLine{Start: vertices[i], End: vertices[j], Color: engine.ColorWhite})
			}
		}
	}
	return lines
}

// document collects authored descriptions for hierarchy validation, not live poses.
func (s *Session) document() project.Document {
	d := project.Document{Version: project.Version, Name: "Runtime"}
	for _, e := range s.entities {
		if e.active {
			d.Entities = append(d.Entities, e.description)
		}
	}
	return d
}
func (s *Session) validateSpawn(e project.Entity) error {
	d := s.document()
	d.Entities = append(d.Entities, e)
	return d.Validate()
}
