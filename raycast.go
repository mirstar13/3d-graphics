package main

import "math"

// Ray represents a ray in 3D space
type Ray struct {
	Origin    Point
	Direction Point // Must be normalized
}

// RayHit contains information about a ray intersection
type RayHit struct {
	Hit      bool
	Distance float64
	Point    Point
	Normal   Point
	Node     *SceneNode
	Triangle *Triangle
}

// NewRay creates a new ray
func NewRay(origin, direction Point) Ray {
	// Normalize direction
	dx, dy, dz := normalizeVector(direction.X, direction.Y, direction.Z)
	return Ray{
		Origin:    origin,
		Direction: Point{X: dx, Y: dy, Z: dz},
	}
}

// GetPoint returns a point along the ray at distance t
func (r *Ray) GetPoint(t float64) Point {
	return Point{
		X: r.Origin.X + r.Direction.X*t,
		Y: r.Origin.Y + r.Direction.Y*t,
		Z: r.Origin.Z + r.Direction.Z*t,
	}
}

// IntersectsTriangle performs ray-triangle intersection test
// Returns (hit, distance, barycentric coordinates u, v)
func (r *Ray) IntersectsTriangle(t *Triangle) (bool, float64, float64, float64) {
	const EPSILON = 0.0000001

	// Edge vectors
	edge1X := t.P1.X - t.P0.X
	edge1Y := t.P1.Y - t.P0.Y
	edge1Z := t.P1.Z - t.P0.Z

	edge2X := t.P2.X - t.P0.X
	edge2Y := t.P2.Y - t.P0.Y
	edge2Z := t.P2.Z - t.P0.Z

	// Begin calculating determinant
	hX, hY, hZ := crossProduct(r.Direction.X, r.Direction.Y, r.Direction.Z, edge2X, edge2Y, edge2Z)

	// Determinant
	det := edge1X*hX + edge1Y*hY + edge1Z*hZ

	// Use absolute value comparison
	if math.Abs(det) < EPSILON {
		return false, 0, 0, 0
	}

	invDet := 1.0 / det

	// Calculate distance from P0 to ray origin
	sX := r.Origin.X - t.P0.X
	sY := r.Origin.Y - t.P0.Y
	sZ := r.Origin.Z - t.P0.Z

	// Calculate u parameter
	u := invDet * (sX*hX + sY*hY + sZ*hZ)

	// Check bounds with epsilon
	if u < -EPSILON || u > 1.0+EPSILON {
		return false, 0, 0, 0
	}

	// Prepare to test v parameter
	qX, qY, qZ := crossProduct(sX, sY, sZ, edge1X, edge1Y, edge1Z)

	// Calculate v parameter
	v := invDet * (r.Direction.X*qX + r.Direction.Y*qY + r.Direction.Z*qZ)

	// Check bounds with epsilon
	if v < -EPSILON || u+v > 1.0+EPSILON {
		return false, 0, 0, 0
	}

	// Calculate t (distance along ray)
	distance := invDet * (edge2X*qX + edge2Y*qY + edge2Z*qZ)

	// Check if hit is in front of ray
	if distance > EPSILON {
		return true, distance, u, v
	}

	return false, 0, 0, 0
}

func (r *Ray) IntersectsTriangleWorld(t *Triangle, worldTransform Matrix4x4) (bool, float64, float64, float64) {
	// Transform triangle to world space
	p0 := worldTransform.TransformPoint(t.P0)
	p1 := worldTransform.TransformPoint(t.P1)
	p2 := worldTransform.TransformPoint(t.P2)

	worldTri := &Triangle{P0: p0, P1: p1, P2: p2}
	return r.IntersectsTriangle(worldTri)
}

// CameraScreenPointToRay converts a screen coordinate to a world-space ray
// screenX, screenY are in screen coordinates (pixels)
// Returns a ray in world space
func CameraScreenPointToRay(camera *Camera, screenX, screenY, screenWidth, screenHeight int) Ray {
	// Convert screen coordinates to normalized device coordinates [-1, 1]
	ndcX := (2.0*float64(screenX))/float64(screenWidth) - 1.0
	ndcY := 1.0 - (2.0*float64(screenY))/float64(screenHeight) // Flip Y

	// Convert to view space using inverse projection
	// For simple perspective projection: x_view = x_ndc * z / FOV
	// We'll use z=1 for the direction (point on near plane)

	viewX := ndcX / camera.FOV.X
	viewY := ndcY / camera.FOV.Y
	viewZ := 1.0 // Point on near plane

	// Normalize direction in view space
	viewDirX, viewDirY, viewDirZ := normalizeVector(viewX, viewY, viewZ)

	// Transform direction from view space to world space
	// View space direction needs to be rotated by camera's rotation
	worldDir := camera.Transform.TransformDirection(Point{X: viewDirX, Y: viewDirY, Z: viewDirZ})

	// Ray origin is camera position
	origin := camera.GetPosition()

	return NewRay(origin, worldDir)
}

// Raycast performs raycasting against the entire scene
// Returns the closest hit
func (s *Scene) Raycast(ray Ray, maxDistance float64) RayHit {
	closestHit := RayHit{Hit: false, Distance: maxDistance}

	// Test all renderable objects with proper transforms
	s.raycastNode(s.Root, ray, &closestHit)

	return closestHit
}

// raycastNode recursively tests a node and its children
func (s *Scene) raycastNode(node *SceneNode, ray Ray, closestHit *RayHit) {
	if !node.IsEnabled() {
		return
	}

	// Get world transform for this node
	worldMatrix := node.Transform.GetWorldMatrix()

	// Test bounding volume first for early rejection
	if node.Object != nil {
		bounds := s.computeNodeBounds(node)
		if bounds != nil {
			hit, _ := bounds.IntersectsRay(ray)
			if !hit {
				return // Skip this branch entirely
			}
		}

		// Now test the actual geometry
		s.raycastObject(node, ray, worldMatrix, closestHit)
	}

	// Test children
	for _, child := range node.Children {
		s.raycastNode(child, ray, closestHit)
	}
}

// raycastObject tests a ray against a drawable object
func (s *Scene) raycastObject(node *SceneNode, ray Ray, worldMatrix Matrix4x4, closestHit *RayHit) {
	switch obj := node.Object.(type) {
	case *Triangle:
		// Transform triangle to world space
		p0 := worldMatrix.TransformPoint(obj.P0)
		p1 := worldMatrix.TransformPoint(obj.P1)
		p2 := worldMatrix.TransformPoint(obj.P2)

		worldTri := &Triangle{P0: p0, P1: p1, P2: p2}
		hit, distance, _, _ := ray.IntersectsTriangle(worldTri)

		if hit && distance > 0 && distance < closestHit.Distance {
			hitPoint := ray.GetPoint(distance)

			var normal Point
			if obj.UseSetNormal && obj.Normal != nil {
				normal = worldMatrix.TransformDirection(*obj.Normal)
			} else {
				normal = CalculateSurfaceNormal(&p0, &p1, &p2, nil, false)
			}

			closestHit.Hit = true
			closestHit.Distance = distance
			closestHit.Point = hitPoint
			closestHit.Normal = normal
			closestHit.Node = node
			closestHit.Triangle = obj
		}

	case *Mesh:
		// Process each triangle in the mesh
		for i := 0; i < len(obj.Indices); i += 3 {
			if i+2 < len(obj.Indices) {
				idx0, idx1, idx2 := obj.Indices[i], obj.Indices[i+1], obj.Indices[i+2]
				if idx0 < len(obj.Vertices) && idx1 < len(obj.Vertices) && idx2 < len(obj.Vertices) {
					// Apply mesh position offset, then world transform
					v0 := obj.Vertices[idx0]
					localP0 := Point{
						X: v0.X + obj.Position.X,
						Y: v0.Y + obj.Position.Y,
						Z: v0.Z + obj.Position.Z,
					}
					p0 := worldMatrix.TransformPoint(localP0)

					v1 := obj.Vertices[idx1]
					localP1 := Point{
						X: v1.X + obj.Position.X,
						Y: v1.Y + obj.Position.Y,
						Z: v1.Z + obj.Position.Z,
					}
					p1 := worldMatrix.TransformPoint(localP1)

					v2 := obj.Vertices[idx2]
					localP2 := Point{
						X: v2.X + obj.Position.X,
						Y: v2.Y + obj.Position.Y,
						Z: v2.Z + obj.Position.Z,
					}
					p2 := worldMatrix.TransformPoint(localP2)

					tri := &Triangle{P0: p0, P1: p1, P2: p2}
					hit, distance, _, _ := ray.IntersectsTriangle(tri)

					if hit && distance > 0 && distance < closestHit.Distance {
						closestHit.Hit = true
						closestHit.Distance = distance
						closestHit.Point = ray.GetPoint(distance)
						closestHit.Normal = CalculateSurfaceNormal(&p0, &p1, &p2, nil, false)
						closestHit.Node = node
					}
				}
			}
		}

	case *LODGroup:
		// Raycast against current LOD mesh
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil {
			for i := 0; i < len(currentMesh.Indices); i += 3 {
				if i+2 < len(currentMesh.Indices) {
					idx0, idx1, idx2 := currentMesh.Indices[i], currentMesh.Indices[i+1], currentMesh.Indices[i+2]
					if idx0 < len(currentMesh.Vertices) && idx1 < len(currentMesh.Vertices) && idx2 < len(currentMesh.Vertices) {
						v0 := currentMesh.Vertices[idx0]
						localP0 := Point{
							X: v0.X + currentMesh.Position.X,
							Y: v0.Y + currentMesh.Position.Y,
							Z: v0.Z + currentMesh.Position.Z,
						}
						p0 := worldMatrix.TransformPoint(localP0)

						v1 := currentMesh.Vertices[idx1]
						localP1 := Point{
							X: v1.X + currentMesh.Position.X,
							Y: v1.Y + currentMesh.Position.Y,
							Z: v1.Z + currentMesh.Position.Z,
						}
						p1 := worldMatrix.TransformPoint(localP1)

						v2 := currentMesh.Vertices[idx2]
						localP2 := Point{
							X: v2.X + currentMesh.Position.X,
							Y: v2.Y + currentMesh.Position.Y,
							Z: v2.Z + currentMesh.Position.Z,
						}
						p2 := worldMatrix.TransformPoint(localP2)

						tri := &Triangle{P0: p0, P1: p1, P2: p2}
						hit, distance, _, _ := ray.IntersectsTriangle(tri)

						if hit && distance > 0 && distance < closestHit.Distance {
							closestHit.Hit = true
							closestHit.Distance = distance
							closestHit.Point = ray.GetPoint(distance)
							closestHit.Normal = CalculateSurfaceNormal(&p0, &p1, &p2, nil, false)
							closestHit.Node = node
						}
					}
				}
			}
		}
	}
}

// testTriangle tests a ray against a single triangle
func (s *Scene) testTriangle(tri *Triangle, ray Ray, node *SceneNode, closestHit *RayHit) {
	hit, distance, u, v := ray.IntersectsTriangle(tri)

	if hit && distance > 0 && distance < closestHit.Distance {
		// Calculate hit point
		hitPoint := ray.GetPoint(distance)

		// Calculate normal (use set normal if available, otherwise compute from geometry)
		var normal Point
		if tri.UseSetNormal && tri.Normal != nil {
			normal = *tri.Normal
		} else {
			normal = CalculateSurfaceNormal(&tri.P0, &tri.P1, &tri.P2, nil, false)
		}

		// Update closest hit
		closestHit.Hit = true
		closestHit.Distance = distance
		closestHit.Point = hitPoint
		closestHit.Normal = normal
		closestHit.Node = node
		closestHit.Triangle = tri

		// Store barycentric coords (could be useful for texture mapping later)
		_ = u
		_ = v
	}
}

// RaycastFromScreen performs raycasting from screen coordinates
func (s *Scene) RaycastFromScreen(screenX, screenY, screenWidth, screenHeight int, maxDistance float64) RayHit {
	ray := CameraScreenPointToRay(s.Camera, screenX, screenY, screenWidth, screenHeight)
	return s.Raycast(ray, maxDistance)
}

// RaycastAll returns all hits along a ray (not just closest)
func (s *Scene) RaycastAll(ray Ray, maxDistance float64) []RayHit {
	hits := make([]RayHit, 0)
	s.raycastNodeAll(s.Root, ray, maxDistance, &hits)
	return hits
}

func (s *Scene) raycastNodeAll(node *SceneNode, ray Ray, maxDistance float64, hits *[]RayHit) {
	if !node.IsEnabled() {
		return
	}

	if node.Object != nil {
		s.raycastObjectAll(node, ray, maxDistance, hits)
	}

	for _, child := range node.Children {
		s.raycastNodeAll(child, ray, maxDistance, hits)
	}
}

func (s *Scene) raycastObjectAll(node *SceneNode, ray Ray, maxDistance float64, hits *[]RayHit) {
	transformed := node.TransformSceneObject()
	if transformed == nil {
		return
	}

	switch obj := transformed.(type) {
	case *Triangle:
		hit, distance, _, _ := ray.IntersectsTriangle(obj)
		if hit && distance > 0 && distance < maxDistance {
			hitPoint := ray.GetPoint(distance)
			normal := CalculateSurfaceNormal(&obj.P0, &obj.P1, &obj.P2, obj.Normal, obj.UseSetNormal)

			*hits = append(*hits, RayHit{
				Hit:      true,
				Distance: distance,
				Point:    hitPoint,
				Normal:   normal,
				Node:     node,
				Triangle: obj,
			})
		}

	case *Mesh:
		for i := 0; i < len(obj.Indices); i += 3 {
			if i+2 < len(obj.Indices) {
				idx0, idx1, idx2 := obj.Indices[i], obj.Indices[i+1], obj.Indices[i+2]
				if idx0 < len(obj.Vertices) && idx1 < len(obj.Vertices) && idx2 < len(obj.Vertices) {
					tri := NewTriangle(obj.Vertices[idx0], obj.Vertices[idx1], obj.Vertices[idx2], 'o')
					tri.Material = obj.Material

					hit, distance, _, _ := ray.IntersectsTriangle(tri)
					if hit && distance > 0 && distance < maxDistance {
						hitPoint := ray.GetPoint(distance)
						normal := CalculateSurfaceNormal(&tri.P0, &tri.P1, &tri.P2, tri.Normal, tri.UseSetNormal)

						*hits = append(*hits, RayHit{
							Hit:      true,
							Distance: distance,
							Point:    hitPoint,
							Normal:   normal,
							Node:     node,
							Triangle: tri,
						})
					}
				}
			}
		}
	}
}

// LineOfSight checks if there's a clear line of sight between two points
func (s *Scene) LineOfSight(from, to Point, maxDistance float64) bool {
	// Create ray from 'from' to 'to'
	dirX := to.X - from.X
	dirY := to.Y - from.Y
	dirZ := to.Z - from.Z

	distance := math.Sqrt(dirX*dirX + dirY*dirY + dirZ*dirZ)
	if distance > maxDistance {
		return false
	}

	ray := NewRay(from, Point{X: dirX, Y: dirY, Z: dirZ})
	hit := s.Raycast(ray, distance)

	// If we hit something before reaching the target, no line of sight
	return !hit.Hit || hit.Distance >= distance-0.01
}
