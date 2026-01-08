package main

import "math"

// InstanceData holds per-instance data for instanced rendering
type InstanceData struct {
	Transform Matrix4x4
	Color     Color
	LODLevel  int
	UserData  interface{} // For custom per-instance data
}

// InstancedMesh represents a mesh with multiple instances
type InstancedMesh struct {
	BaseMesh  *Mesh
	Instances []InstanceData
	Material  Material
	Enabled   bool
}

// NewInstancedMesh creates a new instanced mesh
func NewInstancedMesh(mesh *Mesh) *InstancedMesh {
	return &InstancedMesh{
		BaseMesh:  mesh,
		Instances: make([]InstanceData, 0),
		Enabled:   true,
	}
}

// AddInstance adds an instance with a transform
func (im *InstancedMesh) AddInstance(transform Matrix4x4, color Color) {
	im.Instances = append(im.Instances, InstanceData{
		Transform: transform,
		Color:     color,
		LODLevel:  0,
	})
}

// AddInstanceAt adds an instance at a specific position
func (im *InstancedMesh) AddInstanceAt(x, y, z float64, color Color) {
	transform := IdentityMatrix()
	transform.M[3] = x  // Translation X
	transform.M[7] = y  // Translation Y
	transform.M[11] = z // Translation Z
	im.AddInstance(transform, color)
}

// ClearInstances removes all instances
func (im *InstancedMesh) ClearInstances() {
	im.Instances = im.Instances[:0]
}

// GetInstanceCount returns the number of instances
func (im *InstancedMesh) GetInstanceCount() int {
	return len(im.Instances)
}

// SetInstanceTransform updates an instance's transform
func (im *InstancedMesh) SetInstanceTransform(index int, transform Matrix4x4) {
	if index >= 0 && index < len(im.Instances) {
		im.Instances[index].Transform = transform
	}
}

// SetInstanceColor updates an instance's color
func (im *InstancedMesh) SetInstanceColor(index int, color Color) {
	if index >= 0 && index < len(im.Instances) {
		im.Instances[index].Color = color
	}
}

// InstanceBatch groups instances for efficient rendering
type InstanceBatch struct {
	Mesh      *Mesh
	Material  Material
	Instances []InstanceData
}

// InstanceManager manages instanced rendering
type InstanceManager struct {
	batches map[*Mesh]*InstanceBatch
}

// NewInstanceManager creates a new instance manager
func NewInstanceManager() *InstanceManager {
	return &InstanceManager{
		batches: make(map[*Mesh]*InstanceBatch),
	}
}

// AddInstance adds an instance to be rendered
func (im *InstanceManager) AddInstance(mesh *Mesh, material Material, transform Matrix4x4, color Color) {
	batch, exists := im.batches[mesh]
	if !exists {
		batch = &InstanceBatch{
			Mesh:      mesh,
			Material:  material,
			Instances: make([]InstanceData, 0),
		}
		im.batches[mesh] = batch
	}

	batch.Instances = append(batch.Instances, InstanceData{
		Transform: transform,
		Color:     color,
	})
}

// GetBatches returns all batches for rendering
func (im *InstanceManager) GetBatches() []*InstanceBatch {
	batches := make([]*InstanceBatch, 0, len(im.batches))
	for _, batch := range im.batches {
		batches = append(batches, batch)
	}
	return batches
}

// Clear removes all instances
func (im *InstanceManager) Clear() {
	im.batches = make(map[*Mesh]*InstanceBatch)
}

// GetStats returns statistics
func (im *InstanceManager) GetStats() InstanceStats {
	totalInstances := 0
	for _, batch := range im.batches {
		totalInstances += len(batch.Instances)
	}
	return InstanceStats{
		BatchCount:     len(im.batches),
		InstanceCount:  totalInstances,
		DrawCallsSaved: maxInt(totalInstances-len(im.batches), 0),
	}
}

// InstanceStats holds instancing statistics
type InstanceStats struct {
	BatchCount     int
	InstanceCount  int
	DrawCallsSaved int
}

// Helper functions for instancing

// CreateInstanceGrid creates a grid of instances
func CreateInstanceGrid(mesh *Mesh, gridSize int, spacing float64, baseColor Color) *InstancedMesh {
	im := NewInstancedMesh(mesh)

	halfGrid := float64(gridSize) / 2.0
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			px := (float64(x) - halfGrid) * spacing
			pz := (float64(z) - halfGrid) * spacing

			// Vary color slightly
			hue := float64(x*gridSize+z) / float64(gridSize*gridSize)
			color := ColorFromHSV(hue*360.0, 0.7, 0.9)

			im.AddInstanceAt(px, 0, pz, color)
		}
	}

	return im
}

// CreateInstanceCircle creates instances in a circle
func CreateInstanceCircle(mesh *Mesh, count int, radius float64, baseColor Color) *InstancedMesh {
	im := NewInstancedMesh(mesh)

	for i := 0; i < count; i++ {
		angle := (float64(i) / float64(count)) * 6.28318 // 2*PI
		x := radius * cosine(angle)
		z := radius * sine(angle)

		// Rotate instance to face center
		transform := IdentityMatrix()
		transform.M[3] = x  // Translation X
		transform.M[11] = z // Translation Z

		color := ColorFromHSV(float64(i)/float64(count)*360.0, 0.8, 0.9)
		im.AddInstance(transform, color)
	}

	return im
}

// ColorFromHSV creates a color from HSV values
func ColorFromHSV(h, s, v float64) Color {
	// Normalize h to [0, 360]
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}

	c := v * s
	x := c * (1.0 - math.Abs(modFloat(h/60.0, 2.0)-1.0))
	m := v - c

	var r, g, b float64

	if h < 60 {
		r, g, b = c, x, 0
	} else if h < 120 {
		r, g, b = x, c, 0
	} else if h < 180 {
		r, g, b = 0, c, x
	} else if h < 240 {
		r, g, b = 0, x, c
	} else if h < 300 {
		r, g, b = x, 0, c
	} else {
		r, g, b = c, 0, x
	}

	return Color{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
	}
}

// absFloat is an alias to math.Abs for consistency
func absFloatLocal(x float64) float64 {
	return math.Abs(x)
}

func modFloat(x, y float64) float64 {
	return x - float64(int(x/y))*y
}

func cosine(angle float64) float64 {
	return math.Cos(angle)
}

func sine(angle float64) float64 {
	return math.Sin(angle)
}
