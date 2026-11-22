package physics

import "math"

type polyhedron struct {
	vertices []vec
	faces    []hullFace
	edges    [][2]int
}

func bodyPoly(b *body) polyhedron {
	var local polyhedron
	if b.state.Collider.Shape.kind == HullShape {
		h := b.state.Collider.Shape.hull
		local = polyhedron{vertices: h.vertices, faces: h.faces, edges: h.edges}
	} else {
		h := b.state.Collider.Shape.half
		for i := 0; i < 8; i++ {
			p := vec{X: -h.X, Y: -h.Y, Z: -h.Z}
			if i&1 != 0 {
				p.X = h.X
			}
			if i&2 != 0 {
				p.Y = h.Y
			}
			if i&4 != 0 {
				p.Z = h.Z
			}
			local.vertices = append(local.vertices, p)
			for k := 0; k < 3; k++ {
				j := i ^ (1 << k)
				if i < j {
					local.edges = append(local.edges, [2]int{i, j})
				}
			}
		}
		polygons := [][]int{{0, 4, 6, 2}, {1, 3, 7, 5}, {0, 1, 5, 4}, {2, 6, 7, 3}, {0, 2, 3, 1}, {4, 5, 7, 6}}
		for i, ids := range polygons {
			n := axis(i / 2)
			if i%2 == 0 {
				n = mul(n, -1)
			}
			local.faces = append(local.faces, hullFace{normal: n, indices: ids})
		}
	}
	result := polyhedron{edges: local.edges}
	for _, p := range local.vertices {
		result.vertices = append(result.vertices, add(b.state.Position, b.state.Rotation.RotateVector(p)))
	}
	for _, f := range local.faces {
		n := b.state.Rotation.RotateVector(f.normal)
		result.faces = append(result.faces, hullFace{normal: n, offset: dot(n, result.vertices[f.indices[0]]), indices: f.indices})
	}
	return result
}
func projection(p polyhedron, n vec) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range p.vertices {
		d := dot(v, n)
		lo = math.Min(lo, d)
		hi = math.Max(hi, d)
	}
	return lo, hi
}
func supportingEdge(p polyhedron, direction, n vec, maximize bool) (vec, vec, int) {
	best := -math.MaxFloat64
	index := 0
	for i, e := range p.edges {
		a, b := p.vertices[e[0]], p.vertices[e[1]]
		if math.Abs(dot(unit(sub(b, a)), direction)) < .999 {
			continue
		}
		score := dot(mul(add(a, b), .5), n)
		if !maximize {
			score = -score
		}
		if score > best {
			best = score
			index = i
		}
	}
	edge := p.edges[index]
	return p.vertices[edge[0]], p.vertices[edge[1]], index
}
func polyContacts(a, b *body, ap, bp polyhedron) []Contact {
	depth := math.Inf(1)
	var normal vec
	kind := 0
	edgeA, edgeB := 0, 0
	test := func(n vec, k, ea, eb int) bool {
		if length(n) < 1e-8 {
			return true
		}
		n = unit(n)
		alo, ahi := projection(ap, n)
		blo, bhi := projection(bp, n)
		sep := blo - ahi
		if alo-bhi > sep {
			sep = alo - bhi
			n = mul(n, -1)
		}
		if sep > contactSlop {
			return false
		}
		bias := faceAxisBias
		if k == 2 {
			bias = edgeAxisBias
		}
		if -sep < depth-bias {
			depth = -sep
			normal = n
			kind = k
			edgeA = ea
			edgeB = eb
		}
		return true
	}
	for _, f := range ap.faces {
		if !test(f.normal, 0, 0, 0) {
			return nil
		}
	}
	for _, f := range bp.faces {
		if !test(f.normal, 1, 0, 0) {
			return nil
		}
	}
	for i, ae := range ap.edges {
		for j, be := range bp.edges {
			if !test(cross(sub(ap.vertices[ae[1]], ap.vertices[ae[0]]), sub(bp.vertices[be[1]], bp.vertices[be[0]])), 2, i, j) {
				return nil
			}
		}
	}
	if kind == 2 {
		ea, eb := ap.edges[edgeA], bp.edges[edgeB]
		a0, a1, ai := supportingEdge(ap, unit(sub(ap.vertices[ea[1]], ap.vertices[ea[0]])), normal, true)
		b0, b1, bi := supportingEdge(bp, unit(sub(bp.vertices[eb[1]], bp.vertices[eb[0]])), normal, false)
		pa, pb := closestSegments(a0, a1, b0, b1)
		return []Contact{contact(a, b, pa, pb, normal, uint64(1<<60|ai<<16|bi))}
	}
	ref, inc := ap, bp
	refBody, incBody := a, b
	n := normal
	if kind == 1 {
		ref, inc = bp, ap
		refBody, incBody = b, a
		n = mul(n, -1)
	}
	face := 0
	for i := range ref.faces {
		if dot(ref.faces[i].normal, n) > dot(ref.faces[face].normal, n) {
			face = i
		}
	}
	incident := 0
	for i := range inc.faces {
		if dot(inc.faces[i].normal, n) < dot(inc.faces[incident].normal, n) {
			incident = i
		}
	}
	f := ref.faces[face]
	var poly []clipVertex
	for _, index := range inc.faces[incident].indices {
		poly = append(poly, clipVertex{p: inc.vertices[index], id: uint64(index + 1)})
	}
	for i, index := range f.indices {
		start, end := ref.vertices[index], ref.vertices[f.indices[(i+1)%len(f.indices)]]
		side := unit(cross(sub(end, start), n))
		poly = clipPolygon(poly, side, dot(side, start), uint64(i+1))
	}
	var out []Contact
	for _, v := range poly {
		separation := dot(n, v.p) - dot(n, ref.vertices[f.indices[0]])
		if separation > contactSlop {
			continue
		}
		c := contact(refBody, incBody, sub(v.p, mul(n, separation)), v.p, n, v.id^uint64((kind+1)<<56|face<<40|incident<<32))
		if kind == 1 {
			c = flipContacts([]Contact{c})[0]
		}
		out = append(out, c)
	}
	return reduceContacts(out)
}
func hullPlane(a, b *body) []Contact {
	n, offset := planeWorld(b)
	var out []Contact
	for i, p := range bodyPoly(a).vertices {
		d := dot(n, p) - offset
		if d <= contactSlop {
			out = append(out, contact(a, b, p, sub(p, mul(n, d)), mul(n, -1), uint64(i)))
		}
	}
	return reduceContacts(out)
}
func sphereHull(a, b *body) []Contact {
	poly := bodyPoly(b)
	center := a.state.Position
	inside := true
	nearestFace := 0
	bestSeparation := -math.MaxFloat64
	for i, f := range poly.faces {
		d := dot(f.normal, center) - f.offset
		if d > 0 {
			inside = false
		}
		if d > bestSeparation {
			bestSeparation = d
			nearestFace = i
		}
	}
	var closest, n vec
	feature := uint64(nearestFace)
	if inside {
		f := poly.faces[nearestFace]
		closest = sub(center, mul(f.normal, bestSeparation))
		n = mul(f.normal, -1)
	} else {
		best := math.Inf(1)
		for i, f := range poly.faces {
			for j := 1; j+1 < len(f.indices); j++ {
				p := closestTriangle(center, poly.vertices[f.indices[0]], poly.vertices[f.indices[j]], poly.vertices[f.indices[j+1]])
				distance := length(sub(p, center))
				if distance < best {
					best = distance
					closest = p
					feature = uint64(i)
				}
			}
		}
		if best > a.state.Collider.Shape.radius+contactSlop {
			return nil
		}
		n = unit(sub(closest, center))
	}
	return []Contact{contact(a, b, add(center, mul(n, a.state.Collider.Shape.radius)), closest, n, feature)}
}
func hullTriangle(a, b *body, tri [3]vec, feature uint64) []Contact {
	n := unit(cross(sub(tri[1], tri[0]), sub(tri[2], tri[0])))
	poly := polyhedron{vertices: tri[:], edges: [][2]int{{0, 1}, {1, 2}, {2, 0}}, faces: []hullFace{{normal: n, indices: []int{0, 1, 2}}, {normal: mul(n, -1), indices: []int{2, 1, 0}}}}
	cs := polyContacts(a, b, bodyPoly(a), poly)
	for i := range cs {
		cs[i].Feature ^= feature * 0x9e3779b97f4a7c15
	}
	return cs
}
func hullPair(a, b *body) []Contact {
	ak, bk := a.state.Collider.Shape.kind, b.state.Collider.Shape.kind
	switch {
	case ak == SphereShape:
		return sphereHull(a, b)
	case bk == SphereShape:
		return flipContacts(sphereHull(b, a))
	case bk == PlaneShape:
		return hullPlane(a, b)
	case ak == PlaneShape:
		return flipContacts(hullPlane(b, a))
	case bk == MeshShape:
		return primitiveMesh(a, b)
	case ak == MeshShape:
		return flipContacts(primitiveMesh(b, a))
	default:
		return polyContacts(a, b, bodyPoly(a), bodyPoly(b))
	}
}
