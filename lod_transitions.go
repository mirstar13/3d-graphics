package main

import "math"

// LODTransitionMode specifies how LODs transition
type LODTransitionMode int

const (
	LODTransitionNone      LODTransitionMode = iota // Instant switch
	LODTransitionFade                               // Alpha blend between LODs
	LODTransitionMorph                              // Vertex morphing (geomorphing)
	LODTransitionCrossFade                          // Render both with fade
)

// LODTransitionState tracks the current transition
type LODTransitionState struct {
	FromLOD         int
	ToLOD           int
	Progress        float64 // 0.0 to 1.0
	IsTransitioning bool
	Mode            LODTransitionMode
	Duration        float64 // Transition duration in seconds
}

// NewLODTransitionState creates a new transition state
func NewLODTransitionState(mode LODTransitionMode, duration float64) *LODTransitionState {
	return &LODTransitionState{
		Mode:     mode,
		Duration: duration,
	}
}

// UpdateTransition updates the transition progress
func (ts *LODTransitionState) UpdateTransition(dt float64) {
	if !ts.IsTransitioning {
		return
	}

	ts.Progress += dt / ts.Duration
	if ts.Progress >= 1.0 {
		ts.Progress = 1.0
		ts.IsTransitioning = false
		ts.FromLOD = ts.ToLOD
	}
}

// StartTransition begins a new transition
func (ts *LODTransitionState) StartTransition(fromLOD, toLOD int) {
	if fromLOD == toLOD {
		return
	}

	ts.FromLOD = fromLOD
	ts.ToLOD = toLOD
	ts.Progress = 0.0
	ts.IsTransitioning = true
}

// GetAlpha returns the blend factor for transitions (0.0 = from, 1.0 = to)
func (ts *LODTransitionState) GetAlpha() float64 {
	if !ts.IsTransitioning {
		return 1.0
	}

	// Smooth transition using ease-in-out
	t := ts.Progress
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - math.Pow(-2*t+2, 2)/2
}

// LODGroupWithTransitions extends LODGroup with smooth transitions
type LODGroupWithTransitions struct {
	*LODGroup
	TransitionState *LODTransitionState
	LastDistance    float64
	LastCameraPos   Point
}

// NewLODGroupWithTransitions creates a new LOD group with transitions
func NewLODGroupWithTransitions(mode LODTransitionMode, duration float64) *LODGroupWithTransitions {
	return &LODGroupWithTransitions{
		LODGroup:        NewLODGroup(),
		TransitionState: NewLODTransitionState(mode, duration),
	}
}

// UpdateWithTransition updates LOD selection with smooth transitions
func (lg *LODGroupWithTransitions) UpdateWithTransition(worldPos Point, camera *Camera, dt float64) {
	lg.TransitionState.UpdateTransition(dt)

	camPos := camera.GetPosition()
	dx := worldPos.X - camPos.X
	dy := worldPos.Y - camPos.Y
	dz := worldPos.Z - camPos.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	newLOD := lg.SelectLOD(worldPos, camera)

	if !lg.TransitionState.IsTransitioning && newLOD != lg.CurrentLOD {
		distanceChange := math.Abs(distance - lg.LastDistance)

		if distanceChange > lg.UpdateHysteresis {
			lg.TransitionState.StartTransition(lg.CurrentLOD, newLOD)
		}
	}

	if !lg.TransitionState.IsTransitioning {
		lg.CurrentLOD = newLOD
	}

	lg.LastDistance = distance
	lg.LastCameraPos = camPos
}

// GetBlendedMesh returns the mesh(es) to render with alpha values
func (lg *LODGroupWithTransitions) GetBlendedMesh() (fromMesh, toMesh *Mesh, alpha float64) {
	if !lg.TransitionState.IsTransitioning {
		return lg.GetCurrentMesh(), nil, 1.0
	}

	fromLOD := lg.TransitionState.FromLOD
	toLOD := lg.TransitionState.ToLOD

	if fromLOD < 0 || fromLOD >= len(lg.Levels) {
		return lg.GetCurrentMesh(), nil, 1.0
	}
	if toLOD < 0 || toLOD >= len(lg.Levels) {
		return lg.GetCurrentMesh(), nil, 1.0
	}

	alpha = lg.TransitionState.GetAlpha()
	return lg.Levels[fromLOD].Mesh, lg.Levels[toLOD].Mesh, alpha
}

// MorphedMesh represents a mesh with interpolated vertices
type MorphedMesh struct {
	Vertices  []Point
	Triangles []*Triangle
}

// CreateMorphedMesh creates a morphed mesh between two LODs
func CreateMorphedMesh(fromMesh, toMesh *Mesh, t float64) *MorphedMesh {
	morphed := &MorphedMesh{
		Triangles: make([]*Triangle, 0),
	}

	if len(fromMesh.Triangles) != len(toMesh.Triangles) {
		morphed.Triangles = fromMesh.Triangles
		return morphed
	}

	for i := 0; i < len(fromMesh.Triangles); i++ {
		fromTri := fromMesh.Triangles[i]
		toTri := toMesh.Triangles[i]

		p0 := lerpPoint(fromTri.P0, toTri.P0, t)
		p1 := lerpPoint(fromTri.P1, toTri.P1, t)
		p2 := lerpPoint(fromTri.P2, toTri.P2, t)

		tri := NewTriangle(p0, p1, p2, fromTri.char)
		tri.Material = fromTri.Material

		if fromTri.UseSetNormal && toTri.UseSetNormal {
			normal := lerpPoint(*fromTri.Normal, *toTri.Normal, t)
			tri.SetNormal(normal)
		}

		morphed.Triangles = append(morphed.Triangles, tri)
	}

	return morphed
}

// lerpPoint linearly interpolates between two points
func lerpPoint(from, to Point, t float64) Point {
	return Point{
		X: from.X + (to.X-from.X)*t,
		Y: from.Y + (to.Y-from.Y)*t,
		Z: from.Z + (to.Z-from.Z)*t,
	}
}

// UpdateLODsWithTransitions updates all LOD groups with transitions
func (s *Scene) UpdateLODsWithTransitions(dt float64) {
	lodNodes := s.FindNodesByTag("lod-transition-enabled")

	for _, node := range lodNodes {
		if lodGroup, ok := node.Object.(*LODGroupWithTransitions); ok {
			worldPos := node.Transform.GetWorldPosition()
			lodGroup.UpdateWithTransition(worldPos, s.Camera, dt)
		}
	}
}

// SetLODGroupWithTransition sets a LOD group with transitions on a node
func (sn *SceneNode) SetLODGroupWithTransition(lodGroup *LODGroupWithTransitions) {
	sn.AddTag("lod-transition-enabled")
	sn.Object = lodGroup
}

// DitherPattern represents a dithering pattern for LOD transitions
type DitherPattern struct {
	Pattern [16][16]bool
}

// NewDitherPattern creates a Bayer dithering pattern
func NewDitherPattern() *DitherPattern {
	dp := &DitherPattern{}

	bayer4x4 := [4][4]int{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			value := bayer4x4[y%4][x%4]
			dp.Pattern[y][x] = value < 8
		}
	}

	return dp
}

// ShouldRenderPixel determines if a pixel should be rendered based on dither
func (dp *DitherPattern) ShouldRenderPixel(x, y int, alpha float64) bool {
	threshold := dp.Pattern[y%16][x%16]
	return alpha > 0.5 == threshold
}

// ComputeLODScreenCoverage computes approximate screen coverage
func ComputeLODScreenCoverage(worldPos Point, radius float64, camera *Camera) float64 {
	camPos := camera.GetPosition()
	dx := worldPos.X - camPos.X
	dy := worldPos.Y - camPos.Y
	dz := worldPos.Z - camPos.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	if distance < 0.001 {
		return 1.0
	}

	projectedSize := (radius * camera.FOV.X) / distance
	coverage := projectedSize / camera.FOV.X

	if coverage > 1.0 {
		coverage = 1.0
	}
	if coverage < 0.0 {
		coverage = 0.0
	}

	return coverage
}

// SmoothStep provides smooth interpolation
func SmoothStep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0.0, 1.0)
	return t * t * (3.0 - 2.0*t)
}

// SmootherStep provides even smoother interpolation (Ken Perlin)
func SmootherStep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0.0, 1.0)
	return t * t * t * (t*(t*6.0-15.0) + 10.0)
}
