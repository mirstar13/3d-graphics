// Package creator runs authored terminal-game scenes with project-owned Go behaviors.
//
// Open loads project.yaml and its entry scene. Supply a Behavior to read entity
// snapshots, inspect the preceding tick's contacts and queue impulses before
// each fixed physics update. The caller owns the game loop and calls Step once
// per simulation tick. Use one goroutine for a game and its world. Open the
// project again with fresh behavior state to restart from the saved scene.
package creator

import (
	"fmt"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/game"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

// Vec3 is a world-space vector with X, Y and Z components.
type Vec3 struct{ X, Y, Z float64 }

// Entity is a comparable, session-local handle. Find resolves an authored ID.
type Entity struct{ handle game.EntityID }

// State is a detached snapshot. Position uses metres, Rotation Euler degrees,
// and Velocity metres per second. Visual-only entities have zero velocity.
type State struct {
	ID, Name, Body               string
	Position, Rotation, Velocity Vec3
}

// Contact describes a pair from the most recently completed physics tick.
// Phase is one of ContactBegin, ContactStay or ContactEnd.
type Contact struct {
	A, B   Entity
	Phase  ContactPhase
	Sensor bool
}

// ContactPhase identifies a pair entering, maintaining or leaving contact.
type ContactPhase uint8

const (
	ContactBegin ContactPhase = iota
	ContactStay
	ContactEnd
)

// Tick identifies the upcoming fixed update, starting at one.
type Tick struct {
	Index uint64
	Delta time.Duration
}

// Behavior runs immediately before each physics tick. Return an error to stop
// the game before that tick runs. A closure may retain project-specific state.
type Behavior func(*World, Tick) error

// World exposes the running scene to behaviors.
type World struct{ session *game.Session }

// Find resolves a persistent scene entity ID.
func (w *World) Find(id string) (Entity, bool) {
	handle, ok := w.session.Find(id)
	return Entity{handle}, ok
}

// Inspect returns current live state, rejecting stale and foreign handles.
func (w *World) Inspect(e Entity) (State, error) {
	v, err := w.session.Inspect(e.handle)
	if err != nil {
		return State{}, err
	}
	s := State{ID: v.Entity.ID, Name: v.Entity.Name, Body: v.Entity.Body, Position: Vec3(v.Entity.Position), Rotation: Vec3(v.Entity.Rotation)}
	if v.Body != nil {
		s.Velocity = Vec3(v.Body.Velocity)
	}
	return s, nil
}

// Impulse queues a world-space impulse in newton-seconds on a dynamic body.
func (w *World) Impulse(e Entity, j Vec3) error {
	return w.session.QueueImpulse(e.handle, engine.Point(j))
}

// Contacts returns a detached list of the preceding tick's contact transitions.
func (w *World) Contacts() []Contact {
	events := w.session.World.ContactEvents()
	if len(events) == 0 {
		return nil
	}
	handles := w.session.BodyEntities()
	result := make([]Contact, 0, len(events))
	for _, ev := range events {
		a, okA := handles[ev.A]
		b, okB := handles[ev.B]
		if okA && okB {
			result = append(result, Contact{Entity{a}, Entity{b}, ContactPhase(ev.Phase), ev.Sensor})
		}
	}
	return result
}

// Game owns one isolated runtime and its behavior.
type Game struct {
	world    World
	behavior Behavior
	failure  error
	stepping bool
}

// Open loads the project's saved scene, resolves linked assets and applies scene
// physics overrides to the engine defaults. A nil behavior runs physics alone.
func Open(root string, b Behavior) (*Game, error) {
	p, err := project.Open(root)
	if err != nil {
		return nil, err
	}
	s, err := game.NewProjectSession(p.Root, p.Scene, physics.DefaultOptions())
	if err != nil {
		return nil, err
	}
	s.World.SetPaused(true)
	return &Game{world: World{s}, behavior: b}, nil
}

// World returns the live scene, including before the first tick.
func (g *Game) World() *World { return &g.world }

// Step invokes the behavior and simulates one tick. After a behavior or
// simulation error, subsequent calls return the failure without advancing.
func (g *Game) Step() error {
	if g.failure != nil {
		return g.failure
	}
	if g.stepping {
		return fmt.Errorf("recursive game step")
	}
	g.stepping = true
	defer func() { g.stepping = false }()
	s := g.world.session
	if g.behavior != nil {
		stats := s.World.Stats()
		if err := g.behavior(&g.world, Tick{stats.Tick + 1, stats.FixedStep}); err != nil {
			g.failure = fmt.Errorf("behavior tick %d: %w", stats.Tick+1, err)
			return g.failure
		}
	}
	g.failure = s.Step()
	return g.failure
}
