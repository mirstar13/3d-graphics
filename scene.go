package main

// SceneNode represents a node in the scene graph
// Object can be any geometry type: *Triangle, *Quad, *Line, *Mesh, *Circle, *Point
type SceneNode struct {
	Transform *Transform
	Object    any // Can be any geometry type
	Children  []*SceneNode
	Parent    *SceneNode
	Name      string
	Enabled   bool
	Tags      []string
	OnUpdate  func(*SceneNode, float64)
}

// Scene manages the scene graph
type Scene struct {
	Root     *SceneNode
	AllNodes map[string]*SceneNode
	Camera   *Camera
}

// NewScene creates a new scene
func NewScene() *Scene {
	root := &SceneNode{
		Transform: NewTransform(),
		Object:    nil,
		Children:  make([]*SceneNode, 0),
		Parent:    nil,
		Name:      "Root",
		Enabled:   true,
		Tags:      make([]string, 0),
	}

	return &Scene{
		Root:     root,
		AllNodes: map[string]*SceneNode{"Root": root},
		Camera:   NewCamera(),
	}
}

// NewSceneNode creates a new scene node
func NewSceneNode(name string) *SceneNode {
	return &SceneNode{
		Transform: NewTransform(),
		Object:    nil,
		Children:  make([]*SceneNode, 0),
		Parent:    nil,
		Name:      name,
		Enabled:   true,
		Tags:      make([]string, 0),
	}
}

// NewSceneNodeWithObject creates a node with an object
func NewSceneNodeWithObject(name string, obj any) *SceneNode {
	return &SceneNode{
		Transform: NewTransform(),
		Object:    obj,
		Children:  make([]*SceneNode, 0),
		Parent:    nil,
		Name:      name,
		Enabled:   true,
		Tags:      make([]string, 0),
	}
}

// AddNode adds a node to the scene
func (s *Scene) AddNode(node *SceneNode) {
	s.Root.AddChild(node)
	s.AllNodes[node.Name] = node
}

// AddNodeTo adds a node as a child of a parent
func (s *Scene) AddNodeTo(node *SceneNode, parent *SceneNode) {
	parent.AddChild(node)
	s.AllNodes[node.Name] = node
}

// RemoveNode removes a node from the scene
func (s *Scene) RemoveNode(node *SceneNode) {
	if node.Parent != nil {
		node.Parent.RemoveChild(node)
	}
	delete(s.AllNodes, node.Name)
}

// FindNode finds a node by name
func (s *Scene) FindNode(name string) *SceneNode {
	return s.AllNodes[name]
}

// FindNodesByTag finds all nodes with a tag
func (s *Scene) FindNodesByTag(tag string) []*SceneNode {
	results := make([]*SceneNode, 0)
	for _, node := range s.AllNodes {
		if node.HasTag(tag) {
			results = append(results, node)
		}
	}
	return results
}

// GetRenderableNodes returns all nodes with objects
func (s *Scene) GetRenderableNodes() []*SceneNode {
	nodes := make([]*SceneNode, 0)
	s.collectRenderables(s.Root, &nodes)
	return nodes
}

func (s *Scene) collectRenderables(node *SceneNode, renderables *[]*SceneNode) {
	if !node.IsEnabled() {
		return
	}

	if node.Object != nil {
		*renderables = append(*renderables, node)
	}

	for _, child := range node.Children {
		s.collectRenderables(child, renderables)
	}
}

// Update updates all nodes
func (s *Scene) Update(dt float64) {
	s.updateNodeRecursive(s.Root, dt)
}

func (s *Scene) computeNodeBounds(node *SceneNode) *AABB {
	worldMatrix := node.Transform.GetWorldMatrix()

	switch obj := node.Object.(type) {
	case *Mesh:
		if len(obj.Vertices) == 0 {
			return nil
		}

		// Transform all vertices to world space
		// First apply mesh position (local offset), then world transform
		points := make([]Point, len(obj.Vertices))
		for i, v := range obj.Vertices {
			// Vertex position + mesh offset (still in local space)
			localPoint := Point{
				X: v.X + obj.Position.X,
				Y: v.Y + obj.Position.Y,
				Z: v.Z + obj.Position.Z,
			}
			// Now transform to world space
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

	case *LODGroup:
		// Get current mesh from LOD group
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil && len(currentMesh.Vertices) > 0 {
			points := make([]Point, len(currentMesh.Vertices))
			for i, v := range currentMesh.Vertices {
				localPoint := Point{
					X: v.X + currentMesh.Position.X,
					Y: v.Y + currentMesh.Position.Y,
					Z: v.Z + currentMesh.Position.Z,
				}
				points[i] = worldMatrix.TransformPoint(localPoint)
			}
			return NewAABBFromPoints(points)
		}
	}

	return nil
}

func (s *Scene) updateNodeRecursive(node *SceneNode, dt float64) {
	if !node.IsEnabled() {
		return
	}

	if node.OnUpdate != nil {
		node.OnUpdate(node, dt)
	}

	for _, child := range node.Children {
		s.updateNodeRecursive(child, dt)
	}
}

// Clear removes all nodes except root
func (s *Scene) Clear() {
	s.Root.Children = make([]*SceneNode, 0)
	s.AllNodes = map[string]*SceneNode{"Root": s.Root}
}

// GetAllNodes returns all nodes
func (s *Scene) GetAllNodes() []*SceneNode {
	nodes := make([]*SceneNode, 0, len(s.AllNodes))
	for _, node := range s.AllNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetEnabledNodes returns only enabled nodes
func (s *Scene) GetEnabledNodes() []*SceneNode {
	nodes := make([]*SceneNode, 0)
	for _, node := range s.AllNodes {
		if node.IsEnabled() {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// AddChild adds a child node
func (sn *SceneNode) AddChild(child *SceneNode) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}

	child.Parent = sn
	child.Transform.SetParent(sn.Transform)
	sn.Children = append(sn.Children, child)
}

// RemoveChild removes a child node
func (sn *SceneNode) RemoveChild(child *SceneNode) {
	for i, c := range sn.Children {
		if c == child {
			sn.Children = append(sn.Children[:i], sn.Children[i+1:]...)
			child.Parent = nil
			child.Transform.SetParent(nil)
			break
		}
	}
}

// SetEnabled enables/disables the node
func (sn *SceneNode) SetEnabled(enabled bool) {
	sn.Enabled = enabled
}

// IsEnabled checks if the node is enabled
func (sn *SceneNode) IsEnabled() bool {
	if !sn.Enabled {
		return false
	}
	if sn.Parent != nil {
		return sn.Parent.IsEnabled()
	}
	return true
}

// AddTag adds a tag
func (sn *SceneNode) AddTag(tag string) {
	sn.Tags = append(sn.Tags, tag)
}

// HasTag checks if the node has a tag
func (sn *SceneNode) HasTag(tag string) bool {
	for _, t := range sn.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// RemoveTag removes a tag
func (sn *SceneNode) RemoveTag(tag string) {
	for i, t := range sn.Tags {
		if t == tag {
			sn.Tags = append(sn.Tags[:i], sn.Tags[i+1:]...)
			break
		}
	}
}

// GetWorldTransform returns the world transform
func (sn *SceneNode) GetWorldTransform() *Transform {
	worldTransform := NewTransform()
	worldPos := sn.Transform.GetWorldPosition()
	worldRot := sn.Transform.GetWorldRotation()

	worldTransform.SetPosition(worldPos.X, worldPos.Y, worldPos.Z)
	worldTransform.SetRotation(worldRot.X, worldRot.Y, worldRot.Z)
	worldTransform.SetScale(sn.Transform.Scale.X, sn.Transform.Scale.Y, sn.Transform.Scale.Z)

	return worldTransform
}

// RotateLocal rotates the node
func (sn *SceneNode) RotateLocal(dpitch, dyaw, droll float64) {
	sn.Transform.Rotate(dpitch, dyaw, droll)
	sn.MarkTransformDirty()
}

// TranslateLocal translates in local space
func (sn *SceneNode) TranslateLocal(dx, dy, dz float64) {
	right := sn.Transform.GetRightVector()
	up := sn.Transform.GetUpVector()
	forward := sn.Transform.GetForwardVector()

	sn.Transform.Position.X += right.X*dx + up.X*dy + forward.X*dz
	sn.Transform.Position.Y += right.Y*dx + up.Y*dy + forward.Y*dz
	sn.Transform.Position.Z += right.Z*dx + up.Z*dy + forward.Z*dz
	sn.Transform.MarkDirty()
	sn.MarkTransformDirty()
}

// MarkTransformDirty marks transform as dirty
func (sn *SceneNode) MarkTransformDirty() {
	sn.Transform.MarkDirty()

	for _, child := range sn.Children {
		child.MarkTransformDirty()
	}
}

// TransformSceneObject applies the node's transform to its object
// Returns a transformed copy of the object for rendering/physics
func (sn *SceneNode) TransformSceneObject() any {
	if sn.Object == nil {
		return nil
	}

	worldTransform := sn.GetWorldTransform()

	switch obj := sn.Object.(type) {
	case *Triangle:
		transformed := &Triangle{
			P0:           worldTransform.TransformPoint(obj.P0),
			P1:           worldTransform.TransformPoint(obj.P1),
			P2:           worldTransform.TransformPoint(obj.P2),
			char:         obj.char,
			Material:     obj.Material,
			UseSetNormal: obj.UseSetNormal,
		}

		if obj.UseSetNormal && obj.Normal != nil {
			transformedNormal := worldTransform.TransformDirection(*obj.Normal)
			transformed.Normal = &transformedNormal
		}

		return transformed

	case *Quad:
		transformed := &Quad{
			P0:           worldTransform.TransformPoint(obj.P0),
			P1:           worldTransform.TransformPoint(obj.P1),
			P2:           worldTransform.TransformPoint(obj.P2),
			P3:           worldTransform.TransformPoint(obj.P3),
			Material:     obj.Material,
			UseSetNormal: obj.UseSetNormal,
		}

		if obj.UseSetNormal && obj.Normal != nil {
			transformedNormal := worldTransform.TransformDirection(*obj.Normal)
			transformed.Normal = &transformedNormal
		}

		return transformed

	case *Line:
		return &Line{
			Start: worldTransform.TransformPoint(obj.Start),
			End:   worldTransform.TransformPoint(obj.End),
		}

	case *Circle:
		transformedPoints := make([]Point, len(obj.Points))
		for i, p := range obj.Points {
			transformedPoints[i] = worldTransform.TransformPoint(p)
		}
		return &Circle{
			Center: worldTransform.TransformPoint(obj.Center),
			Radius: obj.Radius,
			Points: transformedPoints,
		}

	case *Mesh:
		transformedMesh := NewMesh()
		transformedMesh.Position = worldTransform.TransformPoint(obj.Position)
		transformedMesh.Material = obj.Material // Copy mesh material for Indexed mode

		// Handle Optimized Indexed Geometry
		if len(obj.Vertices) > 0 {
			// Transform vertices
			transformedMesh.Vertices = make([]Point, len(obj.Vertices))
			for i, v := range obj.Vertices {
				transformedMesh.Vertices[i] = worldTransform.TransformPoint(v)
			}
			// Copy indices directly (structure doesn't change with transform)
			transformedMesh.Indices = make([]int, len(obj.Indices))
			copy(transformedMesh.Indices, obj.Indices)
		}

		return transformedMesh
	}

	return sn.Object
}

func (sn *SceneNode) SetObject(obj any) {
	sn.Object = obj
}

// CreateCube creates a cube scene node using indexed geometry
func (s *Scene) CreateCube(name string, size float64, material Material) *SceneNode {
	node := NewSceneNode(name)
	mesh := NewMesh()
	mesh.Material = material
	d := size

	// 1. Add Vertices
	// Front Face
	v0 := mesh.AddVertex(-d, -d, -d) // 0
	v1 := mesh.AddVertex(d, -d, -d)  // 1
	v2 := mesh.AddVertex(d, d, -d)   // 2
	v3 := mesh.AddVertex(-d, d, -d)  // 3
	// Back Face
	v4 := mesh.AddVertex(-d, -d, d) // 4
	v5 := mesh.AddVertex(d, -d, d)  // 5
	v6 := mesh.AddVertex(d, d, d)   // 6
	v7 := mesh.AddVertex(-d, d, d)  // 7

	// 2. Add Indices (Quads converted to 2 triangles)
	// Front (v1, v0, v3, v2) -> Normal -Z
	mesh.AddQuadIndices(v1, v0, v3, v2)

	// Back (v4, v5, v6, v7) -> Normal +Z
	mesh.AddQuadIndices(v4, v5, v6, v7)

	// Right (v5, v1, v2, v6) -> Normal +X
	mesh.AddQuadIndices(v5, v1, v2, v6)

	// Left (v0, v4, v7, v3) -> Normal -X
	mesh.AddQuadIndices(v0, v4, v7, v3)

	// Top (v7, v6, v2, v3) -> Normal +Y
	mesh.AddQuadIndices(v7, v6, v2, v3)

	// Bottom (v0, v1, v5, v4) -> Normal -Y
	mesh.AddQuadIndices(v0, v1, v5, v4)

	node.Object = mesh
	s.AddNode(node)
	return node
}

// CreateSphere creates a sphere scene node using optimized geometry
func (s *Scene) CreateSphere(name string, radius float64, rings, sectors int, material Material) *SceneNode {
	node := NewSceneNode(name)
	// Use the optimized generator
	mesh := GenerateSphere(radius, rings, sectors)
	// Assign the material to the whole mesh
	mesh.Material = material

	node.Object = mesh
	s.AddNode(node)
	return node
}

// CreateEmpty creates an empty scene node
func (s *Scene) CreateEmpty(name string) *SceneNode {
	node := NewSceneNode(name)
	s.AddNode(node)
	return node
}
