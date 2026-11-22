package physics

import (
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"math"
	"testing"
)

func boxHullGeometry(half, offset vec) ([]vec, []int) {
	b := &body{state: BodyState{BodyDesc: NewBodyDesc(Static, Box(half))}}
	poly := bodyPoly(b)
	vertices := append([]vec(nil), poly.vertices...)
	for i := range vertices {
		vertices[i] = add(vertices[i], offset)
	}
	var indices []int
	for _, f := range poly.faces {
		for i := 1; i+1 < len(f.indices); i++ {
			indices = append(indices, f.indices[0], f.indices[i], f.indices[i+1])
		}
	}
	return vertices, indices
}
func testHull(t *testing.T, half vec) Shape {
	t.Helper()
	v, i := boxHullGeometry(half, vec{})
	s, e := ConvexHull(v, i)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestHullValidationAndMassProperties(t *testing.T) {
	half := vec{X: 1, Y: 2, Z: 3}
	v, i := boxHullGeometry(half, vec{X: 10000, Y: -500, Z: 3})
	s, e := ConvexHull(v, i)
	if e != nil {
		t.Fatal(e)
	}
	if length(sub(s.CenterOfMass(), vec{X: 10000, Y: -500, Z: 3})) > 1e-7 {
		t.Fatal(s.CenterOfMass())
	}
	tensor := tensorFor(s, 2)
	expected := tensorFor(Box(half), 2)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if math.Abs(tensor[i][j]-expected[i][j]) > 1e-8 {
				t.Fatal(tensor, expected)
			}
		}
	}
	v[0].X += 100
	if s.hull.vertices[0].X != -1 {
		t.Fatal("hull retained mutable input")
	}
	cases := []struct {
		vertices []vec
		indices  []int
	}{{v, i[:len(i)-3]}, {append(v, v[1]), i}}
	for _, c := range cases {
		if _, e := ConvexHull(c.vertices, c.indices); e == nil {
			t.Fatal("invalid hull accepted")
		}
	}
	v, i = boxHullGeometry(vec{X: 1, Y: 1, Z: .001}, vec{})
	if _, e := ConvexHull(v, i); e != nil {
		t.Fatal("thin valid hull rejected", e)
	}
	v, i = boxHullGeometry(vec{X: 1, Y: 1, Z: 1}, vec{})
	v[0] = vec{}
	if _, e := ConvexHull(v, i); e == nil {
		t.Fatal("concave hull accepted")
	}
}
func TestHullPrimitiveParityAndSymmetry(t *testing.T) {
	half := vec{X: 1, Y: 1, Z: 1}
	h := testHull(t, half)
	for _, position := range []vec{{X: 3}, {X: 1.5}, {X: 1.9, Y: .2}, {}} {
		a, b := collisionBodies(Box(half), Box(half), position)
		a.state.Rotation = engine.QuaternionFromEuler(.1, .2, .05)
		baseline := collide(a, b)
		a.state.Collider.Shape = h
		actual := collide(a, b)
		reverse := collide(b, a)
		if (len(baseline) > 0) != (len(actual) > 0) || len(actual) != len(reverse) {
			t.Fatal(position, baseline, actual, reverse)
		}
		if len(actual) > 0 && dot(actual[0].Normal, reverse[0].Normal) > -.99 {
			t.Fatal("hull normal not reversed")
		}
	}
	a, b := collisionBodies(Sphere(.5), h, vec{})
	cs := collide(a, b)
	if len(cs) != 1 || cs[0].Separation >= 0 {
		t.Fatal("contained sphere missed", cs)
	}
}
func TestHullStacksAndMeshSeams(t *testing.T) {
	for _, meshFloor := range []bool{false, true} {
		w := newTestWorld(t)
		w.options.Sleeping = false
		shape := Plane(vec{Y: 1}, 0)
		if meshFloor {
			shape = Mesh(tiledTerrain(t))
		}
		f := NewBodyDesc(Static, shape)
		f.Collider.Restitution = 0
		addTestBody(t, w, f)
		hull := testHull(t, vec{X: .5, Y: .5, Z: .5})
		var ids []BodyID
		for i := 0; i < 3; i++ {
			d := NewBodyDesc(Dynamic, hull)
			d.Position.Y = .5 + float64(i)
			d.Collider.Restitution = 0
			ids = append(ids, addTestBody(t, w, d))
		}
		stepN(t, w, 600)
		for i, id := range ids {
			s := stateOf(t, w, id)
			if math.Abs(s.Position.Y-(.5+float64(i))) > .08 || length(s.Velocity) > .1 {
				t.Fatal("hull stack unstable", meshFloor, s)
			}
		}
	}
	w := newTestWorld(t)
	f := NewBodyDesc(Static, Mesh(tiledTerrain(t)))
	f.Collider.Friction = 0
	f.Collider.Restitution = 0
	addTestBody(t, w, f)
	d := NewBodyDesc(Dynamic, testHull(t, vec{X: .4, Y: .4, Z: .4}))
	d.Position = vec{X: -2, Y: .4}
	d.Velocity.X = 1
	d.LinearDamping = 0
	d.Collider.Friction = 0
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	for i := 0; i < 180; i++ {
		stepN(t, w, 1)
		s := stateOf(t, w, id)
		if math.Abs(s.Position.Y-.4) > .05 || s.Velocity.X < .9 {
			t.Fatal("hull snagged seam", s)
		}
	}
}
func TestCompoundConvexHull(t *testing.T) {
	h := testHull(t, vec{X: .3, Y: .5, Z: .4})
	s, e := Compound([]CompoundChild{{Shape: h, Position: vec{X: -1}, Rotation: engine.IdentityQuaternion(), Weight: 1}, {Shape: h, Position: vec{X: 1}, Rotation: engine.QuaternionFromEuler(0, .2, 0), Weight: 1}})
	if e != nil {
		t.Fatal(e)
	}
	w := newTestWorld(t)
	addTestBody(t, w, NewBodyDesc(Static, Plane(vec{Y: 1}, 0)))
	d := NewBodyDesc(Dynamic, s)
	d.Position.Y = 2
	id := addTestBody(t, w, d)
	stepN(t, w, 240)
	if stateOf(t, w, id).Position.Y < .4 || len(w.Contacts()) < 2 {
		t.Fatal("compound hull fell through")
	}
}

func TestTetrahedronMassAndTerrain(t *testing.T) {
	s, e := ConvexHull([]vec{{}, {X: 1}, {Y: 1}, {Z: 1}}, []int{0, 2, 1, 0, 1, 3, 0, 3, 2, 1, 2, 3})
	if e != nil {
		t.Fatal(e)
	}
	if length(sub(s.CenterOfMass(), vec{X: .25, Y: .25, Z: .25})) > 1e-10 {
		t.Fatal(s.CenterOfMass())
	}
	tensor := tensorFor(s, 1)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			want := .0125
			if i == j {
				want = .075
			}
			if math.Abs(tensor[i][j]-want) > 1e-10 {
				t.Fatal(tensor)
			}
		}
	}
	w := newTestWorld(t)
	addTestBody(t, w, NewBodyDesc(Static, Mesh(tiledTerrain(t))))
	d := NewBodyDesc(Dynamic, s)
	d.Position.Y = 2
	d.Collider.Restitution = 0
	id := addTestBody(t, w, d)
	stepN(t, w, 300)
	state := stateOf(t, w, id)
	if state.Position.Y < .15 || state.Position.Y > 1 || !finitePoint(state.Velocity) {
		t.Fatal("irregular hull terrain failure", state)
	}
}
