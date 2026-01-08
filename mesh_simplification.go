package main

import (
	"container/heap"
	"math"
)

// Quadric represents a quadric error metric (4x4 symmetric matrix)
type Quadric struct {
	A [10]float64 // Symmetric matrix stored as: a11, a12, a13, a14, a22, a23, a24, a33, a34, a44
}

// NewQuadric creates a quadric from a plane equation
func NewQuadric(a, b, c, d float64) *Quadric {
	q := &Quadric{}
	// Q = [a b c d]^T * [a b c d]
	q.A[0] = a * a // a11
	q.A[1] = a * b // a12
	q.A[2] = a * c // a13
	q.A[3] = a * d // a14
	q.A[4] = b * b // a22
	q.A[5] = b * c // a23
	q.A[6] = b * d // a24
	q.A[7] = c * c // a33
	q.A[8] = c * d // a34
	q.A[9] = d * d // a44
	return q
}

// Add adds another quadric to this one
func (q *Quadric) Add(other *Quadric) *Quadric {
	result := &Quadric{}
	for i := 0; i < 10; i++ {
		result.A[i] = q.A[i] + other.A[i]
	}
	return result
}

// Error computes the error at a point
func (q *Quadric) Error(x, y, z float64) float64 {
	return q.A[0]*x*x + 2*q.A[1]*x*y + 2*q.A[2]*x*z + 2*q.A[3]*x +
		q.A[4]*y*y + 2*q.A[5]*y*z + 2*q.A[6]*y +
		q.A[7]*z*z + 2*q.A[8]*z +
		q.A[9]
}

// SimplificationVertex represents a vertex in the simplification mesh
type SimplificationVertex struct {
	Position Point
	Quadric  *Quadric
	ID       int
	Edges    []*SimplificationEdge
}

// SimplificationEdge represents an edge that can be collapsed
type SimplificationEdge struct {
	V0        *SimplificationVertex
	V1        *SimplificationVertex
	Cost      float64
	TargetPos Point
	Index     int // Index in heap
	Collapsed bool
}

// EdgeHeap implements heap.Interface for edge priority queue
type EdgeHeap []*SimplificationEdge

func (h EdgeHeap) Len() int           { return len(h) }
func (h EdgeHeap) Less(i, j int) bool { return h[i].Cost < h[j].Cost }
func (h EdgeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *EdgeHeap) Push(x interface{}) {
	edge := x.(*SimplificationEdge)
	edge.Index = len(*h)
	*h = append(*h, edge)
}

func (h *EdgeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	edge := old[n-1]
	old[n-1] = nil
	edge.Index = -1
	*h = old[0 : n-1]
	return edge
}

// SimplificationMesh represents a mesh being simplified
type SimplificationMesh struct {
	Vertices  []*SimplificationVertex
	Triangles [][3]int // Indices into Vertices
	Edges     EdgeHeap
}

// SimplifyMesh simplifies a mesh using quadric error metrics
func SimplifyMeshQEM(mesh *Mesh, targetTriangleCount int) *Mesh {
	// Build simplification mesh
	simpMesh := buildSimplificationMesh(mesh)

	if len(simpMesh.Triangles) <= targetTriangleCount {
		return mesh // Already simple enough
	}

	// Compute initial quadrics for all vertices
	simpMesh.computeQuadrics()

	// Compute costs for all edges
	simpMesh.computeEdgeCosts()

	// Build heap
	heap.Init(&simpMesh.Edges)

	// Collapse edges until we reach target
	targetEdgeCollapses := len(simpMesh.Triangles) - targetTriangleCount
	collapseCount := 0

	for collapseCount < targetEdgeCollapses && len(simpMesh.Edges) > 0 {
		// Get cheapest edge
		edge := heap.Pop(&simpMesh.Edges).(*SimplificationEdge)

		if edge.Collapsed {
			continue
		}

		// Collapse edge
		if simpMesh.collapseEdge(edge) {
			collapseCount++
		}
	}

	// Convert back to regular mesh
	return simpMesh.toMesh(mesh.Material)
}

// buildSimplificationMesh creates a simplification mesh from a regular mesh
func buildSimplificationMesh(mesh *Mesh) *SimplificationMesh {
	simpMesh := &SimplificationMesh{
		Vertices:  make([]*SimplificationVertex, 0),
		Triangles: make([][3]int, 0),
		Edges:     make(EdgeHeap, 0),
	}

	// Build vertex map
	vertexMap := make(map[Point]int)
	vertexID := 0

	// Helper to get or create vertex
	getVertex := func(p Point) int {
		// Round to reduce duplicates
		rounded := Point{
			X: math.Round(p.X*1000) / 1000,
			Y: math.Round(p.Y*1000) / 1000,
			Z: math.Round(p.Z*1000) / 1000,
		}

		if id, exists := vertexMap[rounded]; exists {
			return id
		}

		simpMesh.Vertices = append(simpMesh.Vertices, &SimplificationVertex{
			Position: p,
			Quadric:  &Quadric{},
			ID:       vertexID,
			Edges:    make([]*SimplificationEdge, 0),
		})
		vertexMap[rounded] = vertexID
		vertexID++
		return vertexID - 1
	}

	// Add all triangles
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				v0 := getVertex(mesh.Vertices[idx0])
				v1 := getVertex(mesh.Vertices[idx1])
				v2 := getVertex(mesh.Vertices[idx2])
				simpMesh.Triangles = append(simpMesh.Triangles, [3]int{v0, v1, v2})
			}
		}
	}

	// Build edge list
	edgeMap := make(map[[2]int]*SimplificationEdge)

	for _, tri := range simpMesh.Triangles {
		// Add three edges
		for i := 0; i < 3; i++ {
			v0 := tri[i]
			v1 := tri[(i+1)%3]

			// Order vertices
			if v0 > v1 {
				v0, v1 = v1, v0
			}

			key := [2]int{v0, v1}
			if _, exists := edgeMap[key]; !exists {
				edge := &SimplificationEdge{
					V0: simpMesh.Vertices[v0],
					V1: simpMesh.Vertices[v1],
				}
				edgeMap[key] = edge
				simpMesh.Vertices[v0].Edges = append(simpMesh.Vertices[v0].Edges, edge)
				simpMesh.Vertices[v1].Edges = append(simpMesh.Vertices[v1].Edges, edge)
			}
		}
	}

	// Convert edges to slice
	for _, edge := range edgeMap {
		simpMesh.Edges = append(simpMesh.Edges, edge)
	}

	return simpMesh
}

// computeQuadrics computes initial quadrics for all vertices
func (sm *SimplificationMesh) computeQuadrics() {
	// For each triangle, create a quadric and add to vertices
	for _, tri := range sm.Triangles {
		v0 := sm.Vertices[tri[0]].Position
		v1 := sm.Vertices[tri[1]].Position
		v2 := sm.Vertices[tri[2]].Position

		// Compute plane equation: ax + by + cz + d = 0
		// Normal = (v1-v0) × (v2-v0)
		e1x, e1y, e1z := v1.X-v0.X, v1.Y-v0.Y, v1.Z-v0.Z
		e2x, e2y, e2z := v2.X-v0.X, v2.Y-v0.Y, v2.Z-v0.Z

		nx, ny, nz := crossProduct(e1x, e1y, e1z, e2x, e2y, e2z)
		length := math.Sqrt(nx*nx + ny*ny + nz*nz)

		if length < 1e-10 {
			continue // Degenerate triangle
		}

		// Normalize
		a := nx / length
		b := ny / length
		c := nz / length
		d := -(a*v0.X + b*v0.Y + c*v0.Z)

		// Create quadric
		q := NewQuadric(a, b, c, d)

		// Add to all three vertices
		sm.Vertices[tri[0]].Quadric = sm.Vertices[tri[0]].Quadric.Add(q)
		sm.Vertices[tri[1]].Quadric = sm.Vertices[tri[1]].Quadric.Add(q)
		sm.Vertices[tri[2]].Quadric = sm.Vertices[tri[2]].Quadric.Add(q)
	}
}

// computeEdgeCosts computes collapse cost for all edges
func (sm *SimplificationMesh) computeEdgeCosts() {
	for _, edge := range sm.Edges {
		sm.computeEdgeCost(edge)
	}
}

// computeEdgeCost computes cost for a single edge
func (sm *SimplificationMesh) computeEdgeCost(edge *SimplificationEdge) {
	// Combined quadric
	q := edge.V0.Quadric.Add(edge.V1.Quadric)

	// Optimal position is midpoint (simplified)
	// For full QEM, solve for optimal position using matrix inversion
	edge.TargetPos = Point{
		X: (edge.V0.Position.X + edge.V1.Position.X) / 2,
		Y: (edge.V0.Position.Y + edge.V1.Position.Y) / 2,
		Z: (edge.V0.Position.Z + edge.V1.Position.Z) / 2,
	}

	// Compute error at target position
	edge.Cost = q.Error(edge.TargetPos.X, edge.TargetPos.Y, edge.TargetPos.Z)

	// Penalize boundary edges (simplified)
	if len(edge.V0.Edges) < 4 || len(edge.V1.Edges) < 4 {
		edge.Cost *= 1000.0
	}
}

// collapseEdge collapses an edge
func (sm *SimplificationMesh) collapseEdge(edge *SimplificationEdge) bool {
	if edge.Collapsed {
		return false
	}

	v0 := edge.V0
	v1 := edge.V1

	// Move v0 to target position
	v0.Position = edge.TargetPos
	v0.Quadric = v0.Quadric.Add(v1.Quadric)

	// Track which triangles to remove
	toRemove := make(map[int]bool)

	// Remove triangles using both v0 and v1 (degenerate after collapse)
	for i, tri := range sm.Triangles {
		hasV0 := tri[0] == v0.ID || tri[1] == v0.ID || tri[2] == v0.ID
		hasV1 := tri[0] == v1.ID || tri[1] == v1.ID || tri[2] == v1.ID

		if hasV0 && hasV1 {
			toRemove[i] = true
		}
	}

	// Create new triangle list without removed ones
	newTriangles := make([][3]int, 0, len(sm.Triangles)-len(toRemove))
	for i, tri := range sm.Triangles {
		if !toRemove[i] {
			// Replace v1 with v0
			for j := 0; j < 3; j++ {
				if tri[j] == v1.ID {
					tri[j] = v0.ID
				}
			}

			// Check for degenerate triangle (all same vertex)
			if tri[0] != tri[1] && tri[1] != tri[2] && tri[0] != tri[2] {
				newTriangles = append(newTriangles, tri)
			}
		}
	}
	sm.Triangles = newTriangles

	// Update edge list
	newEdges := EdgeHeap{}
	for _, e := range sm.Edges {
		if e == edge || e.Collapsed {
			continue
		}

		// Update edges connected to v1 to point to v0
		if e.V0 == v1 {
			e.V0 = v0
		}
		if e.V1 == v1 {
			e.V1 = v0
		}

		// Skip edges that became degenerate (both vertices same)
		if e.V0 != e.V1 {
			sm.computeEdgeCost(e)
			newEdges = append(newEdges, e)
		}
	}

	// Rebuild heap
	sm.Edges = newEdges
	heap.Init(&sm.Edges)

	edge.Collapsed = true
	return true
}

// toMesh converts back to regular mesh
func (sm *SimplificationMesh) toMesh(material Material) *Mesh {
	mesh := NewMesh()
	mesh.Material = material

	// Build vertex map
	vertexMap := make(map[int]int)
	for _, tri := range sm.Triangles {
		for _, vIdx := range tri {
			if _, exists := vertexMap[vIdx]; !exists {
				vertexMap[vIdx] = len(mesh.Vertices)
				mesh.Vertices = append(mesh.Vertices, sm.Vertices[vIdx].Position)
			}
		}
	}

	// Add indices
	for _, tri := range sm.Triangles {
		mesh.Indices = append(mesh.Indices, vertexMap[tri[0]], vertexMap[tri[1]], vertexMap[tri[2]])
	}

	return mesh
}

// SimplifyMeshClustering simplifies using vertex clustering
func SimplifyMeshClustering(mesh *Mesh, gridSize float64) *Mesh {
	// Compute bounds
	boundsVol := ComputeMeshBounds(mesh)
	bounds, ok := boundsVol.(*AABB)
	if !ok {
		return mesh
	}

	// Create grid
	clusters := make(map[[3]int][]Point)

	// Hash function for grid cells
	hashPoint := func(p Point) [3]int {
		return [3]int{
			int(math.Floor((p.X - bounds.Min.X) / gridSize)),
			int(math.Floor((p.Y - bounds.Min.Y) / gridSize)),
			int(math.Floor((p.Z - bounds.Min.Z) / gridSize)),
		}
	}

	// Assign vertices to clusters
	allPoints := make([]Point, 0)
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				allPoints = append(allPoints, mesh.Vertices[idx0], mesh.Vertices[idx1], mesh.Vertices[idx2])
			}
		}
	}

	for _, p := range allPoints {
		cell := hashPoint(p)
		clusters[cell] = append(clusters[cell], p)
	}

	// Compute cluster representatives (average position)
	representatives := make(map[[3]int]Point)
	for cell, points := range clusters {
		avg := Point{}
		for _, p := range points {
			avg.X += p.X
			avg.Y += p.Y
			avg.Z += p.Z
		}
		avg.X /= float64(len(points))
		avg.Y /= float64(len(points))
		avg.Z /= float64(len(points))
		representatives[cell] = avg
	}

	// Build simplified mesh
	simplified := NewMesh()
	simplified.Material = mesh.Material

	vertexMap := make(map[Point]int)

	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 < len(mesh.Indices) {
			idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
			if idx0 < len(mesh.Vertices) && idx1 < len(mesh.Vertices) && idx2 < len(mesh.Vertices) {
				v0 := representatives[hashPoint(mesh.Vertices[idx0])]
				v1 := representatives[hashPoint(mesh.Vertices[idx1])]
				v2 := representatives[hashPoint(mesh.Vertices[idx2])]

				// Skip degenerate triangles
				if pointsEqual(v0, v1) || pointsEqual(v1, v2) || pointsEqual(v2, v0) {
					continue
				}

				// Get or add vertices to the simplified mesh
				getOrAddVertex := func(v Point) int {
					if idx, exists := vertexMap[v]; exists {
						return idx
					}
					idx := len(simplified.Vertices)
					simplified.Vertices = append(simplified.Vertices, v)
					vertexMap[v] = idx
					return idx
				}

				i0 := getOrAddVertex(v0)
				i1 := getOrAddVertex(v1)
				i2 := getOrAddVertex(v2)
				simplified.Indices = append(simplified.Indices, i0, i1, i2)
			}
		}
	}

	return simplified
}

func pointsEqual(a, b Point) bool {
	const epsilon = 1e-6
	return math.Abs(a.X-b.X) < epsilon &&
		math.Abs(a.Y-b.Y) < epsilon &&
		math.Abs(a.Z-b.Z) < epsilon
}

// GenerateAdvancedLODChain generates LOD chain with proper simplification
func GenerateAdvancedLODChain(baseMesh *Mesh, numLevels int, useQEM bool) *LODGroup {
	lodGroup := NewLODGroup()

	// Add highest detail
	lodGroup.AddLOD(baseMesh, 50.0)

	// Generate progressively simpler LODs
	for i := 1; i < numLevels; i++ {
		targetRatio := 1.0 - (float64(i) / float64(numLevels))

		var simplifiedMesh *Mesh
		if useQEM {
			// Use quadric error metrics (slower but better quality)
			targetTris := int(float64(len(baseMesh.Indices)/3) * targetRatio)
			if targetTris < 4 {
				targetTris = 4
			}
			simplifiedMesh = SimplifyMeshQEM(baseMesh, targetTris)
		} else {
			// Use vertex clustering (faster but lower quality)
			boundsVol := ComputeMeshBounds(baseMesh)
			if aabb, ok := boundsVol.(*AABB); ok {
				size := aabb.GetSize()
				avgSize := (size.X + size.Y + size.Z) / 3.0
				gridSize := avgSize * (1.0 - targetRatio) * 0.5
				simplifiedMesh = SimplifyMeshClustering(baseMesh, gridSize)
			} else {
				simplifiedMesh = SimplifyMesh(baseMesh, targetRatio)
			}
		}

		distance := 50.0 * float64(i+1)
		lodGroup.AddLOD(simplifiedMesh, distance)
	}

	return lodGroup
}

// SimplifyMeshToRatio simplifies a mesh to a target triangle ratio
func SimplifyMeshToRatio(mesh *Mesh, ratio float64, useQEM bool) *Mesh {
	if ratio >= 1.0 {
		return mesh
	}

	if useQEM {
		targetTris := int(float64(len(mesh.Indices)/3) * ratio)
		if targetTris < 4 {
			targetTris = 4
		}
		return SimplifyMeshQEM(mesh, targetTris)
	}

	boundsVol := ComputeMeshBounds(mesh)
	if aabb, ok := boundsVol.(*AABB); ok {
		size := aabb.GetSize()
		avgSize := (size.X + size.Y + size.Z) / 3.0
		gridSize := avgSize * (1.0 - ratio) * 0.5
		return SimplifyMeshClustering(mesh, gridSize)
	}

	return SimplifyMesh(mesh, ratio)
}
