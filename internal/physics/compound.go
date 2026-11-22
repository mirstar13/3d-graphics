package physics

import "fmt"

// CompoundChild specifies a centred primitive's local pose and positive relative
// mass weight. Child order is stable and forms part of contact feature identity.
type CompoundChild struct {
	Shape    Shape
	Position vec
	Rotation quat
	Weight   float64
}
type compoundShape struct {
	children []CompoundChild
	center   vec
	inertia  InertiaTensor
}

// Compound copies up to 32 child spheres/boxes and recentres them about their
// weighted centre of mass. Body poses refer to that centre, not the input origin.
func Compound(children []CompoundChild) (Shape, error) {
	if len(children) == 0 || len(children) > 32 {
		return Shape{}, fmt.Errorf("compound needs 1..32 children")
	}
	data := &compoundShape{children: append([]CompoundChild(nil), children...)}
	total := 0.0
	for i := range data.children {
		child := &data.children[i]
		if child.Shape.kind != SphereShape && child.Shape.kind != BoxShape {
			return Shape{}, fmt.Errorf("compound children must be spheres or boxes")
		}
		if err := (Collider{Shape: child.Shape}).validate(Dynamic); err != nil {
			return Shape{}, err
		}
		if !finitePoint(child.Position) || !validRotation(child.Rotation) || !finite(child.Weight) || child.Weight <= 0 {
			return Shape{}, fmt.Errorf("invalid compound child pose or weight")
		}
		child.Rotation = child.Rotation.Normalize()
		total += child.Weight
		data.center = add(data.center, mul(child.Position, child.Weight))
	}
	if !finite(total) || !finitePoint(data.center) {
		return Shape{}, fmt.Errorf("compound mass overflow")
	}
	data.center = mul(data.center, 1/total)
	for k := range data.children {
		child := &data.children[k]
		child.Weight /= total
		child.Position = sub(child.Position, data.center)
		local := tensorFor(child.Shape, child.Weight)
		axes := [3]vec{child.Rotation.RotateVector(axis(0)), child.Rotation.RotateVector(axis(1)), child.Rotation.RotateVector(axis(2))}
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				for u := 0; u < 3; u++ {
					for v := 0; v < 3; v++ {
						data.inertia[i][j] += component(axes[u], i) * local[u][v] * component(axes[v], j)
					}
				}
				shift := -component(child.Position, i) * component(child.Position, j)
				if i == j {
					shift += dot(child.Position, child.Position)
				}
				data.inertia[i][j] += child.Weight * shift
			}
		}
	}
	if _, ok := tensorInverse(data.inertia); !ok {
		return Shape{}, fmt.Errorf("compound inertia is singular or nonfinite")
	}
	return Shape{kind: CompoundShape, compound: data}, nil
}

// CenterOfMass returns the offset from the construction origin to the mass centre.
func (s Shape) CenterOfMass() vec {
	if s.compound != nil {
		return s.compound.center
	}
	return vec{}
}

// Children returns centred child poses without exposing mutable shape storage.
func (s Shape) Children() []CompoundChild {
	if s.compound == nil {
		return nil
	}
	return append([]CompoundChild(nil), s.compound.children...)
}
func compoundBody(parent *body, child CompoundChild) *body {
	b := *parent
	b.state.Collider.Shape = child.Shape
	b.state.Position = add(parent.state.Position, parent.state.Rotation.RotateVector(child.Position))
	b.state.Rotation = parent.state.Rotation.Multiply(child.Rotation)
	return &b
}
func compoundContacts(a, b *body) []Contact {
	var contacts []Contact
	for index, child := range a.state.Collider.Shape.compound.children {
		cs := collide(compoundBody(a, child), b)
		for i := range cs {
			cs[i].Feature = (cs[i].Feature ^ uint64(index+1)*0x9e3779b97f4a7c15) * 1099511628211
		}
		// Keep each child's support manifold; a global four-point reduction would
		// discard distinct support regions on long or disconnected compounds.
		contacts = append(contacts, cs...)
	}
	return contacts
}
