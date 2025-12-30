package main

// Quad represents a quadrilateral (4-sided polygon) in 3D space
// NOTE: Quads are converted to triangles for rendering
type Quad struct {
	P0           Point
	P1           Point
	P2           Point
	P3           Point
	Material     Material
	Normal       *Point
	UseSetNormal bool
}

func NewQuad(p0, p1, p2, p3 Point) *Quad {
	return &Quad{
		P0:           p0,
		P1:           p1,
		P2:           p2,
		P3:           p3,
		Material:     NewMaterial(),
		Normal:       nil,
		UseSetNormal: false,
	}
}

func (q *Quad) SetMaterial(material Material) *Quad {
	q.Material = material
	return q
}

func (q *Quad) SetColor(color Color) *Quad {
	q.Material.DiffuseColor = color
	return q
}

func (q *Quad) SetNormal(normal Point) *Quad {
	q.Normal = &normal
	q.UseSetNormal = true
	return q
}

func (q *Quad) Draw(renderer *Renderer, camera *Camera) {
	NewLine(q.P0, q.P1).Draw(renderer, camera)
	NewLine(q.P1, q.P2).Draw(renderer, camera)
	NewLine(q.P2, q.P3).Draw(renderer, camera)
	NewLine(q.P3, q.P0).Draw(renderer, camera)
}

func (q *Quad) Project(renderer *Renderer, camera *Camera) {
	NewLine(q.P0, q.P1).Project(renderer, camera)
	NewLine(q.P1, q.P2).Project(renderer, camera)
	NewLine(q.P2, q.P3).Project(renderer, camera)
	NewLine(q.P3, q.P0).Project(renderer, camera)
}

func (q *Quad) RotateGlobal(axis byte, angle float64) {
	q.P0.RotateGlobal(axis, angle)
	q.P1.RotateGlobal(axis, angle)
	q.P2.RotateGlobal(axis, angle)
	q.P3.RotateGlobal(axis, angle)

	if q.UseSetNormal && q.Normal != nil {
		q.Normal.Rotate(axis, angle)
	}
}

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

	q.P0.RotateGlobal(axis, angle)
	q.P1.RotateGlobal(axis, angle)
	q.P2.RotateGlobal(axis, angle)
	q.P3.RotateGlobal(axis, angle)

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

// DrawFilled renders the quad by converting to triangles
func (q *Quad) DrawFilled(renderer *Renderer, camera *Camera) {
	// Convert to triangles and render them
	triangles := ConvertQuadToTriangles(q)
	for _, tri := range triangles {
		tri.DrawFilled(renderer, camera)
	}
}
