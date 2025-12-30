package main

// SceneNode represents a node in the scene graph
type SceneNode struct {
	Transform *Transform
	Object    Drawable
	Children  []*SceneNode
	Parent    *SceneNode
	Name      string
	Enabled   bool
	Tags      []string                  // For grouping/filtering nodes
	OnUpdate  func(*SceneNode, float64) // Optional update callback
}

// Scene manages the entire scene graph and provides high-level API
type Scene struct {
	Root     *SceneNode
	AllNodes map[string]*SceneNode // Map by name for fast lookup
	Camera   *Camera               // Active camera for this scene
}

// NewScene creates a new empty scene
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

// NewSceneNodeWithObject creates a scene node with a drawable object
func NewSceneNodeWithObject(name string, obj Drawable) *SceneNode {
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

// AddNode adds a node to the scene (as a child of root)
func (s *Scene) AddNode(node *SceneNode) {
	s.Root.AddChild(node)
	s.AllNodes[node.Name] = node
}

// AddNodeTo adds a node as a child of a specific parent
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

// FindNode finds a node by name (O(1) lookup)
func (s *Scene) FindNode(name string) *SceneNode {
	return s.AllNodes[name]
}

// FindNodesByTag finds all nodes with a specific tag
func (s *Scene) FindNodesByTag(tag string) []*SceneNode {
	results := make([]*SceneNode, 0)
	for _, node := range s.AllNodes {
		if node.HasTag(tag) {
			results = append(results, node)
		}
	}
	return results
}

// GetRenderableObjects returns all objects ready for rendering
// Applies transforms and filters disabled nodes
func (s *Scene) GetRenderableObjects() []Drawable {
	drawables := make([]Drawable, 0)
	s.collectRenderables(s.Root, &drawables)
	return drawables
}

func (s *Scene) collectRenderables(node *SceneNode, drawables *[]Drawable) {
	if !node.IsEnabled() {
		return
	}

	// Add this node's object WITHOUT frustum culling here
	// Frustum culling should happen at render time on individual primitives
	if node.Object != nil {
		transformed := node.TransformSceneObject()
		if transformed != nil {
			*drawables = append(*drawables, transformed)
		}
	}

	// Always recurse to children (don't cull parent hierarchies)
	for _, child := range node.Children {
		s.collectRenderables(child, drawables)
	}
}

// Update updates all nodes in the scene
func (s *Scene) Update(dt float64) {
	s.updateNodeRecursive(s.Root, dt)
}

func (s *Scene) updateNodeRecursive(node *SceneNode, dt float64) {
	if !node.IsEnabled() {
		return
	}

	// Call update callback if node has one
	if node.OnUpdate != nil {
		node.OnUpdate(node, dt)
	}

	// Recursively update children
	for _, child := range node.Children {
		s.updateNodeRecursive(child, dt)
	}
}

// AddChild adds a child node to this node
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

// SetEnabled enables or disables this node (affects rendering)
func (sn *SceneNode) SetEnabled(enabled bool) {
	sn.Enabled = enabled
}

// IsEnabled checks if this node is enabled (checks parent chain)
func (sn *SceneNode) IsEnabled() bool {
	if !sn.Enabled {
		return false
	}
	if sn.Parent != nil {
		return sn.Parent.IsEnabled()
	}
	return true
}

// AddTag adds a tag to this node
func (sn *SceneNode) AddTag(tag string) {
	sn.Tags = append(sn.Tags, tag)
}

// HasTag checks if this node has a specific tag
func (sn *SceneNode) HasTag(tag string) bool {
	for _, t := range sn.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// RemoveTag removes a tag from this node
func (sn *SceneNode) RemoveTag(tag string) {
	for i, t := range sn.Tags {
		if t == tag {
			sn.Tags = append(sn.Tags[:i], sn.Tags[i+1:]...)
			break
		}
	}
}

// GetWorldTransform returns the world-space transform for a node
func (sn *SceneNode) GetWorldTransform() *Transform {
	worldTransform := NewTransform()
	worldPos := sn.Transform.GetWorldPosition()
	worldRot := sn.Transform.GetWorldRotation()

	worldTransform.SetPosition(worldPos.X, worldPos.Y, worldPos.Z)
	worldTransform.SetRotation(worldRot.X, worldRot.Y, worldRot.Z)
	worldTransform.SetScale(sn.Transform.Scale.X, sn.Transform.Scale.Y, sn.Transform.Scale.Z)

	return worldTransform
}

// TransformSceneObject applies the node's transform to its object
func (sn *SceneNode) TransformSceneObject() Drawable {
	if sn.Object == nil {
		return nil
	}

	worldTransform := sn.GetWorldTransform()

	// Create a transformed copy based on object type
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

		// Transform the normal if it's set
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

		// Transform the normal if it's set
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

		for _, tri := range obj.Triangles {
			transformed := &Triangle{
				P0:           worldTransform.TransformPoint(tri.P0),
				P1:           worldTransform.TransformPoint(tri.P1),
				P2:           worldTransform.TransformPoint(tri.P2),
				char:         tri.char,
				Material:     tri.Material,
				UseSetNormal: tri.UseSetNormal,
			}

			// Transform the normal if it's set
			if tri.UseSetNormal && tri.Normal != nil {
				transformedNormal := worldTransform.TransformDirection(*tri.Normal)
				transformed.Normal = &transformedNormal
			}

			transformedMesh.AddTriangle(transformed)
		}

		for _, quad := range obj.Quads {
			transformed := &Quad{
				P0:           worldTransform.TransformPoint(quad.P0),
				P1:           worldTransform.TransformPoint(quad.P1),
				P2:           worldTransform.TransformPoint(quad.P2),
				P3:           worldTransform.TransformPoint(quad.P3),
				Material:     quad.Material,
				UseSetNormal: quad.UseSetNormal,
			}

			// Transform the normal if it's set
			if quad.UseSetNormal && quad.Normal != nil {
				transformedNormal := worldTransform.TransformDirection(*quad.Normal)
				transformed.Normal = &transformedNormal
			}

			transformedMesh.AddQuad(transformed)
		}

		return transformedMesh
	}

	return sn.Object
}

// RotateLocal rotates this node around its local axes
func (sn *SceneNode) RotateLocal(dpitch, dyaw, droll float64) {
	sn.Transform.Rotate(dpitch, dyaw, droll)
}

// TranslateLocal moves this node in its local space
func (sn *SceneNode) TranslateLocal(dx, dy, dz float64) {
	right := sn.Transform.GetRightVector()
	up := sn.Transform.GetUpVector()
	forward := sn.Transform.GetForwardVector()

	sn.Transform.Position.X += right.X*dx + up.X*dy + forward.X*dz
	sn.Transform.Position.Y += right.Y*dx + up.Y*dy + forward.Y*dz
	sn.Transform.Position.Z += right.Z*dx + up.Z*dy + forward.Z*dz
}

// OnUpdate is called each frame for this node (optional)
var OnUpdate func(*SceneNode, float64)

// CreateCube creates a cube as a scene node
func (s *Scene) CreateCube(name string, size float64, material Material) *SceneNode {
	node := NewSceneNode(name)
	mesh := NewMesh()
	d := size

	// Define 8 vertices of the cube
	v0 := Point{X: -d, Y: -d, Z: -d} // 0: left-bottom-back
	v1 := Point{X: d, Y: -d, Z: -d}  // 1: right-bottom-back
	v2 := Point{X: d, Y: d, Z: -d}   // 2: right-top-back
	v3 := Point{X: -d, Y: d, Z: -d}  // 3: left-top-back
	v4 := Point{X: -d, Y: -d, Z: d}  // 4: left-bottom-front
	v5 := Point{X: d, Y: -d, Z: d}   // 5: right-bottom-front
	v6 := Point{X: d, Y: d, Z: d}    // 6: right-top-front
	v7 := Point{X: -d, Y: d, Z: d}   // 7: left-top-front

	// Helper function to create a quad with explicit normal
	createQuad := func(p0, p1, p2, p3 Point, normal Point) {
		// Create two triangles with the explicit normal
		t1 := NewTriangle(p0, p1, p2, 'x').SetMaterial(material)
		t1.SetNormal(normal)
		mesh.AddTriangle(t1)

		t2 := NewTriangle(p0, p2, p3, 'x').SetMaterial(material)
		t2.SetNormal(normal)
		mesh.AddTriangle(t2)
	}

	// Front face (Z+): looking at it from +Z direction
	// Winding: counter-clockwise from outside
	createQuad(v4, v5, v6, v7, Point{X: 0, Y: 0, Z: 1})

	// Back face (Z-): looking at it from -Z direction
	createQuad(v1, v0, v3, v2, Point{X: 0, Y: 0, Z: -1})

	// Right face (X+): looking at it from +X direction
	createQuad(v5, v1, v2, v6, Point{X: 1, Y: 0, Z: 0})

	// Left face (X-): looking at it from -X direction
	createQuad(v0, v4, v7, v3, Point{X: -1, Y: 0, Z: 0})

	// Top face (Y+): looking at it from +Y direction
	createQuad(v7, v6, v2, v3, Point{X: 0, Y: 1, Z: 0})

	// Bottom face (Y-): looking at it from -Y direction
	createQuad(v0, v1, v5, v4, Point{X: 0, Y: -1, Z: 0})

	node.Object = mesh
	s.AddNode(node)
	return node
}

// CreateSphere creates a sphere as a scene node
func (s *Scene) CreateSphere(name string, radius float64, rings, sectors int, material Material) *SceneNode {
	node := NewSceneNode(name)
	mesh := GenerateSphere(radius, rings, sectors, material)
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

// GetAllNodes returns all nodes in the scene
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

// Clear removes all nodes except root
func (s *Scene) Clear() {
	s.Root.Children = make([]*SceneNode, 0)
	s.AllNodes = map[string]*SceneNode{"Root": s.Root}
}
