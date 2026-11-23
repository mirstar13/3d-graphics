package physics

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

var allQueryLayers = QueryFilter{LayerMask: ^uint32(0), IncludeSensors: true}

func TestRayShapeGeometry(t *testing.T) {
	mesh, err := NewTriangleMesh([]vec{{X: -2, Y: -2}, {X: 2, Y: -2}, {Y: 2}}, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}
	compound, err := Compound([]CompoundChild{{Shape: Sphere(1), Position: vec{X: -2}, Rotation: engine.IdentityQuaternion(), Weight: 1}, {Shape: Sphere(1), Position: vec{X: 2}, Rotation: engine.IdentityQuaternion(), Weight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		shape    Shape
		ray      Ray
		distance float64
		normal   vec
		child    int
	}{
		{"sphere", Sphere(1), Ray{Origin: vec{Z: -3}, Direction: vec{Z: 4}, MaxDistance: 10}, 2, vec{Z: -1}, -1},
		{"box", Box(vec{X: 1, Y: 1, Z: 1}), Ray{Origin: vec{Z: -3}, Direction: vec{Z: 1}, MaxDistance: 2}, 2, vec{Z: -1}, -1},
		{"hull", testHull(t, vec{X: 1, Y: 1, Z: 1}), Ray{Origin: vec{Z: -3}, Direction: vec{Z: 1}, MaxDistance: 10}, 2, vec{Z: -1}, -1},
		{"plane", Plane(vec{Y: 2}, 2), Ray{Origin: vec{Y: 3}, Direction: vec{Y: -1}, MaxDistance: 10}, 2, vec{Y: 1}, -1},
		{"mesh", Mesh(mesh), Ray{Origin: vec{Z: 3}, Direction: vec{Z: -1}, MaxDistance: 10}, 3, vec{Z: 1}, -1},
		{"compound", compound, Ray{Origin: vec{X: 2, Z: -3}, Direction: vec{Z: 1}, MaxDistance: 10}, 2, vec{Z: -1}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rotation := engine.QuaternionFromEuler(.2, .6, -.3)
			offset := vec{X: 4, Y: -2, Z: 7}
			ray := tc.ray
			ray.Origin = add(offset, rotation.RotateVector(ray.Origin))
			ray.Direction = rotation.RotateVector(ray.Direction)
			hit, ok, err := RaycastShape(ray, tc.shape, offset, rotation)
			if err != nil || !ok || math.Abs(hit.Distance-tc.distance) > 1e-8 || length(sub(hit.Normal, rotation.RotateVector(tc.normal))) > 1e-8 || hit.Child != tc.child {
				t.Fatal(hit, ok, err)
			}
		})
	}
	for _, shape := range []Shape{Sphere(1), Box(vec{X: 1, Y: 1, Z: 1}), testHull(t, vec{X: 1, Y: 1, Z: 1}), Plane(vec{Y: 1}, 0)} {
		hit, ok, err := RaycastShape(Ray{Direction: vec{Z: 1}, MaxDistance: 0}, shape, vec{}, engine.IdentityQuaternion())
		if err != nil || !ok || hit.Distance != 0 || hit.Normal != (vec{Z: -1}) {
			t.Fatal("inside", hit, ok, err)
		}
	}
	for _, r := range []Ray{{Origin: vec{X: 2, Z: -3}, Direction: vec{Z: 1}, MaxDistance: 10}, {Origin: vec{Z: -3}, Direction: vec{Z: 1}, MaxDistance: 1}, {Origin: vec{Z: -3}, Direction: vec{Z: -1}, MaxDistance: 10}} {
		if _, ok, err := RaycastShape(r, Box(vec{X: 1, Y: 1, Z: 1}), vec{}, engine.IdentityQuaternion()); err != nil || ok {
			t.Fatal("miss", ok, err)
		}
	}
}
func TestWorldQueriesFiltersTiesAndPurity(t *testing.T) {
	options := DefaultOptions()
	options.Gravity = vec{}
	w, _ := NewWorld(options)
	desc := NewBodyDesc(Dynamic, Sphere(1))
	first, err := w.AddBody(desc)
	if err != nil {
		t.Fatal(err)
	}
	// Let the dynamic body sleep before adding coincident query targets.
	for i := 0; i < 120; i++ {
		if err = w.Advance(options.FixedStep, nil); err != nil {
			t.Fatal(err)
		}
	}
	state, _ := w.Body(first)
	if !state.Sleeping {
		t.Fatal("fixture failed to sleep")
	}
	desc.Type = Static
	second, _ := w.AddBody(desc)
	desc.Collider.Layer = 2
	desc.Collider.Sensor = true
	desc.Position = vec{Z: -1}
	sensor, _ := w.AddBody(desc)
	desc.Position = vec{Z: -2}
	desc.Disabled = true
	w.AddBody(desc)
	before, stats, contacts := w.Bodies(), w.Stats(), append([]Contact(nil), w.contacts...)
	ray := Ray{Origin: vec{Z: -5}, Direction: vec{Z: 1}, MaxDistance: 10}
	hit, ok, err := w.Raycast(ray, QueryFilter{LayerMask: 1})
	if err != nil || !ok || hit.Body != first {
		t.Fatal("stable tie", hit, err)
	}
	hit, ok, err = w.Raycast(ray, allQueryLayers)
	if err != nil || !ok || hit.Body != sensor {
		t.Fatal("sensor", hit, err)
	}
	if _, ok, _ = w.Raycast(ray, QueryFilter{LayerMask: 2}); ok {
		t.Fatal("sensor included by default")
	}
	if _, ok, _ = w.Raycast(ray, QueryFilter{}); ok {
		t.Fatal("zero mask hit")
	}
	hits, err := w.OverlapSphere(vec{X: 2}, 1, QueryFilter{LayerMask: 1})
	if err != nil || len(hits) != 2 || hits[0].Body != first || hits[1].Body != second {
		t.Fatal("touching/order", hits, err)
	}
	hits, err = w.OverlapSphere(vec{X: 2.00001}, 1, QueryFilter{LayerMask: 1})
	if err != nil || len(hits) != 0 {
		t.Fatal("contact slop became overlap", hits, err)
	}
	if !reflect.DeepEqual(before, w.Bodies()) || stats != w.Stats() || !reflect.DeepEqual(contacts, w.contacts) {
		t.Fatal("query mutated simulation")
	}
	w.RemoveBody(first)
	hit, ok, err = w.Raycast(ray, QueryFilter{LayerMask: 1})
	if err != nil || !ok || hit.Body != second {
		t.Fatal("stale removed hit")
	}
}
func TestMeshQueriesAgainstBruteForce(t *testing.T) {
	var vertices []vec
	var indices []int
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			base := len(vertices)
			vertices = append(vertices, vec{X: float64(x), Y: float64(y)}, vec{X: float64(x + 1), Y: float64(y)}, vec{X: float64(x), Y: float64(y + 1)})
			indices = append(indices, base, base+1, base+2)
		}
	}
	mesh, err := NewTriangleMesh(vertices, indices)
	if err != nil {
		t.Fatal(err)
	}
	random := rand.New(rand.NewSource(7))
	for i := 0; i < 400; i++ {
		ray := Ray{Origin: vec{X: random.Float64()*10 - 1, Y: random.Float64()*10 - 1, Z: -3}, Direction: unit(vec{X: random.Float64() - .5, Y: random.Float64() - .5, Z: 1}), MaxDistance: 20}
		distance, _, hit := rayLocal(ray, Mesh(mesh))
		best := math.Inf(1)
		for triangle := range mesh.triangles {
			d, _, ok := rayTriangle(ray, mesh.triangle(triangle))
			if ok {
				best = math.Min(best, d)
			}
		}
		if hit != !math.IsInf(best, 1) || hit && math.Abs(distance-best) > 1e-10 {
			t.Fatal("BVH disagrees", ray, distance, best)
		}
	}
	w, _ := NewWorld(DefaultOptions())
	w.AddBody(NewBodyDesc(Static, Mesh(mesh)))
	hits, err := w.OverlapSphere(vec{X: .2, Y: .2, Z: .1}, .2, allQueryLayers)
	if err != nil || len(hits) != 1 {
		t.Fatal("mesh overlap", hits, err)
	}
}
func TestQueryValidationAndCompoundOverlap(t *testing.T) {
	w, _ := NewWorld(DefaultOptions())
	for _, ray := range []Ray{{}, {Direction: vec{Z: 1}, MaxDistance: -1}, {Direction: vec{X: math.NaN()}, MaxDistance: 1}, {Origin: vec{X: math.Inf(1)}, Direction: vec{Z: 1}, MaxDistance: 1}} {
		if _, _, err := w.Raycast(ray, allQueryLayers); err == nil {
			t.Fatal("invalid ray accepted", ray)
		}
	}
	if _, err := w.OverlapSphere(vec{}, 0, allQueryLayers); err == nil {
		t.Fatal("invalid overlap")
	}
	compound, err := Compound([]CompoundChild{{Shape: Sphere(1), Position: vec{X: -2}, Rotation: engine.IdentityQuaternion(), Weight: 1}, {Shape: Sphere(1), Position: vec{X: 2}, Rotation: engine.IdentityQuaternion(), Weight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := w.AddBody(NewBodyDesc(Static, compound))
	hits, err := w.OverlapSphere(vec{}, 2, allQueryLayers)
	if err != nil || len(hits) != 2 || hits[0].Body != id || hits[0].Child != 0 || hits[1].Child != 1 {
		t.Fatal("child identities", hits, err)
	}
}

func TestOverlapShapeCoverageAndReservedBodies(t *testing.T) {
	for _, shape := range []Shape{Box(vec{X: 1, Y: 1, Z: 1}), testHull(t, vec{X: 1, Y: 1, Z: 1}), Plane(vec{Y: 1}, 0)} {
		w, _ := NewWorld(DefaultOptions())
		desc := NewBodyDesc(Static, shape)
		desc.Rotation = engine.QuaternionFromEuler(.3, .2, .1)
		desc.Position = vec{X: 5, Y: 2}
		w.AddBody(desc)
		hits, err := w.OverlapSphere(desc.Position, .2, allQueryLayers)
		if err != nil || len(hits) != 1 {
			t.Fatal("inside overlap", shape.Kind(), hits, err)
		}
	}
	w, _ := NewWorld(DefaultOptions())
	ray := Ray{Origin: vec{Z: -3}, Direction: vec{Z: 1}, MaxDistance: 10}
	if err := w.Advance(DefaultOptions().FixedStep, func(Tick) {
		if _, err := w.AddBody(NewBodyDesc(Static, Sphere(1))); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := w.Raycast(ray, allQueryLayers); err != nil || ok {
			t.Fatal("reserved body returned", ok, err)
		}
		if hits, err := w.OverlapSphere(vec{}, 1, allQueryLayers); err != nil || len(hits) != 0 {
			t.Fatal("reserved overlap", hits, err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := w.Advance(DefaultOptions().FixedStep, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := w.Raycast(ray, allQueryLayers); err != nil || !ok {
		t.Fatal("activated body absent", ok, err)
	}
}

func TestQueryIntermediateOverflow(t *testing.T) {
	hit, ok, err := RaycastShape(Ray{Origin: vec{X: math.MaxFloat64}, Direction: vec{Y: 1}, MaxDistance: 1}, Box(vec{X: 1, Y: 1, Z: 1}), vec{X: -math.MaxFloat64}, engine.IdentityQuaternion())
	if err != nil || ok {
		t.Fatal("overflow manufactured a hit", hit, ok, err)
	}
	w, _ := NewWorld(DefaultOptions())
	if _, err = w.OverlapSphere(vec{X: math.MaxFloat64}, math.MaxFloat64, allQueryLayers); err == nil {
		t.Fatal("overflowed bounds accepted")
	}
}
