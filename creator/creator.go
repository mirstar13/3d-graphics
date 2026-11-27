// Package creator runs authored terminal-game scenes with project-owned Go behaviors.
//
// Open loads project.yaml and its entry scene. Supply a Behavior to read entity
// snapshots, inspect the preceding tick's contacts and queue impulses before
// each fixed physics update. The caller owns the game loop and calls Step once
// per simulation tick. Use one goroutine for a game and its world. Open the
// project again with fresh behavior state to restart from the saved scene.
package creator

import (
	"context"
	"fmt"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/audio"
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
type World struct {
	session *game.Session
	owner   *Game
}

// ActionPressed reports a project action in the current behavior invocation.
// Map input tokens to actions under gameplay_bindings in project.yaml.
func (w *World) ActionPressed(action string) bool {
	if action == "" || !w.owner.inBehavior {
		return false
	}
	for _, key := range w.owner.keys {
		if w.owner.project.Manifest.GameplayBindings[key] == action {
			return true
		}
	}
	return false
}

// Pick returns the nearest entity along a finite world-space ray.
func (w *World) Pick(origin, direction Vec3, distance float64) (Entity, bool, error) {
	hit, ok, err := w.session.Pick(physics.Ray{Origin: engine.Point(origin), Direction: engine.Point(direction), MaxDistance: distance})
	return Entity{hit.Entity}, ok, err
}

// ChangeScene validates a project-relative destination immediately. On success
// the new scene replaces this one after its current tick finishes. Handle
// errors in the behavior to continue playing the current scene.
func (w *World) ChangeScene(path string) error {
	g := w.owner
	if !g.inBehavior || g.nextSession != nil {
		return fmt.Errorf("one scene change can be queued by the active behavior")
	}
	session, err := g.loadScene(path)
	if err != nil {
		return err
	}
	g.nextSession, g.nextPath = session, path
	return nil
}

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

// PlaySound requests a named project cue during the current behavior tick.
func (w *World) PlaySound(cue string) error {
	if !w.owner.inBehavior {
		return fmt.Errorf("PlaySound requires an active behavior tick")
	}
	return w.session.QueueSound(cue)
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
	audio       *audio.Manager
	sounds      map[string]*audio.Clip
	world       World
	behavior    Behavior
	failure     error
	stepping    bool
	inBehavior  bool
	project     *project.Project
	scenePath   string
	nextPath    string
	nextSession *game.Session
	keys        []string
	capture     bool
	dx, dy      int
}

// Open loads the project's saved scene, resolves linked assets and applies scene
// physics overrides to the engine defaults. A nil callback still runs any attached project behaviors.
func Open(root string, b Behavior) (*Game, error) {
	p, err := project.Open(root)
	if err != nil {
		return nil, err
	}
	sounds, err := p.LoadSounds()
	if err != nil {
		return nil, err
	}
	s, err := newSession(p, p.Scene, sounds)
	if err != nil {
		return nil, err
	}
	g := &Game{project: p, sounds: sounds, scenePath: p.Manifest.EntryScene, behavior: b}
	g.world = World{session: s, owner: g}
	return g, nil
}

func newSession(p *project.Project, scene project.Document, sounds map[string]*audio.Clip) (*game.Session, error) {
	s, err := game.NewProjectSession(p.Root, scene, physics.DefaultOptions())
	if err == nil {
		err = s.ConfigureAudio(sounds)
	}
	if err == nil {
		s.World.SetPaused(true)
		s.EnableBehaviors(context.Background(), p)
	}
	return s, err
}

func (g *Game) loadScene(path string) (*game.Session, error) {
	scene, err := g.project.LoadScene(path)
	if err != nil {
		return nil, err
	}
	return newSession(g.project, scene, g.sounds)
}

// Scene returns the project-relative path of the current room.
func (g *Game) Scene() string { return g.scenePath }

// Restart reloads the current room and installs fresh behavior. Existing
// entity handles expire. A failed reload leaves the current room intact.
func (g *Game) Restart(b Behavior) error { return g.restart(g.scenePath, b) }

func (g *Game) restart(path string, b Behavior) error {
	if g.stepping {
		return fmt.Errorf("cannot restart during a game step")
	}
	s, err := g.loadScene(path)
	if err != nil {
		return err
	}
	g.world.session.Close()
	s.AttachAudio(g.audio)
	g.world.session, g.behavior, g.failure, g.scenePath = s, b, nil, path
	g.capture = false
	return nil
}

// Close releases the project's external behavior worker. Use it when finished
// with the game. RunTerminal closes the worker when the terminal session exits.
func (g *Game) Close() {
	g.world.session.Close()
	if g.audio != nil {
		g.audio.Close()
		g.audio = nil
	}
}

// World returns the live scene, including before the first tick.
func (g *Game) World() *World { return &g.world }

// Step invokes the behavior and simulates one tick. After a behavior or
// simulation error, subsequent calls return the failure without advancing.
func (g *Game) Step() error { return g.StepWithKeys() }

// StepWithKeys delivers input tokens for this tick only.
func (g *Game) StepWithKeys(keys ...string) error {
	return g.StepWithInput(Input{Keys: keys})
}

// Input carries one tick of key presses and relative terminal cell motion.
// Positive X points right and positive Y points down.
type Input struct {
	Keys           []string
	MouseX, MouseY int
}

// CaptureMouse requests terminal motion reporting, or releases it.
func (w *World) CaptureMouse(on bool) { w.owner.capture = on }

// MouseCaptured reports whether the game currently requests pointer motion.
func (g *Game) MouseCaptured() bool { return g.capture }

// MouseDelta returns accumulated motion for this behavior invocation.
func (w *World) MouseDelta() (int, int) { return w.owner.dx, w.owner.dy }

// CameraRay returns the current eye and forward direction.
func (w *World) CameraRay() (Vec3, Vec3) {
	c := w.session.Scene.Camera
	return Vec3(c.GetPosition()), Vec3(c.GetForwardVectorPoint())
}

// TurnCamera rotates relative to the current view in radians. Positive angles
// turn right and up. Pitch stops just short of the poles.
func (w *World) TurnCamera(right, up float64) {
	w.session.Scene.Camera.Turn(right, up)
}

// MouseLook applies this project's configured sensitivity to captured motion.
func (w *World) MouseLook() {
	right, up := w.owner.project.Manifest.MouseLook.Angles(w.MouseDelta())
	w.TurnCamera(right, up)
}

// SetCameraPosition translates the camera without changing its orientation.
func (w *World) SetCameraPosition(eye Vec3) {
	w.session.Scene.Camera.SetPosition(eye.X, eye.Y, eye.Z)
}

// StepWithInput delivers motion only when capture was active at tick start.
func (g *Game) StepWithInput(in Input) error {
	if g.failure != nil || g.world.session.Objective().Finished() {
		return g.failure
	}
	if g.stepping {
		return fmt.Errorf("recursive game step")
	}
	g.stepping = true
	if g.capture {
		g.dx, g.dy = in.MouseX, in.MouseY
	}
	defer func() {
		g.dx, g.dy = 0, 0
		g.stepping, g.inBehavior = false, false
		g.keys, g.nextSession, g.nextPath = nil, nil, ""
	}()
	s := g.world.session
	defer s.DiscardSounds()
	if g.behavior != nil {
		stats := s.World.Stats()
		g.inBehavior, g.keys = true, in.Keys
		err := g.behavior(&g.world, Tick{stats.Tick + 1, stats.FixedStep})
		g.inBehavior = false
		if err != nil {
			g.failure = fmt.Errorf("behavior tick %d: %w", stats.Tick+1, err)
			return g.failure
		}
	}
	s.BehaviorInput(in.Keys...)
	g.failure = s.Step()
	if g.failure == nil && g.nextSession != nil {
		s.Close()
		g.nextSession.AttachAudio(g.audio)
		g.world.session, g.scenePath = g.nextSession, g.nextPath
		g.capture = false
	}
	return g.failure
}

// Objective describes the HUD for the current round. Outcome is "Playing",
// "Won", "Lost", or empty for a scene without an objective.
type Objective struct {
	Text, Signal, Outcome string
	Tick, TickLimit       uint64
}

// Objective returns a detached snapshot. Step holds the terminal tick after a
// win or loss. Open a fresh Game and behavior to start another round.
func (g *Game) Objective() Objective {
	o := g.world.session.Objective()
	return Objective{o.Text, o.Signal, string(o.Outcome), o.Tick, o.TickLimit}
}
