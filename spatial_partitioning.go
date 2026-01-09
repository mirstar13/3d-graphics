package main

import (
	"math"
	"sort"
)

// OctreeNode represents a node in the octree
type OctreeNode struct {
	Bounds   *AABB
	Objects  []*SceneNode
	Children [8]*OctreeNode
	IsLeaf   bool
	Depth    int
}

// Octree manages spatial partitioning for static scenes
type Octree struct {
	Root           *OctreeNode
	MaxDepth       int
	MaxObjectsLeaf int
	TotalNodes     int
	TotalObjects   int
}

// NewOctree creates a new octree
func NewOctree(bounds *AABB, maxDepth, maxObjectsPerLeaf int) *Octree {
	return &Octree{
		Root: &OctreeNode{
			Bounds:   bounds,
			Objects:  make([]*SceneNode, 0),
			Children: [8]*OctreeNode{},
			IsLeaf:   true,
			Depth:    0,
		},
		MaxDepth:       maxDepth,
		MaxObjectsLeaf: maxObjectsPerLeaf,
		TotalNodes:     1,
	}
}

// Insert adds an object to the octree
func (ot *Octree) Insert(node *SceneNode, bounds *AABB) {
	ot.insertRecursive(ot.Root, node, bounds)
	ot.TotalObjects++
}

func (ot *Octree) insertRecursive(octNode *OctreeNode, sceneNode *SceneNode, bounds *AABB) {
	// If this is a leaf and we haven't exceeded limits, add here
	if octNode.IsLeaf {
		octNode.Objects = append(octNode.Objects, sceneNode)

		// Check if we need to subdivide
		if len(octNode.Objects) > ot.MaxObjectsLeaf && octNode.Depth < ot.MaxDepth {
			ot.subdivide(octNode)
		}
		return
	}

	// Not a leaf - find which children contain this object
	childIdx := ot.getChildIndices(octNode, bounds)
	for _, idx := range childIdx {
		if octNode.Children[idx] != nil {
			ot.insertRecursive(octNode.Children[idx], sceneNode, bounds)
		}
	}
}

// subdivide splits a leaf node into 8 children
func (ot *Octree) subdivide(node *OctreeNode) {
	node.IsLeaf = false
	center := node.Bounds.GetCenter()

	// Create 8 octants
	octants := [8]*AABB{
		// Bottom 4 (Z-)
		NewAABB(node.Bounds.Min, center),
		NewAABB(Point{X: center.X, Y: node.Bounds.Min.Y, Z: node.Bounds.Min.Z},
			Point{X: node.Bounds.Max.X, Y: center.Y, Z: center.Z}),
		NewAABB(Point{X: node.Bounds.Min.X, Y: center.Y, Z: node.Bounds.Min.Z},
			Point{X: center.X, Y: node.Bounds.Max.Y, Z: center.Z}),
		NewAABB(Point{X: center.X, Y: center.Y, Z: node.Bounds.Min.Z},
			Point{X: node.Bounds.Max.X, Y: node.Bounds.Max.Y, Z: center.Z}),
		// Top 4 (Z+)
		NewAABB(Point{X: node.Bounds.Min.X, Y: node.Bounds.Min.Y, Z: center.Z},
			Point{X: center.X, Y: center.Y, Z: node.Bounds.Max.Z}),
		NewAABB(Point{X: center.X, Y: node.Bounds.Min.Y, Z: center.Z},
			Point{X: node.Bounds.Max.X, Y: center.Y, Z: node.Bounds.Max.Z}),
		NewAABB(Point{X: node.Bounds.Min.X, Y: center.Y, Z: center.Z},
			Point{X: center.X, Y: node.Bounds.Max.Y, Z: node.Bounds.Max.Z}),
		NewAABB(center, node.Bounds.Max),
	}

	// Create child nodes
	for i := 0; i < 8; i++ {
		node.Children[i] = &OctreeNode{
			Bounds:   octants[i],
			Objects:  make([]*SceneNode, 0),
			Children: [8]*OctreeNode{},
			IsLeaf:   true,
			Depth:    node.Depth + 1,
		}
		ot.TotalNodes++
	}

	// Redistribute objects to children
	for _, obj := range node.Objects {
		objBounds := ot.getObjectBounds(obj)
		if objBounds != nil {
			childIdx := ot.getChildIndices(node, objBounds)
			for _, idx := range childIdx {
				node.Children[idx].Objects = append(node.Children[idx].Objects, obj)
			}
		}
	}

	// Clear parent's object list
	node.Objects = nil
}

// getChildIndices returns which children intersect with the bounds
func (ot *Octree) getChildIndices(node *OctreeNode, bounds *AABB) []int {
	indices := make([]int, 0, 8)
	center := node.Bounds.GetCenter()

	// Check each octant
	for i := 0; i < 8; i++ {
		if node.Children[i] != nil && node.Children[i].Bounds.IntersectsAABB(bounds) {
			indices = append(indices, i)
		}
	}

	// If no children yet (during subdivision), determine by position
	if len(indices) == 0 {
		objCenter := bounds.GetCenter()
		idx := 0
		if objCenter.X >= center.X {
			idx |= 1
		}
		if objCenter.Y >= center.Y {
			idx |= 2
		}
		if objCenter.Z >= center.Z {
			idx |= 4
		}
		indices = append(indices, idx)
	}

	return indices
}

// Query returns all objects that intersect with the given bounds
func (ot *Octree) Query(bounds *AABB) []*SceneNode {
	results := make([]*SceneNode, 0)
	visited := make(map[*SceneNode]bool)
	ot.queryRecursive(ot.Root, bounds, &results, visited)
	return results
}

func (ot *Octree) queryRecursive(node *OctreeNode, bounds *AABB, results *[]*SceneNode, visited map[*SceneNode]bool) {
	// Check if query bounds intersect this node
	if !node.Bounds.IntersectsAABB(bounds) {
		return
	}

	// If leaf, return objects
	if node.IsLeaf {
		for _, obj := range node.Objects {
			if !visited[obj] {
				*results = append(*results, obj)
				visited[obj] = true
			}
		}
		return
	}

	// Recurse to children
	for i := 0; i < 8; i++ {
		if node.Children[i] != nil {
			ot.queryRecursive(node.Children[i], bounds, results, visited)
		}
	}
}

// RayQuery returns objects that intersect with a ray
func (ot *Octree) RayQuery(ray Ray, maxDistance float64) []*SceneNode {
	results := make([]*SceneNode, 0)
	visited := make(map[*SceneNode]bool)
	ot.rayQueryRecursive(ot.Root, ray, maxDistance, &results, visited)
	return results
}

func (ot *Octree) rayQueryRecursive(node *OctreeNode, ray Ray, maxDist float64, results *[]*SceneNode, visited map[*SceneNode]bool) {
	// Check if ray intersects this node
	hit, dist := node.Bounds.IntersectsRay(ray)
	if !hit || dist > maxDist {
		return
	}

	if node.IsLeaf {
		for _, obj := range node.Objects {
			if !visited[obj] {
				*results = append(*results, obj)
				visited[obj] = true
			}
		}
		return
	}

	// Recurse to children
	for i := 0; i < 8; i++ {
		if node.Children[i] != nil {
			ot.rayQueryRecursive(node.Children[i], ray, maxDist, results, visited)
		}
	}
}

// getObjectBounds computes bounds for a scene node
func (ot *Octree) getObjectBounds(node *SceneNode) *AABB {
	worldMatrix := node.Transform.GetWorldMatrix()

	switch obj := node.Object.(type) {
	case *Mesh:
		if len(obj.Vertices) == 0 {
			return nil
		}
		points := make([]Point, len(obj.Vertices))
		for i, v := range obj.Vertices {
			localPoint := Point{
				X: v.X + obj.Position.X,
				Y: v.Y + obj.Position.Y,
				Z: v.Z + obj.Position.Z,
			}
			points[i] = worldMatrix.TransformPoint(localPoint)
		}
		return NewAABBFromPoints(points)

	case *Triangle:
		p0 := worldMatrix.TransformPoint(obj.P0)
		p1 := worldMatrix.TransformPoint(obj.P1)
		p2 := worldMatrix.TransformPoint(obj.P2)
		return NewAABBFromPoints([]Point{p0, p1, p2})

	case *Quad:
		p0 := worldMatrix.TransformPoint(obj.P0)
		p1 := worldMatrix.TransformPoint(obj.P1)
		p2 := worldMatrix.TransformPoint(obj.P2)
		p3 := worldMatrix.TransformPoint(obj.P3)
		return NewAABBFromPoints([]Point{p0, p1, p2, p3})
	}
	return nil
}

// BVHNode represents a node in the BVH
type BVHNode struct {
	Bounds *AABB
	Left   *BVHNode
	Right  *BVHNode
	Object *SceneNode
	IsLeaf bool
}

// BVH manages dynamic spatial partitioning
type BVH struct {
	Root         *BVHNode
	Objects      []*SceneNode
	ObjectBounds []*AABB
	TotalNodes   int
}

// NewBVH creates a new BVH from a list of objects
func NewBVH(objects []*SceneNode) *BVH {
	bvh := &BVH{
		Objects:      objects,
		ObjectBounds: make([]*AABB, len(objects)),
	}

	// Compute bounds for all objects
	for i, obj := range objects {
		bvh.ObjectBounds[i] = bvh.computeObjectBounds(obj)
	}

	// Build tree
	if len(objects) > 0 {
		bvh.Root = bvh.buildRecursive(0, len(objects))
	}

	return bvh
}

// buildRecursive builds the BVH tree recursively using SAH (Surface Area Heuristic)
func (bvh *BVH) buildRecursive(start, end int) *BVHNode {
	bvh.TotalNodes++
	node := &BVHNode{}

	// Compute bounds for this node
	node.Bounds = bvh.ObjectBounds[start]
	for i := start + 1; i < end; i++ {
		node.Bounds = node.Bounds.Merge(bvh.ObjectBounds[i])
	}

	numObjects := end - start

	// Leaf node
	if numObjects == 1 {
		node.IsLeaf = true
		node.Object = bvh.Objects[start]
		return node
	}

	// Find best split using SAH
	bestAxis, bestSplit := bvh.findBestSplit(start, end)

	// Sort objects along best axis
	bvh.sortObjectsAlongAxis(start, end, bestAxis)

	// Split at best position
	mid := start + bestSplit
	if mid <= start {
		mid = start + 1
	}
	if mid >= end {
		mid = end - 1
	}

	// Build children
	node.Left = bvh.buildRecursive(start, mid)
	node.Right = bvh.buildRecursive(mid, end)
	node.IsLeaf = false

	return node
}

// findBestSplit finds the best axis and split position using SAH
func (bvh *BVH) findBestSplit(start, end int) (int, int) {
	bestCost := math.Inf(1)
	bestAxis := 0
	bestSplit := (end - start) / 2

	// Try each axis
	for axis := 0; axis < 3; axis++ {
		// Try different split positions
		numBuckets := 12
		if end-start < numBuckets {
			numBuckets = end - start
		}

		for bucket := 1; bucket < numBuckets; bucket++ {
			splitPos := start + (end-start)*bucket/numBuckets

			// Compute cost
			leftBounds := bvh.ObjectBounds[start]
			for i := start + 1; i < splitPos; i++ {
				leftBounds = leftBounds.Merge(bvh.ObjectBounds[i])
			}

			rightBounds := bvh.ObjectBounds[splitPos]
			for i := splitPos + 1; i < end; i++ {
				rightBounds = rightBounds.Merge(bvh.ObjectBounds[i])
			}

			cost := bvh.computeSAH(leftBounds, rightBounds, splitPos-start, end-splitPos)

			if cost < bestCost {
				bestCost = cost
				bestAxis = axis
				bestSplit = splitPos - start
			}
		}
	}

	return bestAxis, bestSplit
}

// computeSAH computes Surface Area Heuristic cost
func (bvh *BVH) computeSAH(leftBounds, rightBounds *AABB, leftCount, rightCount int) float64 {
	leftArea := bvh.surfaceArea(leftBounds)
	rightArea := bvh.surfaceArea(rightBounds)
	return leftArea*float64(leftCount) + rightArea*float64(rightCount)
}

// surfaceArea computes surface area of AABB
func (bvh *BVH) surfaceArea(bounds *AABB) float64 {
	size := bounds.GetSize()
	return 2.0 * (size.X*size.Y + size.Y*size.Z + size.Z*size.X)
}

func (bvh *BVH) sortObjectsAlongAxis(start, end, axis int) {
	sort.Slice(bvh.Objects[start:end], func(i, j int) bool {
		center1 := bvh.ObjectBounds[start+i].GetCenter()
		center2 := bvh.ObjectBounds[start+j].GetCenter()

		switch axis {
		case 0:
			return center1.X < center2.X
		case 1:
			return center1.Y < center2.Y
		case 2:
			return center1.Z < center2.Z
		}
		return false
	})

	// Also sort bounds array in parallel
	sortedBounds := make([]*AABB, end-start)
	for i := 0; i < end-start; i++ {
		sortedBounds[i] = bvh.ObjectBounds[start+i]
	}
	copy(bvh.ObjectBounds[start:end], sortedBounds)
}

// Query returns objects intersecting with bounds
func (bvh *BVH) Query(bounds *AABB) []*SceneNode {
	if bvh.Root == nil {
		return nil
	}

	results := make([]*SceneNode, 0)
	bvh.queryRecursive(bvh.Root, bounds, &results)
	return results
}

func (bvh *BVH) queryRecursive(node *BVHNode, bounds *AABB, results *[]*SceneNode) {
	if !node.Bounds.IntersectsAABB(bounds) {
		return
	}

	if node.IsLeaf {
		*results = append(*results, node.Object)
		return
	}

	if node.Left != nil {
		bvh.queryRecursive(node.Left, bounds, results)
	}
	if node.Right != nil {
		bvh.queryRecursive(node.Right, bounds, results)
	}
}

// RayQuery returns objects intersecting with ray
func (bvh *BVH) RayQuery(ray Ray, maxDistance float64) []*SceneNode {
	if bvh.Root == nil {
		return nil
	}

	results := make([]*SceneNode, 0)
	bvh.rayQueryRecursive(bvh.Root, ray, maxDistance, &results)
	return results
}

func (bvh *BVH) rayQueryRecursive(node *BVHNode, ray Ray, maxDist float64, results *[]*SceneNode) {
	hit, dist := node.Bounds.IntersectsRay(ray)
	if !hit || dist > maxDist {
		return
	}

	if node.IsLeaf {
		*results = append(*results, node.Object)
		return
	}

	if node.Left != nil {
		bvh.rayQueryRecursive(node.Left, ray, maxDist, results)
	}
	if node.Right != nil {
		bvh.rayQueryRecursive(node.Right, ray, maxDist, results)
	}
}

// Rebuild rebuilds the BVH (call when objects move significantly)
func (bvh *BVH) Rebuild() {
	// Update bounds
	for i, obj := range bvh.Objects {
		bvh.ObjectBounds[i] = bvh.computeObjectBounds(obj)
	}

	// Rebuild tree
	bvh.TotalNodes = 0
	if len(bvh.Objects) > 0 {
		bvh.Root = bvh.buildRecursive(0, len(bvh.Objects))
	}
}

func (bvh *BVH) computeObjectBounds(node *SceneNode) *AABB {
	worldMatrix := node.Transform.GetWorldMatrix()

	switch obj := node.Object.(type) {
	case *Mesh:
		if len(obj.Vertices) == 0 {
			return NewAABB(Point{}, Point{})
		}
		points := make([]Point, len(obj.Vertices))
		for i, v := range obj.Vertices {
			localPoint := Point{
				X: v.X + obj.Position.X,
				Y: v.Y + obj.Position.Y,
				Z: v.Z + obj.Position.Z,
			}
			points[i] = worldMatrix.TransformPoint(localPoint)
		}
		return NewAABBFromPoints(points)

	case *Triangle:
		p0 := worldMatrix.TransformPoint(obj.P0)
		p1 := worldMatrix.TransformPoint(obj.P1)
		p2 := worldMatrix.TransformPoint(obj.P2)
		return NewAABBFromPoints([]Point{p0, p1, p2})

	case *Quad:
		p0 := worldMatrix.TransformPoint(obj.P0)
		p1 := worldMatrix.TransformPoint(obj.P1)
		p2 := worldMatrix.TransformPoint(obj.P2)
		p3 := worldMatrix.TransformPoint(obj.P3)
		return NewAABBFromPoints([]Point{p0, p1, p2, p3})
	}

	pos := node.Transform.GetWorldPosition()
	return NewAABB(pos, pos)
}

// BuildOctree builds an octree for the scene
func (s *Scene) BuildOctree(maxDepth, maxObjectsPerLeaf int) *Octree {
	// Compute scene bounds
	allNodes := s.GetEnabledNodes()
	if len(allNodes) == 0 {
		return nil
	}

	// Find world bounds
	var sceneBounds *AABB
	for _, node := range allNodes {
		if node.Object != nil {
			bounds := s.computeNodeBounds(node)
			if bounds != nil {
				if sceneBounds == nil {
					sceneBounds = bounds
				} else {
					sceneBounds = sceneBounds.Merge(bounds)
				}
			}
		}
	}

	if sceneBounds == nil {
		return nil
	}

	// Expand bounds slightly
	sceneBounds = sceneBounds.Expand(10.0)

	// Create octree
	octree := NewOctree(sceneBounds, maxDepth, maxObjectsPerLeaf)

	// Insert all objects
	for _, node := range allNodes {
		if node.Object != nil {
			bounds := s.computeNodeBounds(node)
			if bounds != nil {
				octree.Insert(node, bounds)
			}
		}
	}

	return octree
}

// BuildBVH builds a BVH for the scene
func (s *Scene) BuildBVH() *BVH {
	allNodes := s.GetEnabledNodes()
	if len(allNodes) == 0 {
		return nil
	}

	// Filter nodes with objects
	objectNodes := make([]*SceneNode, 0)
	for _, node := range allNodes {
		if node.Object != nil {
			objectNodes = append(objectNodes, node)
		}
	}

	return NewBVH(objectNodes)
}
