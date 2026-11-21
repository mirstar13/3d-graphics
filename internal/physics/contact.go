package physics

import (
	"math"
	"sort"
)

const contactSlop = .005

// Contact holds paired surface points. Normal points from A toward B; negative
// separation means penetration. Feature identifies a stable manifold feature.
type Contact struct {
	A, B                   BodyID
	PointA, PointB, Normal vec
	Separation             float64
	Feature                uint64
	Sensor                 bool
}

func contact(a, b *body, pa, pb, n vec, feature uint64) Contact {
	return Contact{A: a.state.ID, B: b.state.ID, PointA: pa, PointB: pb, Normal: n, Separation: dot(sub(pb, pa), n), Feature: feature, Sensor: a.state.Collider.Sensor || b.state.Collider.Sensor}
}
func flipContacts(cs []Contact) []Contact {
	for i := range cs {
		c := &cs[i]
		c.A, c.B = c.B, c.A
		c.PointA, c.PointB = c.PointB, c.PointA
		c.Normal = mul(c.Normal, -1)
	}
	return cs
}
func collide(a, b *body) []Contact {
	ak, bk := a.state.Collider.Shape.kind, b.state.Collider.Shape.kind
	if ak > bk {
		return flipContacts(collide(b, a))
	}
	switch {
	case ak == SphereShape && bk == SphereShape:
		return sphereSphere(a, b)
	case ak == SphereShape && bk == BoxShape:
		return sphereBox(a, b)
	case ak == SphereShape && bk == PlaneShape:
		return spherePlane(a, b)
	case ak == BoxShape && bk == PlaneShape:
		return boxPlane(a, b)
	case ak == BoxShape && bk == BoxShape:
		return boxBox(a, b)
	case bk == MeshShape && (ak == SphereShape || ak == BoxShape):
		return primitiveMesh(a, b)
	}
	return nil
}
func sphereSphere(a, b *body) []Contact {
	delta := sub(b.state.Position, a.state.Position)
	distance := length(delta)
	ra, rb := a.state.Collider.Shape.radius, b.state.Collider.Shape.radius
	if distance > ra+rb+contactSlop {
		return nil
	}
	n := unit(delta)
	return []Contact{contact(a, b, add(a.state.Position, mul(n, ra)), sub(b.state.Position, mul(n, rb)), n, 0)}
}
func planeWorld(b *body) (vec, float64) {
	s := b.state.Collider.Shape
	n := b.state.Rotation.RotateVector(s.normal)
	return n, s.offset + dot(n, b.state.Position)
}
func spherePlane(a, b *body) []Contact {
	n, offset := planeWorld(b)
	d := dot(n, a.state.Position) - offset
	r := a.state.Collider.Shape.radius
	if d-r > contactSlop {
		return nil
	}
	return []Contact{contact(a, b, sub(a.state.Position, mul(n, r)), sub(a.state.Position, mul(n, d)), mul(n, -1), 0)}
}
func sphereBox(a, b *body) []Contact {
	center := inverseRotate(b.state.Rotation, sub(a.state.Position, b.state.Position))
	h := b.state.Collider.Shape.half
	closest := vec{X: math.Max(-h.X, math.Min(h.X, center.X)), Y: math.Max(-h.Y, math.Min(h.Y, center.Y)), Z: math.Max(-h.Z, math.Min(h.Z, center.Z))}
	delta := sub(closest, center)
	distance := length(delta)
	r := a.state.Collider.Shape.radius
	if distance > r+contactSlop {
		return nil
	}
	n := unit(delta)
	if distance < 1e-10 {
		// The sphere centre is inside: choose the nearest exit face. A is pushed
		// opposite the A->B normal, so negate the outward face direction.
		axisIndex := 0
		gap := math.Inf(1)
		sign := 1.0
		for i := 0; i < 3; i++ {
			d := component(h, i) - math.Abs(component(center, i))
			if d < gap {
				gap = d
				axisIndex = i
				sign = 1
				if component(center, i) < 0 {
					sign = -1
				}
			}
		}
		outward := mul(axis(axisIndex), sign)
		closest = add(center, mul(outward, gap))
		n = mul(outward, -1)
	}
	normal := b.state.Rotation.RotateVector(n)
	pointB := add(b.state.Position, b.state.Rotation.RotateVector(closest))
	return []Contact{contact(a, b, add(a.state.Position, mul(normal, r)), pointB, normal, 0)}
}
func boxPlane(a, b *body) []Contact {
	n, offset := planeWorld(b)
	var result []Contact
	for i, p := range boxVertices(a) {
		d := dot(n, p) - offset
		if d <= contactSlop {
			result = append(result, contact(a, b, p, sub(p, mul(n, d)), mul(n, -1), uint64(i)))
		}
	}
	return reduceContacts(result)
}
func projectionRadius(b *body, n vec) float64 {
	axes := boxAxes(b)
	h := b.state.Collider.Shape.half
	return math.Abs(dot(axes[0], n))*h.X + math.Abs(dot(axes[1], n))*h.Y + math.Abs(dot(axes[2], n))*h.Z
}

type clipVertex struct {
	p  vec
	id uint64
}

func clipPolygon(poly []clipVertex, n vec, offset float64, planeID uint64) []clipVertex {
	if len(poly) == 0 {
		return nil
	}
	var out []clipVertex
	previous := poly[len(poly)-1]
	pd := dot(previous.p, n) - offset
	for _, current := range poly {
		cd := dot(current.p, n) - offset
		if (pd <= 1e-9) != (cd <= 1e-9) {
			t := pd / (pd - cd)
			lo, hi := previous.id, current.id
			if lo > hi {
				lo, hi = hi, lo
			}
			out = append(out, clipVertex{p: add(previous.p, mul(sub(current.p, previous.p), t)), id: ((lo*1099511628211)^hi)*31 + planeID})
		}
		if cd <= 1e-9 {
			out = append(out, current)
		}
		previous = current
		pd = cd
	}
	return out
}
func boxFace(b *body, index int, sign float64) []clipVertex {
	axes := boxAxes(b)
	h := b.state.Collider.Shape.half
	center := add(b.state.Position, mul(axes[index], sign*component(h, index)))
	u, v := (index+1)%3, (index+2)%3
	var out []clipVertex
	for k, pair := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		p := add(center, add(mul(axes[u], pair[0]*component(h, u)), mul(axes[v], pair[1]*component(h, v))))
		base := uint64(index*8 + k + 1)
		if sign > 0 {
			base += 4
		}
		out = append(out, clipVertex{p: p, id: base})
	}
	return out
}
func incidentFace(b *body, n vec) []clipVertex {
	axes := boxAxes(b)
	best := 0
	for i := 1; i < 3; i++ {
		if math.Abs(dot(axes[i], n)) > math.Abs(dot(axes[best], n)) {
			best = i
		}
	}
	sign := 1.0
	if dot(axes[best], n) > 0 {
		sign = -1
	}
	return boxFace(b, best, sign)
}
func boxBox(a, b *body) []Contact {
	aa, ba := boxAxes(a), boxAxes(b)
	delta := sub(b.state.Position, a.state.Position)
	bestDepth := math.Inf(1)
	normal := vec{}
	bestI, bestJ, kind := 0, 0, 0
	test := func(n vec, k, i, j int) bool {
		if length(n) < 1e-8 {
			return true
		}
		n = unit(n)
		separation := math.Abs(dot(delta, n)) - projectionRadius(a, n) - projectionRadius(b, n)
		if separation > contactSlop {
			return false
		}
		depth := -separation
		// Prefer face contacts when axes tie; this produces useful resting manifolds.
		if depth < bestDepth-1e-8 {
			bestDepth = depth
			if dot(delta, n) < 0 {
				n = mul(n, -1)
			}
			normal = n
			kind = k
			bestI = i
			bestJ = j
		}
		return true
	}
	for i := 0; i < 3; i++ {
		if !test(aa[i], 0, i, 0) || !test(ba[i], 1, i, 0) {
			return nil
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if !test(cross(aa[i], ba[j]), 2, i, j) {
				return nil
			}
		}
	}
	if kind == 2 {
		a0, a1 := supportEdge(a, bestI, normal)
		b0, b1 := supportEdge(b, bestJ, mul(normal, -1))
		pa, pb := closestSegments(a0, a1, b0, b1)
		return []Contact{contact(a, b, pa, pb, normal, uint64(1000+bestI*3+bestJ))}
	}
	ref, inc := a, b
	n := normal
	if kind == 1 {
		ref, inc = b, a
		n = mul(normal, -1)
	}
	axes := boxAxes(ref)
	h := ref.state.Collider.Shape.half
	sign := 1.0
	if dot(axes[bestI], n) < 0 {
		sign = -1
	}
	faceCenter := add(ref.state.Position, mul(axes[bestI], sign*component(h, bestI)))
	poly := incidentFace(inc, n)
	for j := 0; j < 3; j++ {
		if j == bestI {
			continue
		}
		for _, s := range []float64{-1, 1} {
			side := mul(axes[j], s)
			poly = clipPolygon(poly, side, dot(side, ref.state.Position)+component(h, j), uint64(j*2+3+int(s)))
		}
	}
	var out []Contact
	for _, vertex := range poly {
		separation := dot(sub(vertex.p, faceCenter), n)
		if separation > contactSlop {
			continue
		}
		projected := sub(vertex.p, mul(n, separation))
		c := contact(ref, inc, projected, vertex.p, n, vertex.id^uint64((kind*3+bestI+1)<<48))
		if kind == 1 {
			c = flipContacts([]Contact{c})[0]
		}
		out = append(out, c)
	}
	return reduceContacts(out)
}
func supportEdge(b *body, index int, n vec) (vec, vec) {
	axes := boxAxes(b)
	h := b.state.Collider.Shape.half
	p := b.state.Position
	for j := 0; j < 3; j++ {
		if j == index {
			continue
		}
		s := 1.0
		if dot(axes[j], n) < 0 {
			s = -1
		}
		p = add(p, mul(axes[j], s*component(h, j)))
	}
	d := mul(axes[index], component(h, index))
	return sub(p, d), add(p, d)
}
func closestSegments(p1, q1, p2, q2 vec) (vec, vec) {
	d1, d2, r := sub(q1, p1), sub(q2, p2), sub(p1, p2)
	a, e := dot(d1, d1), dot(d2, d2)
	f := dot(d2, r)
	s, t := 0.0, 0.0
	clamp := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	if a <= 1e-16 && e <= 1e-16 {
		return p1, p2
	}
	if a <= 1e-16 {
		t = clamp(f / e)
	} else {
		c := dot(d1, r)
		if e <= 1e-16 {
			s = clamp(-c / a)
		} else {
			b := dot(d1, d2)
			denom := a*e - b*b
			if math.Abs(denom) > 1e-16 {
				s = clamp((b*f - c*e) / denom)
			}
			t = (b*s + f) / e
			if t < 0 {
				t = 0
				s = clamp(-c / a)
			} else if t > 1 {
				t = 1
				s = clamp((b - c) / a)
			}
		}
	}
	return add(p1, mul(d1, s)), add(p2, mul(d2, t))
}
func reduceContacts(cs []Contact) []Contact {
	// Keep spatially distinct contacts, beginning with the deepest one.
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Separation < cs[j].Separation })
	var out []Contact
	for _, c := range cs {
		duplicate := false
		for _, kept := range out {
			if length(sub(c.PointA, kept.PointA)) < 1e-5 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, c)
		}
	}
	if len(out) <= 4 {
		return out
	}
	selected := []Contact{out[0]}
	out = out[1:]
	for len(selected) < 4 {
		best, bestDistance := 0, -1.0
		for i, c := range out {
			distance := math.Inf(1)
			for _, k := range selected {
				d := sub(c.PointA, k.PointA)
				distance = math.Min(distance, dot(d, d))
			}
			if distance > bestDistance {
				best, bestDistance = i, distance
			}
		}
		selected = append(selected, out[best])
		out = append(out[:best], out[best+1:]...)
	}
	return selected
}
