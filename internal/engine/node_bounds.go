package engine

// ComputeNodeBounds returns world-space bounds for a node's own geometry.
// Unknown or empty geometry returns nil and must be treated as unbounded
// when culling. Descendants have independent bounds.
func ComputeNodeBounds(node *SceneNode) *AABB {
	matrix := node.Transform.GetWorldMatrix()
	var points []Point
	var offset Point
	switch object := node.Object.(type) {
	case *Mesh:
		points, offset = object.Vertices, object.Position
	case *LODGroupWithTransitions:
		from, to, _ := object.GetBlendedMesh()
		for _, mesh := range []*Mesh{from, to} {
			if mesh != nil {
				for _, p := range mesh.Vertices {
					points = append(points, Point{X: p.X + mesh.Position.X, Y: p.Y + mesh.Position.Y, Z: p.Z + mesh.Position.Z})
				}
			}
		}
	case *LODGroup:
		if mesh := object.GetCurrentMesh(); mesh != nil {
			points, offset = mesh.Vertices, mesh.Position
		}
	case *Triangle:
		points = []Point{object.P0, object.P1, object.P2}
	case *Quad:
		points = []Point{object.P0, object.P1, object.P2, object.P3}
	case *Line:
		points = []Point{object.Start, object.End}
	case *Point:
		points = []Point{*object}
	case *Circle:
		points = object.Points
	}
	if len(points) == 0 {
		return nil
	}
	world := make([]Point, len(points))
	for index, point := range points {
		point.X += offset.X
		point.Y += offset.Y
		point.Z += offset.Z
		world[index] = matrix.TransformPoint(point)
	}
	return NewAABBFromPoints(world)
}
