package main

import "math"

// FrustumPlane represents a plane in the view frustum
type FrustumPlane struct {
	Normal   Point
	Distance float64
}

// ViewFrustum represents the camera's view frustum with 6 planes
type ViewFrustum struct {
	Planes [6]FrustumPlane // Left, Right, Top, Bottom, Near, Far
}

const (
	FrustumLeft = iota
	FrustumRight
	FrustumTop
	FrustumBottom
	FrustumNear
	FrustumFar
)

// Degrees to Radians conversion factor
const DegToRad = math.Pi / 180.0

// BuildFrustum constructs view frustum from camera parameters
func BuildFrustum(camera *Camera) ViewFrustum {
	frustum := ViewFrustum{}

	// Get camera transform data
	pos := camera.GetPosition()

	forward := camera.GetForwardVectorPoint()
	right := camera.GetRightVectorPoint()
	up := camera.GetUpVectorPoint()

	// Calculate frustum dimensions at near and far planes
	// Convert FOV degrees to radians before Tan
	nearHeight := 2.0 * math.Tan((camera.FOV.Y/2.0)*DegToRad) * camera.Near
	nearWidth := 2.0 * math.Tan((camera.FOV.X/2.0)*DegToRad) * camera.Near

	// Near and far plane centers
	nearCenter := Point{
		X: pos.X + forward.X*camera.Near,
		Y: pos.Y + forward.Y*camera.Near,
		Z: pos.Z + forward.Z*camera.Near,
	}

	farCenter := Point{
		X: pos.X + forward.X*camera.Far,
		Y: pos.Y + forward.Y*camera.Far,
		Z: pos.Z + forward.Z*camera.Far,
	}

	// Near plane
	frustum.Planes[FrustumNear].Normal = forward
	frustum.Planes[FrustumNear].Distance = -dotProduct(forward.X, forward.Y, forward.Z, nearCenter.X, nearCenter.Y, nearCenter.Z)

	// Far plane
	frustum.Planes[FrustumFar].Normal = Point{X: -forward.X, Y: -forward.Y, Z: -forward.Z}
	frustum.Planes[FrustumFar].Distance = -dotProduct(-forward.X, -forward.Y, -forward.Z, farCenter.X, farCenter.Y, farCenter.Z)

	// Calculate plane normals for left, right, top, bottom
	// Left plane
	leftNormal := Point{
		X: forward.X - right.X*nearWidth/(2.0*camera.Near),
		Y: forward.Y - right.Y*nearWidth/(2.0*camera.Near),
		Z: forward.Z - right.Z*nearWidth/(2.0*camera.Near),
	}
	leftNormal.X, leftNormal.Y, leftNormal.Z = normalizeVector(leftNormal.X, leftNormal.Y, leftNormal.Z)
	frustum.Planes[FrustumLeft].Normal = leftNormal
	frustum.Planes[FrustumLeft].Distance = -dotProduct(leftNormal.X, leftNormal.Y, leftNormal.Z, pos.X, pos.Y, pos.Z)

	// Right plane
	rightNormal := Point{
		X: forward.X + right.X*nearWidth/(2.0*camera.Near),
		Y: forward.Y + right.Y*nearWidth/(2.0*camera.Near),
		Z: forward.Z + right.Z*nearWidth/(2.0*camera.Near),
	}
	rightNormal.X, rightNormal.Y, rightNormal.Z = normalizeVector(rightNormal.X, rightNormal.Y, rightNormal.Z)
	frustum.Planes[FrustumRight].Normal = Point{X: -rightNormal.X, Y: -rightNormal.Y, Z: -rightNormal.Z}
	frustum.Planes[FrustumRight].Distance = -dotProduct(-rightNormal.X, -rightNormal.Y, -rightNormal.Z, pos.X, pos.Y, pos.Z)

	// Top plane
	topNormal := Point{
		X: forward.X + up.X*nearHeight/(2.0*camera.Near),
		Y: forward.Y + up.Y*nearHeight/(2.0*camera.Near),
		Z: forward.Z + up.Z*nearHeight/(2.0*camera.Near),
	}
	topNormal.X, topNormal.Y, topNormal.Z = normalizeVector(topNormal.X, topNormal.Y, topNormal.Z)
	frustum.Planes[FrustumTop].Normal = Point{X: -topNormal.X, Y: -topNormal.Y, Z: -topNormal.Z}
	frustum.Planes[FrustumTop].Distance = -dotProduct(-topNormal.X, -topNormal.Y, -topNormal.Z, pos.X, pos.Y, pos.Z)

	// Bottom plane
	bottomNormal := Point{
		X: forward.X - up.X*nearHeight/(2.0*camera.Near),
		Y: forward.Y - up.Y*nearHeight/(2.0*camera.Near),
		Z: forward.Z - up.Z*nearHeight/(2.0*camera.Near),
	}
	bottomNormal.X, bottomNormal.Y, bottomNormal.Z = normalizeVector(bottomNormal.X, bottomNormal.Y, bottomNormal.Z)
	frustum.Planes[FrustumBottom].Normal = bottomNormal
	frustum.Planes[FrustumBottom].Distance = -dotProduct(bottomNormal.X, bottomNormal.Y, bottomNormal.Z, pos.X, pos.Y, pos.Z)

	return frustum
}

// TestPoint tests if a point is inside the frustum
func (f *ViewFrustum) TestPoint(p Point) bool {
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		distance := dotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, p.X, p.Y, p.Z) + plane.Distance

		if distance < 0 {
			return false // Outside this plane
		}
	}
	return true
}

// TestSphere tests if a sphere intersects the frustum
func (f *ViewFrustum) TestSphere(center Point, radius float64) bool {
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		distance := dotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, center.X, center.Y, center.Z) + plane.Distance

		if distance < -radius {
			return false // Completely outside this plane
		}
	}
	return true
}

// TestAABB tests if an AABB intersects the frustum
func (f *ViewFrustum) TestAABB(aabb *AABB) bool {
	// Get AABB corners
	corners := [8]Point{
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Min.Z},
		{X: aabb.Min.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Min.Y, Z: aabb.Max.Z},
		{X: aabb.Min.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
		{X: aabb.Max.X, Y: aabb.Max.Y, Z: aabb.Max.Z},
	}

	// Test each plane
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		insideCount := 0

		// Check if any corner is inside this plane
		for _, corner := range corners {
			distance := dotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, corner.X, corner.Y, corner.Z) + plane.Distance
			if distance >= 0 {
				insideCount++
			}
		}

		// If all corners are outside this plane, AABB is outside frustum
		if insideCount == 0 {
			return false
		}
	}

	return true
}

// TestOBB tests if an OBB intersects the frustum
func (f *ViewFrustum) TestOBB(obb *OBB) bool {
	corners := obb.GetCorners()

	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		insideCount := 0

		for _, corner := range corners {
			distance := dotProduct(plane.Normal.X, plane.Normal.Y, plane.Normal.Z, corner.X, corner.Y, corner.Z) + plane.Distance
			if distance >= 0 {
				insideCount++
			}
		}

		if insideCount == 0 {
			return false
		}
	}

	return true
}

// FrustumCullNode recursively culls scene nodes against frustum
func FrustumCullNode(node *SceneNode, frustum *ViewFrustum, visible *[]*SceneNode) {
	if !node.IsEnabled() {
		return
	}

	// Test node's bounding volume
	if node.Object != nil {
		bounds := ComputeNodeBounds(node)
		if bounds != nil && !frustum.TestAABB(bounds) {
			return // Culled
		}
		*visible = append(*visible, node)
	}

	// Recurse to children
	for _, child := range node.Children {
		FrustumCullNode(child, frustum, visible)
	}
}

// ComputeNodeBounds computes AABB for a scene node
func ComputeNodeBounds(node *SceneNode) *AABB {
	worldTransform := node.GetWorldTransform()

	switch obj := node.Object.(type) {
	case *Mesh:
		localBounds := ComputeMeshBounds(obj)
		return TransformAABB(localBounds, worldTransform)

	case *LODGroup:
		if obj.BoundingVolume != nil {
			if aabb, ok := obj.BoundingVolume.(*AABB); ok {
				return TransformAABB(aabb, worldTransform)
			}
		}
		// Compute from current mesh
		mesh := obj.GetCurrentMesh()
		if mesh != nil {
			localBounds := ComputeMeshBounds(mesh)
			return TransformAABB(localBounds, worldTransform)
		}

	case *Triangle:
		points := []Point{obj.P0, obj.P1, obj.P2}
		transformedPoints := make([]Point, 3)
		for i, p := range points {
			transformedPoints[i] = worldTransform.TransformPoint(p)
		}
		return NewAABBFromPoints(transformedPoints)

	case *Quad:
		points := []Point{obj.P0, obj.P1, obj.P2, obj.P3}
		transformedPoints := make([]Point, 4)
		for i, p := range points {
			transformedPoints[i] = worldTransform.TransformPoint(p)
		}
		return NewAABBFromPoints(transformedPoints)
	}

	return nil
}

// BuildFrustumSimple builds a simplified frustum (faster, less accurate)
func BuildFrustumSimple(camera *Camera) ViewFrustum {
	frustum := ViewFrustum{}

	pos := camera.GetPosition()

	forward := camera.GetForwardVectorPoint()
	right := camera.GetRightVectorPoint()
	up := camera.GetUpVectorPoint()

	// Simplified plane calculations
	// FIX: Use DegToRad conversion. Camera FOV is in degrees.
	halfFOVXRad := (camera.FOV.X / 2.0) * DegToRad
	halfFOVYRad := (camera.FOV.Y / 2.0) * DegToRad

	// Use angles directly (no Atan)
	angleX := halfFOVXRad
	angleY := halfFOVYRad

	// Near plane
	frustum.Planes[FrustumNear].Normal = forward
	nearCenter := Point{
		X: pos.X + forward.X*camera.Near,
		Y: pos.Y + forward.Y*camera.Near,
		Z: pos.Z + forward.Z*camera.Near,
	}
	frustum.Planes[FrustumNear].Distance = -dotProduct(forward.X, forward.Y, forward.Z, nearCenter.X, nearCenter.Y, nearCenter.Z)

	// Far plane
	frustum.Planes[FrustumFar].Normal = Point{X: -forward.X, Y: -forward.Y, Z: -forward.Z}
	farCenter := Point{
		X: pos.X + forward.X*camera.Far,
		Y: pos.Y + forward.Y*camera.Far,
		Z: pos.Z + forward.Z*camera.Far,
	}
	frustum.Planes[FrustumFar].Distance = -dotProduct(-forward.X, -forward.Y, -forward.Z, farCenter.X, farCenter.Y, farCenter.Z)

	// Simplified side planes using angles
	cosX := math.Cos(angleX)
	sinX := math.Sin(angleX)
	cosY := math.Cos(angleY)
	sinY := math.Sin(angleY)

	// Left
	leftNormal := Point{
		X: forward.X*cosX + right.X*sinX,
		Y: forward.Y*cosX + right.Y*sinX,
		Z: forward.Z*cosX + right.Z*sinX,
	}
	frustum.Planes[FrustumLeft].Normal = leftNormal
	frustum.Planes[FrustumLeft].Distance = -dotProduct(leftNormal.X, leftNormal.Y, leftNormal.Z, pos.X, pos.Y, pos.Z)

	// Right
	rightNormal := Point{
		X: forward.X*cosX - right.X*sinX,
		Y: forward.Y*cosX - right.Y*sinX,
		Z: forward.Z*cosX - right.Z*sinX,
	}
	frustum.Planes[FrustumRight].Normal = rightNormal
	frustum.Planes[FrustumRight].Distance = -dotProduct(rightNormal.X, rightNormal.Y, rightNormal.Z, pos.X, pos.Y, pos.Z)

	// Top
	topNormal := Point{
		X: forward.X*cosY - up.X*sinY,
		Y: forward.Y*cosY - up.Y*sinY,
		Z: forward.Z*cosY - up.Z*sinY,
	}
	frustum.Planes[FrustumTop].Normal = topNormal
	frustum.Planes[FrustumTop].Distance = -dotProduct(topNormal.X, topNormal.Y, topNormal.Z, pos.X, pos.Y, pos.Z)

	// Bottom
	bottomNormal := Point{
		X: forward.X*cosY + up.X*sinY,
		Y: forward.Y*cosY + up.Y*sinY,
		Z: forward.Z*cosY + up.Z*sinY,
	}
	frustum.Planes[FrustumBottom].Normal = bottomNormal
	frustum.Planes[FrustumBottom].Distance = -dotProduct(bottomNormal.X, bottomNormal.Y, bottomNormal.Z, pos.X, pos.Y, pos.Z)

	return frustum
}
