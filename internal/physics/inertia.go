package physics

// InertiaTensor is a symmetric body-local 3x3 inertia matrix about the centre of mass.
type InertiaTensor [3][3]float64

func tensorVector(t InertiaTensor, v vec) vec {
	return vec{X: t[0][0]*v.X + t[0][1]*v.Y + t[0][2]*v.Z, Y: t[1][0]*v.X + t[1][1]*v.Y + t[1][2]*v.Z, Z: t[2][0]*v.X + t[2][1]*v.Y + t[2][2]*v.Z}
}
func tensorScale(t InertiaTensor, s float64) InertiaTensor {
	for i := range t {
		for j := range t[i] {
			t[i][j] *= s
		}
	}
	return t
}
func tensorInverse(t InertiaTensor) (InertiaTensor, bool) {
	a, b, c, d, e, f := t[0][0], t[0][1], t[0][2], t[1][1], t[1][2], t[2][2]
	determinant := a*(d*f-e*e) - b*(b*f-c*e) + c*(b*e-c*d)
	if !finite(determinant) || a <= 0 || a*d-b*b <= 0 || determinant <= 0 {
		return InertiaTensor{}, false
	}
	result := tensorScale(InertiaTensor{{d*f - e*e, c*e - b*f, b*e - c*d}, {c*e - b*f, a*f - c*c, b*c - a*e}, {b*e - c*d, b*c - a*e, a*d - b*b}}, 1/determinant)
	for i := range result {
		for j := range result[i] {
			if !finite(result[i][j]) {
				return InertiaTensor{}, false
			}
		}
	}
	return result, true
}
func tensorFor(s Shape, mass float64) InertiaTensor {
	if s.kind == CompoundShape {
		return tensorScale(s.compound.inertia, mass)
	}
	i := inertiaFor(s, mass)
	return InertiaTensor{{i.X, 0, 0}, {0, i.Y, 0}, {0, 0, i.Z}}
}
func (b *body) updateInertia() {
	b.state.InertiaTensor = tensorFor(b.state.Collider.Shape, b.state.Mass)
	b.inverseTensor, _ = tensorInverse(b.state.InertiaTensor)
	b.state.Inertia = vec{X: b.state.InertiaTensor[0][0], Y: b.state.InertiaTensor[1][1], Z: b.state.InertiaTensor[2][2]}
}
