package physics

import (
	"fmt"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestKinematicPlatformTransfersMotion(t *testing.T) {
	w := newTestWorld(t)
	platform := NewBodyDesc(Kinematic, Box(vec{X: 3, Y: .5, Z: 3}))
	platform.Collider.Friction = 1
	platform.Collider.Restitution = 0
	pid := addTestBody(t, w, platform)
	d := NewBodyDesc(Dynamic, Box(vec{X: .5, Y: .5, Z: .5}))
	d.Position.Y = 1
	d.Collider.Friction = 1
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	for i := 0; i < 180; i++ {
		w.AdvanceWithHooks(w.options.FixedStep, StepHooks{BeforeStep: func(tick Tick) {
			if err := w.SetKinematicTarget(pid, vec{X: tick.Time * .5}, engine.IdentityQuaternion()); err != nil {
				t.Fatal(err)
			}
		}})
	}
	platformState := stateOf(t, w, pid)
	s := stateOf(t, w, id)
	if math.Abs(platformState.Position.X-1.5) > 1e-6 || s.Position.X < 1 || math.Abs(s.Position.Y-1) > .05 {
		t.Fatal(platformState.Position, s.Position)
	}
}
func TestAngularDynamicsAndDamping(t *testing.T) {
	w := noGravity(t)
	d := NewBodyDesc(Dynamic, Box(vec{X: 1, Y: 1.5, Z: 2}))
	d.LinearDamping = 0
	d.AngularDamping = 0
	d.AngularVelocity = vec{X: .2, Y: .3, Z: .4}
	id := addTestBody(t, w, d)
	b, _ := w.lookup(id)
	initial := worldInertia(b, b.state.AngularVelocity)
	stepN(t, w, 240)
	s := stateOf(t, w, id)
	final := worldInertia(b, s.AngularVelocity)
	if length(sub(initial, final))/length(initial) > .04 {
		t.Fatal("angular momentum drift", initial, final)
	}
	q := s.Rotation
	if math.Abs(q.W*q.W+q.X*q.X+q.Y*q.Y+q.Z*q.Z-1) > 1e-10 {
		t.Fatal("orientation not unit")
	}
	// Isotropic inertia makes the damping result analytically simple.
	w = noGravity(t)
	d = NewBodyDesc(Dynamic, Sphere(1))
	d.Velocity = vec{X: 4}
	d.AngularVelocity = vec{Y: 2}
	d.LinearDamping = .7
	d.AngularDamping = .4
	id = addTestBody(t, w, d)
	stepN(t, w, 60)
	s = stateOf(t, w, id)
	duration := 60 * w.options.FixedStep.Seconds()
	if math.Abs(s.Velocity.X-4*math.Exp(-.7*duration)) > 1e-8 || math.Abs(s.AngularVelocity.Y-2*math.Exp(-.4*duration)) > 1e-8 {
		t.Fatal(s)
	}
}
func TestBodiesIndependentOfRenderRate(t *testing.T) {
	var reference []BodyState
	for _, fps := range []int{30, 60, 144} {
		w := newTestWorld(t)
		floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
		floor.Collider.Restitution = 0
		addTestBody(t, w, floor)
		for i := 0; i < 3; i++ {
			d := NewBodyDesc(Dynamic, Sphere(.5))
			d.Position = vec{X: float64(i) * 2, Y: 2 + float64(i)}
			addTestBody(t, w, d)
		}
		total := 300 * w.options.FixedStep
		previous := time.Duration(0)
		for frame := 1; frame <= fps*5; frame++ {
			now := total * time.Duration(frame) / time.Duration(fps*5)
			if err := w.Advance(now-previous, nil); err != nil {
				t.Fatal(err)
			}
			previous = now
		}
		if reference == nil {
			reference = w.Bodies()
		} else if !reflect.DeepEqual(reference, w.Bodies()) {
			t.Fatalf("%d FPS changed body results", fps)
		}
	}
}
func tiledTerrain(t *testing.T) *TriangleMesh {
	t.Helper()
	var vertices []vec
	var indices []int
	for z := 0; z < 4; z++ {
		for x := 0; x < 4; x++ {
			first := len(vertices)
			vertices = append(vertices, vec{X: float64(x)*2 - 4, Z: float64(z)*2 - 4}, vec{X: float64(x)*2 - 2, Z: float64(z)*2 - 4}, vec{X: float64(x)*2 - 2, Z: float64(z)*2 - 2}, vec{X: float64(x)*2 - 4, Z: float64(z)*2 - 2})
			indices = append(indices, first, first+2, first+1, first, first+3, first+2)
		}
	}
	mesh, err := NewTriangleMesh(vertices, indices)
	if err != nil {
		t.Fatal(err)
	}
	return mesh
}
func TestMeshBVHMatchesBruteForceAndSeams(t *testing.T) {
	mesh := tiledTerrain(t)
	for i := 0; i < 20; i++ {
		q := bounds{min: vec{X: float64(i)*.3 - 4, Y: -1, Z: -1}, max: vec{X: float64(i)*.3 - 3, Y: 1, Z: 1}}
		candidates := map[int]bool{}
		for _, index := range mesh.query(q) {
			candidates[index] = true
		}
		for index := range mesh.triangles {
			b := emptyBounds()
			for _, p := range mesh.triangle(index) {
				b.include(p)
			}
			if b.overlaps(q) && !candidates[index] {
				t.Fatal("BVH missed triangle", index)
			}
		}
	}
	for _, shape := range []Shape{Sphere(.4), Box(vec{X: .4, Y: .4, Z: .4})} {
		w := newTestWorld(t)
		terrain := NewBodyDesc(Static, Mesh(mesh))
		terrain.Position = vec{Y: 1}
		terrain.Rotation = engine.QuaternionFromEuler(0, .4, 0)
		terrain.Collider.Friction = 0
		terrain.Collider.Restitution = 0
		addTestBody(t, w, terrain)
		d := NewBodyDesc(Dynamic, shape)
		d.Position = vec{X: -2, Y: 1.4}
		d.Velocity.X = 1
		d.LinearDamping = 0
		d.Collider.Friction = 0
		d.Collider.Restitution = 0
		id := addTestBody(t, w, d)
		for i := 0; i < 180; i++ {
			stepN(t, w, 1)
			s := stateOf(t, w, id)
			if math.Abs(s.Position.Y-1.4) > .05 || s.Velocity.X < .9 {
				t.Fatal("terrain seam snag or fall", shape.kind, s.Position, s.Velocity)
			}
		}
	}
}
func TestTiltedMeshAndBoxContacts(t *testing.T) {
	mesh := tiledTerrain(t)
	for _, shape := range []Shape{Sphere(.5), Box(vec{X: .5, Y: .5, Z: .5})} {
		w := newTestWorld(t)
		terrain := NewBodyDesc(Static, Mesh(mesh))
		terrain.Rotation = engine.QuaternionFromEuler(0, 0, .15)
		terrain.Collider.Friction = 1
		terrain.Collider.Restitution = 0
		addTestBody(t, w, terrain)
		d := NewBodyDesc(Dynamic, shape)
		d.Position.Y = 2
		d.Collider.Friction = 1
		d.Collider.Restitution = 0
		id := addTestBody(t, w, d)
		stepN(t, w, 120)
		s := stateOf(t, w, id)
		if s.Position.Y < -0.3 || len(w.Contacts()) == 0 {
			t.Fatal("ramp missed body", s.Position)
		}
	}
}
func TestContactSymmetryAndCacheInvalidation(t *testing.T) {
	a, b := collisionBodies(Box(vec{X: 1, Y: 1, Z: 1}), Box(vec{X: 1, Y: 1, Z: 1}), vec{X: 1.5})
	forward, reverse := collide(a, b), collide(b, a)
	if len(forward) != 4 || len(reverse) != 4 || dot(forward[0].Normal, reverse[0].Normal) > -.99 {
		t.Fatal("box manifold symmetry", forward, reverse)
	}
	w := newTestWorld(t)
	addTestBody(t, w, NewBodyDesc(Static, Plane(vec{Y: 1}, 0)))
	d := NewBodyDesc(Dynamic, Sphere(1))
	d.Position.Y = 1
	id := addTestBody(t, w, d)
	stepN(t, w, 2)
	if len(w.cache) == 0 {
		t.Fatal("no warm-start cache")
	}
	w.Teleport(id, vec{Y: 10}, engine.IdentityQuaternion())
	if len(w.cache) != 0 {
		t.Fatal("teleport left stale cache")
	}
	stepN(t, w, 1)
	if len(w.Contacts()) != 0 {
		t.Fatal("teleport left stale contact")
	}
}
func BenchmarkPrimitiveWorld(b *testing.B) {
	for _, count := range []int{16, 64, 128} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			w, _ := NewWorld(DefaultOptions())
			floor := NewBodyDesc(Static, Plane(vec{Y: 1}, 0))
			floor.Collider.Restitution = 0
			w.AddBody(floor)
			for i := 0; i < count; i++ {
				d := NewBodyDesc(Dynamic, Sphere(.5))
				d.Position = vec{X: float64(i%16) * 1.1, Y: .5, Z: float64(i/16) * 1.1}
				d.Collider.Restitution = 0
				w.AddBody(d)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w.Advance(w.options.FixedStep, nil)
			}
			b.StopTimer()
			s := w.SolverStats()
			b.ReportMetric(float64(s.CandidatePairs), "pairs/tick")
			b.ReportMetric(float64(s.Contacts), "contacts/tick")
		})
	}
}
