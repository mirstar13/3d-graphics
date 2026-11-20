package engine

import (
	"math"
)

// LODLevel represents a single level of detail
type LODLevel struct {
	Mesh           *Mesh
	MaxDistance    float64
	ScreenCoverage float64
}

// LODGroup manages multiple LOD levels for an object
type LODGroup struct {
	Levels           []LODLevel
	CurrentLOD       int
	UseScreenSpace   bool
	FadeTransition   bool
	TransitionRange  float64
	BoundingVolume   BoundingVolume
	LastUpdatePos    Point
	LastUpdateLOD    int
	UpdateHysteresis float64
}

// NewLODGroup creates a new LOD group
func NewLODGroup() *LODGroup {
	return &LODGroup{
		Levels:           make([]LODLevel, 0),
		CurrentLOD:       0,
		UseScreenSpace:   false,
		FadeTransition:   false,
		TransitionRange:  0.1,
		UpdateHysteresis: 5.0,
	}
}

// AddLOD adds a level of detail
func (lg *LODGroup) AddLOD(mesh *Mesh, maxDistance float64) {
	level := LODLevel{
		Mesh:        mesh,
		MaxDistance: maxDistance,
	}
	lg.Levels = append(lg.Levels, level)
	lg.sortLODs()

	if len(lg.Levels) > 0 && lg.BoundingVolume == nil {
		lg.BoundingVolume = ComputeMeshBounds(lg.Levels[0].Mesh)
	}
}

// AddLODWithScreenCoverage adds a LOD level with screen space coverage
func (lg *LODGroup) AddLODWithScreenCoverage(mesh *Mesh, screenCoverage float64) {
	level := LODLevel{
		Mesh:           mesh,
		ScreenCoverage: screenCoverage,
	}
	lg.Levels = append(lg.Levels, level)
	lg.UseScreenSpace = true
}

// sortLODs sorts LOD levels by distance
func (lg *LODGroup) sortLODs() {
	n := len(lg.Levels)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if lg.Levels[j].MaxDistance > lg.Levels[j+1].MaxDistance {
				lg.Levels[j], lg.Levels[j+1] = lg.Levels[j+1], lg.Levels[j]
			}
		}
	}
}

// SelectLOD selects the appropriate LOD level based on camera distance
func (lg *LODGroup) SelectLOD(worldPos Point, camera *Camera) int {
	if len(lg.Levels) == 0 {
		return -1
	}

	if lg.UseScreenSpace {
		return lg.selectLODScreenSpace(worldPos, camera)
	}

	return lg.selectLODDistance(worldPos, camera)
}

// selectLODDistance selects LOD based on distance
func (lg *LODGroup) selectLODDistance(worldPos Point, camera *Camera) int {
	camPos := camera.GetPosition()

	dx := worldPos.X - camPos.X
	dy := worldPos.Y - camPos.Y
	dz := worldPos.Z - camPos.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	if lg.CurrentLOD >= 0 && lg.CurrentLOD < len(lg.Levels) {
		currentMaxDist := lg.Levels[lg.CurrentLOD].MaxDistance
		if math.Abs(distance-currentMaxDist) < lg.UpdateHysteresis {
			return lg.CurrentLOD
		}
	}

	selectedLOD := len(lg.Levels) - 1

	for i, level := range lg.Levels {
		if distance <= level.MaxDistance {
			selectedLOD = i
			break
		}
	}

	return selectedLOD
}

// selectLODScreenSpace selects LOD based on screen space coverage
func (lg *LODGroup) selectLODScreenSpace(worldPos Point, camera *Camera) int {
	coverage := lg.calculateScreenCoverage(worldPos, camera)
	selectedLOD := len(lg.Levels) - 1

	for i, level := range lg.Levels {
		if coverage >= level.ScreenCoverage {
			selectedLOD = i
			break
		}
	}

	return selectedLOD
}

// calculateScreenCoverage estimates screen space coverage
func (lg *LODGroup) calculateScreenCoverage(worldPos Point, camera *Camera) float64 {
	if lg.BoundingVolume == nil {
		return 0.0
	}

	return ComputeLODScreenCoverage(worldPos, lg.BoundingVolume.GetRadius(), camera)
}

// Update updates the LOD selection
func (lg *LODGroup) Update(worldPos Point, camera *Camera) {
	lg.LastUpdatePos = worldPos
	newLOD := lg.SelectLOD(worldPos, camera)

	// Clamp LOD index to valid range
	if newLOD < 0 {
		newLOD = 0
	}
	if newLOD >= len(lg.Levels) {
		newLOD = len(lg.Levels) - 1
	}

	lg.CurrentLOD = newLOD
	lg.LastUpdateLOD = lg.CurrentLOD
}

// GetCurrentMesh returns the currently active LOD mesh
func (lg *LODGroup) GetCurrentMesh() *Mesh {
	if lg.CurrentLOD < 0 || lg.CurrentLOD >= len(lg.Levels) {
		// Return first level if index invalid
		if len(lg.Levels) > 0 {
			return lg.Levels[0].Mesh
		}
		return nil
	}
	return lg.Levels[lg.CurrentLOD].Mesh
}

// GetLODLevel returns the LOD level at index
func (lg *LODGroup) GetLODLevel(index int) *LODLevel {
	if index < 0 || index >= len(lg.Levels) {
		return nil
	}
	return &lg.Levels[index]
}

// GetLODCount returns the number of LOD levels
func (lg *LODGroup) GetLODCount() int {
	return len(lg.Levels)
}

// SetLODGroup sets a LOD group on a node
func (sn *SceneNode) SetLODGroup(lodGroup *LODGroup) {
	sn.AddTag("lod-enabled")
	sn.Object = lodGroup
}

// GetLODGroup retrieves LOD group if node has one
func (sn *SceneNode) GetLODGroup() *LODGroup {
	if lodGroup, ok := sn.Object.(*LODGroup); ok {
		return lodGroup
	}
	return nil
}

// UpdateLODs updates all LOD groups in the scene
func (s *Scene) UpdateLODs() {
	lodNodes := s.FindNodesByTag("lod-enabled")

	for _, node := range lodNodes {
		if lodGroup, ok := node.Object.(*LODGroup); ok {
			worldPos := node.Transform.GetWorldPosition()
			lodGroup.Update(worldPos, s.Camera)
		}
	}
}

// SimplifyMesh retains a deterministic subset of complete indexed triangles.
// It preserves vertex attributes and remaps shared indices. A nonpositive or
// NaN ratio produces an empty mesh; ratios >= 1 return the input mesh.
func SimplifyMesh(mesh *Mesh, targetRatio float64) *Mesh {
	if targetRatio >= 1 {
		return mesh
	}
	result := NewMesh()
	result.Position, result.Material = mesh.Position, mesh.Material
	if targetRatio <= 0 || math.IsNaN(targetRatio) {
		return result
	}
	triangleCount := len(mesh.Indices) / 3
	targetCount := int(math.Ceil(float64(triangleCount) * targetRatio))
	remapped := make(map[int]int)
	for sample := 0; sample < targetCount; sample++ {
		first := (sample * triangleCount / targetCount) * 3
		indices := mesh.Indices[first : first+3]
		valid := true
		for _, index := range indices {
			if index < 0 || index >= len(mesh.Vertices) {
				valid = false
			}
		}
		if !valid {
			continue
		}
		if len(mesh.FaceMaterials) > 0 {
			result.FaceMaterials = append(result.FaceMaterials, mesh.MaterialForFace(first/3))
		}
		for _, index := range indices {
			newIndex, exists := remapped[index]
			if !exists {
				newIndex = len(result.Vertices)
				remapped[index] = newIndex
				result.Vertices = append(result.Vertices, mesh.Vertices[index])
				if len(mesh.UVs) == len(mesh.Vertices) {
					result.UVs = append(result.UVs, mesh.UVs[index])
				}
			}
			result.Indices = append(result.Indices, newIndex)
		}
	}
	return result
}

// GenerateLODChain generates multiple LOD levels from a base mesh
func GenerateLODChain(baseMesh *Mesh, numLevels int) *LODGroup {
	lodGroup := NewLODGroup()
	lodGroup.AddLOD(baseMesh, 50.0)

	for i := 1; i < numLevels; i++ {
		ratio := 1.0 - (float64(i) / float64(numLevels))
		simplifiedMesh := SimplifyMesh(baseMesh, ratio)
		distance := 50.0 * float64(i+1)
		lodGroup.AddLOD(simplifiedMesh, distance)
	}

	return lodGroup
}

// GetLODStats returns statistics about LOD usage
type LODStats struct {
	TotalLODGroups int
	ActiveLOD0     int
	ActiveLOD1     int
	ActiveLOD2     int
	ActiveLODOther int
	TotalTriangles int
}

func (s *Scene) GetLODStats() LODStats {
	stats := LODStats{}
	lodNodes := s.FindNodesByTag("lod-enabled")

	stats.TotalLODGroups = len(lodNodes)

	for _, node := range lodNodes {
		if lodGroup, ok := node.Object.(*LODGroup); ok {
			switch lodGroup.CurrentLOD {
			case 0:
				stats.ActiveLOD0++
			case 1:
				stats.ActiveLOD1++
			case 2:
				stats.ActiveLOD2++
			default:
				stats.ActiveLODOther++
			}

			currentMesh := lodGroup.GetCurrentMesh()
			if currentMesh != nil {
				stats.TotalTriangles += len(currentMesh.Indices) / 3
			}
		}
	}

	return stats
}
