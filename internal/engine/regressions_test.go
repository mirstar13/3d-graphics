package engine

import (
	"math"
	"testing"
)

func TestProjectionFrustumAndRayAgree(t *testing.T) {
	camera := NewCameraAt(0, 0, 0)
	camera.SetFOV(90, 90)
	point := Point{X: 5, Y: 0, Z: 10}
	x, y, _ := camera.ProjectPoint(point, 80, 100)
	if x != 75 || y != 40 {
		t.Errorf("90 degree projection = (%d,%d), want (75,40)", x, y)
	}
	narrow := BuildFrustum(camera)
	if !narrow.TestPoint(point) {
		t.Error("visible point rejected by frustum")
	}
	camera.SetFOV(60, 60)
	outside := Point{X: 8, Z: 10}
	f := BuildFrustumSimple(camera)
	if f.TestPoint(outside) {
		t.Error("point beyond 30 degree half-angle accepted")
	}
	camera.SetFOV(90, 90)
	for _, pixel := range [][2]int{{0, 0}, {25, 30}, {99, 79}} {
		ray := CameraScreenPointToRay(camera, pixel[0], pixel[1], 100, 80)
		p := Point{X: ray.Origin.X + ray.Direction.X*20, Y: ray.Origin.Y + ray.Direction.Y*20, Z: ray.Origin.Z + ray.Direction.Z*20}
		x, y, _ := camera.ProjectPoint(p, 80, 100)
		if x != pixel[0] || y != pixel[1] {
			t.Errorf("ray round trip %v -> (%d,%d)", pixel, x, y)
		}
	}
}

func TestParentTransformInvalidatesChild(t *testing.T) {
	parent, child := NewTransform(), NewTransformAt(1, 0, 0)
	child.SetParent(parent)
	child.GetWorldMatrix()
	child.GetInverseMatrix()
	parent.SetPosition(5, 0, 0)
	if got := child.GetWorldPosition().X; got != 6 {
		t.Errorf("child X=%v, want 6", got)
	}
	if got := child.InverseTransformPoint(Point{X: 6}).X; math.Abs(got) > 1e-9 {
		t.Errorf("inverse X=%v, want 0", got)
	}
	local := NewTransform()
	local.GetLocalMatrix()
	local.SetPosition(3, 0, 0)
	if got := local.GetWorldPosition().X; got != 3 {
		t.Errorf("local cache X=%v, want 3", got)
	}
}

func TestSimplifyMeshKeepsValidTriangles(t *testing.T) {
	mesh := NewMesh()
	mesh.Position = Point{X: 4}
	mesh.Vertices = []Point{{X: 0}, {X: 1}, {X: 2}, {X: 3}}
	mesh.UVs = []TextureCoord{{U: 0}, {U: .3}, {U: .6}, {U: 1}}
	mesh.Indices = []int{0, 1, 2, 0, 2, 3}
	for _, ratio := range []float64{0, .25, .5, .9, 1} {
		simplified := SimplifyMesh(mesh, ratio)
		if simplified.Position != mesh.Position {
			t.Error("mesh offset lost")
		}
		for _, idx := range simplified.Indices {
			if idx < 0 || idx >= len(simplified.Vertices) {
				t.Fatalf("ratio %v has invalid index %d in %d vertices", ratio, idx, len(simplified.Vertices))
			}
		}
		if len(simplified.UVs) != len(simplified.Vertices) {
			t.Error("vertex UVs lost")
		}
	}
}

func TestCullingRetainsVisibleChildOfCulledObject(t *testing.T) {
	scene := NewScene()
	scene.Camera = NewCameraAt(0, 0, 0)
	parent := NewSceneNodeWithObject("parent", NewTriangle(Point{X: 100, Z: 10}, Point{X: 101, Z: 10}, Point{X: 100, Y: 1, Z: 10}, 'x'))
	child := NewSceneNodeWithObject("child", NewTriangle(Point{Z: 10}, Point{X: 1, Z: 10}, Point{Y: 1, Z: 10}, 'x'))
	parent.AddChild(child)
	scene.AddNode(parent)
	frustum := BuildFrustumSimple(scene.Camera)
	var visible []*SceneNode
	FrustumCullNode(scene.Root, &frustum, &visible)
	for _, node := range visible {
		if node == child {
			return
		}
	}
	t.Fatal("culled parent's bounds hid a visible child")
}

func TestClippingPreservesTextureAttributes(t *testing.T) {
	camera := NewCameraAt(0, 0, 0)
	camera.Near = 1
	triangle := NewTriangle(Point{Z: 0}, Point{X: 2, Z: 2}, Point{Y: 2, Z: 2}, 'x')
	triangle.SetUVs(TextureCoord{}, TextureCoord{U: 1}, TextureCoord{V: 1})
	clipped := ClipTriangleToNearPlane(triangle, camera)
	if len(clipped) != 2 {
		t.Fatalf("got %d clipped triangles", len(clipped))
	}
	for _, tri := range clipped {
		if !tri.HasUVs || tri.Material != triangle.Material {
			t.Fatal("clipping lost attributes")
		}
		for index, point := range []Point{tri.P0, tri.P1, tri.P2} {
			uv := []TextureCoord{tri.UV0, tri.UV1, tri.UV2}[index]
			if math.Abs(uv.U-point.X/2) > 1e-9 || math.Abs(uv.V-point.Y/2) > 1e-9 {
				t.Fatalf("incorrect intersection UV %+v at %+v", uv, point)
			}
		}
	}
}

func TestPooledTriangleAndCopiesClearStaleAttributes(t *testing.T) {
	triangle := AcquireTriangle()
	triangle.HasUVs = true
	triangle.UseSetNormal = true
	material := NewMaterial()
	triangle.Material = &material
	ReleaseTriangle(triangle)
	next := AcquireTriangle()
	defer ReleaseTriangle(next)
	if next.HasUVs || next.UseSetNormal || next.Material != nil || next.Normal != nil {
		t.Fatal("pooled triangle retained stale attributes")
	}
	source := NewTriangle(Point{}, Point{}, Point{}, 'x')
	source.SetUVs(TextureCoord{U: .5}, TextureCoord{}, TextureCoord{})
	CopyTriangle(next, source)
	if !next.HasUVs || next.UV0.U != .5 {
		t.Fatal("copy lost UVs")
	}
	source.HasUVs = false
	CopyTriangle(next, source)
	if next.HasUVs {
		t.Fatal("copy retained previous UV flag")
	}
}

func TestCameraLookAtAndRotationAxes(t *testing.T) {
	for _, target := range []Point{{Z: 1}, {X: 1}, {Y: 1}, {X: 2, Y: 1, Z: -3}} {
		transform := NewTransform()
		transform.LookAt(target)
		forward := transform.GetForwardVector()
		x, y, z := NormalizeVector(target.X, target.Y, target.Z)
		if math.Abs(forward.X-x) > 1e-9 || math.Abs(forward.Y-y) > 1e-9 || math.Abs(forward.Z-z) > 1e-9 {
			t.Errorf("LookAt %+v faces %+v", target, forward)
		}
	}
	yaw := QuaternionFromEuler(0, math.Pi/2, 0).RotateVector(Point{Z: 1})
	if math.Abs(yaw.X-1) > 1e-9 || math.Abs(yaw.Z) > 1e-9 {
		t.Errorf("yaw rotates wrong axis: %+v", yaw)
	}
	pitch := QuaternionFromEuler(math.Pi/2, 0, 0).RotateVector(Point{Y: 1})
	if math.Abs(pitch.Z-1) > 1e-9 {
		t.Errorf("pitch rotates wrong axis: %+v", pitch)
	}
}

func TestCompositionScalesLocalAxes(t *testing.T) {
	rotation := QuaternionFromAxisAngle(Point{Z: 1}, math.Pi/2)
	matrix := ComposeMatrix(Point{}, rotation, Point{X: 2, Y: 3, Z: 4})
	got := matrix.TransformPoint(Point{X: 1})
	if math.Abs(got.X) > 1e-9 || math.Abs(got.Y-2) > 1e-9 {
		t.Fatalf("local X scale after rotation = %+v", got)
	}
}

func TestFittedBoundsEncloseUnevenPointCloud(t *testing.T) {
	points := []Point{{X: 0}, {X: 1}, {X: 10}}
	bounds := ComputeOptimalOBB(points)
	for _, point := range points {
		if !bounds.Contains(point) {
			t.Errorf("fitted box excludes %+v", point)
		}
	}
}
