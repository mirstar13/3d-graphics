package main

type Mesh struct {
	Triangles []*Triangle
	Quads     []*Quad
	Position  Point
}

func NewMesh() *Mesh {
	return &Mesh{
		Triangles: make([]*Triangle, 0),
		Quads:     make([]*Quad, 0),
		Position:  *NewPoint(0, 0, 0),
	}
}

func (m *Mesh) SetPosition(x, y, z float64) {
	m.Position = *NewPoint(x, y, z)
}

func (m *Mesh) AddTriangle(t *Triangle) {
	m.Triangles = append(m.Triangles, t)
}

func (m *Mesh) AddQuad(q *Quad) {
	m.Quads = append(m.Quads, q)
}

// translatePoint adds the mesh position to a point
func (m *Mesh) translatePoint(p Point) Point {
	return Point{
		X: p.X + m.Position.X,
		Y: p.Y + m.Position.Y,
		Z: p.Z + m.Position.Z,
	}
}

// translateTriangle creates a copy of a triangle moved to mesh position
func (m *Mesh) translateTriangle(t *Triangle) *Triangle {
	newT := *t // Shallow copy
	newT.P0 = m.translatePoint(t.P0)
	newT.P1 = m.translatePoint(t.P1)
	newT.P2 = m.translatePoint(t.P2)
	// Normal doesn't change with translation
	return &newT
}

// translateQuad creates a copy of a quad moved to mesh position
func (m *Mesh) translateQuad(q *Quad) *Quad {
	newQ := *q // Shallow copy
	newQ.P0 = m.translatePoint(q.P0)
	newQ.P1 = m.translatePoint(q.P1)
	newQ.P2 = m.translatePoint(q.P2)
	newQ.P3 = m.translatePoint(q.P3)
	return &newQ
}

func (m *Mesh) Draw(renderer *Renderer, camera *Camera) {
	for _, q := range m.Quads {
		m.translateQuad(q).Draw(renderer, camera)
	}
	for _, t := range m.Triangles {
		m.translateTriangle(t).Draw(renderer, camera)
	}
}

func (m *Mesh) DrawFilled(renderer *Renderer, camera *Camera) {
	for _, q := range m.Quads {
		m.translateQuad(q).DrawFilled(renderer, camera)
	}
	for _, t := range m.Triangles {
		m.translateTriangle(t).DrawFilled(renderer, camera)
	}
}

func (m *Mesh) Project(renderer *Renderer, camera *Camera) {
	for _, q := range m.Quads {
		m.translateQuad(q).Project(renderer, camera)
	}
	for _, t := range m.Triangles {
		m.translateTriangle(t).Project(renderer, camera)
	}
}

func (m *Mesh) RotateGlobal(axis byte, angle float64) {
	// Rotate the geometry itself
	for _, q := range m.Quads {
		q.RotateGlobal(axis, angle)
	}
	for _, t := range m.Triangles {
		t.RotateGlobal(axis, angle)
	}
	// Also rotate the mesh position if you want it to orbit the world origin
	// m.Position.RotateGlobal(axis, angle)
}

func (m *Mesh) RotateLocal(axis byte, angle float64) {
	// Rotate geometry around the mesh's local origin (0,0,0 relative to Position)
	for _, q := range m.Quads {
		q.RotateGlobal(axis, angle) // Since quads are defined locally, Global rotation works as local here
	}
	for _, t := range m.Triangles {
		t.RotateGlobal(axis, angle)
	}
}
