package main

import "math"

// ============================================================================
// PURE GEOMETRY DATA STRUCTURES (NO RENDERING LOGIC)
// ============================================================================

// Point represents a 3D point
type Point struct {
	X, Y, Z float64
	char    byte // Legacy field, unused in new architecturec
}

// NewPoint creates a new point
func NewPoint(x, y, z float64) *Point {
	return &Point{X: x, Y: y, Z: z, char: 'o'}
}

// SetChar is kept for compatibility but unused
func (p *Point) SetChar(char byte) *Point {
	p.char = char
	return p
}

// Rotate rotates a point around an axis (legacy helper)
func (p *Point) Rotate(axis byte, angle float64) {
	c := math.Cos(angle)
	s := math.Sin(angle)

	switch axis {
	case 'x':
		y, z := p.Y, p.Z
		p.Y = y*c - z*s
		p.Z = y*s + z*c
	case 'y':
		x, z := p.X, p.Z
		p.X = x*c - z*s
		p.Z = x*s + z*c
	case 'z':
		x, y := p.X, p.Y
		p.X = x*c - y*s
		p.Y = x*s + y*c
	}
}

// ============================================================================
// LINE
// ============================================================================

// Line represents a line segment in 3D space
type Line struct {
	Start Point
	End   Point
}

// NewLine creates a new line
func NewLine(start, end Point) *Line {
	return &Line{Start: start, End: end}
}

// ============================================================================
// TRIANGLE
// ============================================================================

// Triangle represents a 3D triangle (pure data)
type Triangle struct {
	P0           Point
	P1           Point
	P2           Point
	Normal       *Point
	Material     IMaterial
	char         byte
	UseSetNormal bool
}

// NewTriangle creates a new triangle
func NewTriangle(p0, p1, p2 Point, char byte) *Triangle {
	mat := NewMaterial()
	return &Triangle{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		char:         char,
		Material:     &mat,
		Normal:       nil,
		UseSetNormal: false,
	}
}

// SetMaterial sets the material
func (t *Triangle) SetMaterial(material IMaterial) *Triangle {
	t.Material = material
	return t
}

// SetColor sets the diffuse color
func (t *Triangle) SetColor(color Color) *Triangle {
	if mat, ok := t.Material.(*Material); ok {
		mat.DiffuseColor = color
	}
	return t
}

// SetNormal sets an explicit normal
func (t *Triangle) SetNormal(normal Point) *Triangle {
	t.Normal = &normal
	t.UseSetNormal = true
	return t
}

// RotateLocal rotates triangle around its center
func (t *Triangle) RotateLocal(axis byte, angle float64) {
	centerX := (t.P0.X + t.P1.X + t.P2.X) / 3.0
	centerY := (t.P0.Y + t.P1.Y + t.P2.Y) / 3.0
	centerZ := (t.P0.Z + t.P1.Z + t.P2.Z) / 3.0

	t.P0.X -= centerX
	t.P0.Y -= centerY
	t.P0.Z -= centerZ

	t.P1.X -= centerX
	t.P1.Y -= centerY
	t.P1.Z -= centerZ

	t.P2.X -= centerX
	t.P2.Y -= centerY
	t.P2.Z -= centerZ

	t.P0.Rotate(axis, angle)
	t.P1.Rotate(axis, angle)
	t.P2.Rotate(axis, angle)

	t.P0.X += centerX
	t.P0.Y += centerY
	t.P0.Z += centerZ

	t.P1.X += centerX
	t.P1.Y += centerY
	t.P1.Z += centerZ

	t.P2.X += centerX
	t.P2.Y += centerY
	t.P2.Z += centerZ

	if t.UseSetNormal && t.Normal != nil {
		t.Normal.Rotate(axis, angle)
	}
}

// RotateGlobal rotates triangle around world origin
func (t *Triangle) RotateGlobal(axis byte, angle float64) {
	t.P0.Rotate(axis, angle)
	t.P1.Rotate(axis, angle)
	t.P2.Rotate(axis, angle)

	if t.UseSetNormal && t.Normal != nil {
		t.Normal.Rotate(axis, angle)
	}
}

// ============================================================================
// QUAD
// ============================================================================

// Quad represents a quadrilateral (pure data)
type Quad struct {
	P0           Point
	P1           Point
	P2           Point
	P3           Point
	Material     IMaterial
	Normal       *Point
	UseSetNormal bool
}

// NewQuad creates a new quad
func NewQuad(p0, p1, p2, p3 Point) *Quad {
	mat := NewMaterial()
	return &Quad{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		P3:           p3,
		Material:     &mat,
		Normal:       nil,
		UseSetNormal: false,
	}
}

// SetMaterial sets the material
func (q *Quad) SetMaterial(material IMaterial) *Quad {
	q.Material = material
	return q
}

// SetColor sets the diffuse color
func (q *Quad) SetColor(color Color) *Quad {
	if mat, ok := q.Material.(*Material); ok {
		mat.DiffuseColor = color
	}
	return q
}

// SetNormal sets an explicit normal
func (q *Quad) SetNormal(normal Point) *Quad {
	q.Normal = &normal
	q.UseSetNormal = true
	return q
}

// RotateGlobal rotates quad around world origin
func (q *Quad) RotateGlobal(axis byte, angle float64) {
	q.P0.Rotate(axis, angle)
	q.P1.Rotate(axis, angle)
	q.P2.Rotate(axis, angle)
	q.P3.Rotate(axis, angle)

	if q.UseSetNormal && q.Normal != nil {
		q.Normal.Rotate(axis, angle)
	}
}

// RotateLocal rotates quad around its center
func (q *Quad) RotateLocal(axis byte, angle float64) {
	centerX := (q.P0.X + q.P1.X + q.P2.X + q.P3.X) / 4.0
	centerY := (q.P0.Y + q.P1.Y + q.P2.Y + q.P3.Y) / 4.0
	centerZ := (q.P0.Z + q.P1.Z + q.P2.Z + q.P3.Z) / 4.0

	q.P0.X -= centerX
	q.P0.Y -= centerY
	q.P0.Z -= centerZ

	q.P1.X -= centerX
	q.P1.Y -= centerY
	q.P1.Z -= centerZ

	q.P2.X -= centerX
	q.P2.Y -= centerY
	q.P2.Z -= centerZ

	q.P3.X -= centerX
	q.P3.Y -= centerY
	q.P3.Z -= centerZ

	q.P0.Rotate(axis, angle)
	q.P1.Rotate(axis, angle)
	q.P2.Rotate(axis, angle)
	q.P3.Rotate(axis, angle)

	q.P0.X += centerX
	q.P0.Y += centerY
	q.P0.Z += centerZ

	q.P1.X += centerX
	q.P1.Y += centerY
	q.P1.Z += centerZ

	q.P2.X += centerX
	q.P2.Y += centerY
	q.P2.Z += centerZ

	q.P3.X += centerX
	q.P3.Y += centerY
	q.P3.Z += centerZ

	if q.UseSetNormal && q.Normal != nil {
		q.Normal.Rotate(axis, angle)
	}
}

// ============================================================================
// CIRCLE
// ============================================================================

// Circle represents a 3D circle (pure data)
type Circle struct {
	Center Point
	Radius float64
	Points []Point
}

// NewCircle creates a circle from a center, radius, and number of segments
func NewCircle(x, y, z, r float64, segments int) *Circle {
	if segments < 3 {
		segments = 3
	}

	points := make([]Point, segments)
	step := 2 * math.Pi / float64(segments)

	for i := 0; i < segments; i++ {
		theta := float64(i) * step
		px := x + r*math.Cos(theta)
		py := y + r*math.Sin(theta)
		pz := z
		points[i] = *NewPoint(px, py, pz)
	}

	return &Circle{
		Center: Point{X: x, Y: y, Z: z},
		Radius: r,
		Points: points,
	}
}

// RotateGlobal rotates circle around world origin
func (c *Circle) RotateGlobal(axis byte, angle float64) {
	for i := range c.Points {
		c.Points[i].Rotate(axis, angle)
	}
	c.Center.Rotate(axis, angle)
}

// RotateLocal rotates circle around its center
func (c *Circle) RotateLocal(axis byte, angle float64) {
	for i := range c.Points {
		c.Points[i].X -= c.Center.X
		c.Points[i].Y -= c.Center.Y
		c.Points[i].Z -= c.Center.Z

		c.Points[i].Rotate(axis, angle)

		c.Points[i].X += c.Center.X
		c.Points[i].Y += c.Center.Y
		c.Points[i].Z += c.Center.Z
	}
}

// ============================================================================
// MESH
// ============================================================================

// Mesh represents a collection of indexed vertices
type Mesh struct {
	Vertices []Point
	UVs      []TextureCoord // UV coordinates per vertex
	Normals  []Point        // Normals per vertex
	Indices  []int
	Position Point
	Material IMaterial // Added to store material for the whole mesh
}

// NewMesh creates a new mesh
func NewMesh() *Mesh {
	mat := NewMaterial()
	return &Mesh{
		Vertices: make([]Point, 0),
		UVs:      make([]TextureCoord, 0),
		Normals:  make([]Point, 0),
		Indices:  make([]int, 0),
		Position: *NewPoint(0, 0, 0),
		Material: &mat,
	}
}

// SetPosition sets the mesh position
func (m *Mesh) SetPosition(x, y, z float64) {
	m.Position = *NewPoint(x, y, z)
}

// AddVertex adds a raw vertex to the mesh and returns its index
func (m *Mesh) AddVertex(x, y, z float64) int {
	m.Vertices = append(m.Vertices, Point{X: x, Y: y, Z: z})
	return len(m.Vertices) - 1
}

// AddUV adds a UV coordinate to the mesh
func (m *Mesh) AddUV(u, v float64) {
	m.UVs = append(m.UVs, TextureCoord{U: u, V: v})
}

// AddNormal adds a normal vector to the mesh
func (m *Mesh) AddNormal(x, y, z float64) {
	m.Normals = append(m.Normals, Point{X: x, Y: y, Z: z})
}

// AddVertexWithUV adds a vertex with UV coordinate
func (m *Mesh) AddVertexWithUV(x, y, z, u, v float64) int {
	m.Vertices = append(m.Vertices, Point{X: x, Y: y, Z: z})
	m.UVs = append(m.UVs, TextureCoord{U: u, V: v})
	return len(m.Vertices) - 1
}

// AddIndex adds a single index to the mesh
func (m *Mesh) AddIndex(i int) {
	m.Indices = append(m.Indices, i)
}

// AddTriangleIndices adds 3 indices to form a triangle
func (m *Mesh) AddTriangleIndices(i1, i2, i3 int) {
	m.Indices = append(m.Indices, i1, i2, i3)
}

// AddQuadIndices adds two triangles (6 indices) to form a quad
func (m *Mesh) AddQuadIndices(i1, i2, i3, i4 int) {
	// Triangle 1
	m.Indices = append(m.Indices, i1, i2, i3)
	// Triangle 2
	m.Indices = append(m.Indices, i1, i3, i4)
}

// RotateGlobal rotates all geometry around world origin
func (m *Mesh) RotateGlobal(axis byte, angle float64) {
	for i := range m.Vertices {
		m.Vertices[i].Rotate(axis, angle)
	}
}

// CalculateNormals computes smooth normals for each vertex in the mesh
// by averaging the normals of the faces connected to each vertex.
func (m *Mesh) CalculateNormals() {
	// Initialize normals slice with zero vectors, one for each vertex
	m.Normals = make([]Point, len(m.Vertices))
	for i := range m.Vertices {
		m.Normals[i] = Point{X: 0, Y: 0, Z: 0}
	}

	// Iterate over each face (triangle) in the mesh
	for i := 0; i < len(m.Indices); i += 3 {
		// Get the indices of the vertices that form the triangle
		i0 := m.Indices[i]
		i1 := m.Indices[i+1]
		i2 := m.Indices[i+2]

		// Bounds check for vertex indices to prevent panics on malformed mesh data
		if i0 < 0 || i0 >= len(m.Vertices) || i1 < 0 || i1 >= len(m.Vertices) || i2 < 0 || i2 >= len(m.Vertices) {
			continue
		}

		// Get the vertex positions
		v0 := m.Vertices[i0]
		v1 := m.Vertices[i1]
		v2 := m.Vertices[i2]

		// Calculate the face normal using the cross product of two edges
		// Edge 1: v1 - v0
		e1 := Point{X: v1.X - v0.X, Y: v1.Y - v0.Y, Z: v1.Z - v0.Z}
		// Edge 2: v2 - v0
		e2 := Point{X: v2.X - v0.X, Y: v2.Y - v0.Y, Z: v2.Z - v0.Z}
		// Face normal: e1 x e2
		faceNormal := Point{
			X: e1.Y*e2.Z - e1.Z*e2.Y,
			Y: e1.Z*e2.X - e1.X*e2.Z,
			Z: e1.X*e2.Y - e1.Y*e2.X,
		}

		// Add the face normal to the normals of each vertex of the face
		m.Normals[i0].X += faceNormal.X
		m.Normals[i0].Y += faceNormal.Y
		m.Normals[i0].Z += faceNormal.Z

		m.Normals[i1].X += faceNormal.X
		m.Normals[i1].Y += faceNormal.Y
		m.Normals[i1].Z += faceNormal.Z

		m.Normals[i2].X += faceNormal.X
		m.Normals[i2].Y += faceNormal.Y
		m.Normals[i2].Z += faceNormal.Z
	}

	// Normalize all the vertex normals
	for i := range m.Normals {
		normal := &m.Normals[i]
		length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
		if length > 0.0001 {
			normal.X /= length
			normal.Y /= length
			normal.Z /= length
		}
	}
}

// RotateLocal rotates all geometry around local origin
func (m *Mesh) RotateLocal(axis byte, angle float64) {
	// For indexed meshes, local rotation is just rotating the vertex offsets
	for i := range m.Vertices {
		m.Vertices[i].Rotate(axis, angle)
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// ConvertQuadToTriangles converts a quad into two triangles
func ConvertQuadToTriangles(q *Quad) []*Triangle {
	t1 := NewTriangle(q.P0, q.P1, q.P2, 'x')
	t1.Material = q.Material
	if q.UseSetNormal {
		t1.SetNormal(*q.Normal)
	}

	t2 := NewTriangle(q.P0, q.P2, q.P3, 'x')
	t2.Material = q.Material
	if q.UseSetNormal {
		t2.SetNormal(*q.Normal)
	}

	return []*Triangle{t1, t2}
}
