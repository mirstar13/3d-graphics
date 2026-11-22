package physics

import (
	"fmt"
	"math"
	"sort"
)

// TriangleMesh owns validated, immutable local collision vertices and a triangle BVH.
// It is a two-sided surface, not a filled solid. Only static bodies may use it.
type TriangleMesh struct {
	vertices  []vec
	triangles [][3]int
	root      *triangleNode
}
type triangleNode struct {
	bounds      bounds
	left, right *triangleNode
	indices     []int
}

func NewTriangleMesh(vertices []vec, indices []int) (*TriangleMesh, error) {
	if len(indices) == 0 || len(indices)%3 != 0 {
		return nil, fmt.Errorf("mesh requires complete triangle triples")
	}
	mesh := &TriangleMesh{vertices: append([]vec(nil), vertices...)}
	for _, v := range vertices {
		if !finitePoint(v) {
			return nil, fmt.Errorf("mesh vertex must be finite")
		}
	}
	for i := 0; i < len(indices); i += 3 {
		tri := [3]int{indices[i], indices[i+1], indices[i+2]}
		for _, v := range tri {
			if v < 0 || v >= len(vertices) {
				return nil, fmt.Errorf("mesh index out of range")
			}
		}
		a, b, c := vertices[tri[0]], vertices[tri[1]], vertices[tri[2]]
		if length(cross(sub(b, a), sub(c, a))) < 1e-10 {
			return nil, fmt.Errorf("degenerate collision triangle")
		}
		mesh.triangles = append(mesh.triangles, tri)
	}
	all := make([]int, len(mesh.triangles))
	for i := range all {
		all[i] = i
	}
	mesh.root = mesh.build(all)
	return mesh, nil
}
func (m *TriangleMesh) triangle(index int) [3]vec {
	t := m.triangles[index]
	return [3]vec{m.vertices[t[0]], m.vertices[t[1]], m.vertices[t[2]]}
}
func (m *TriangleMesh) build(indices []int) *triangleNode {
	node := &triangleNode{bounds: emptyBounds()}
	for _, i := range indices {
		for _, p := range m.triangle(i) {
			node.bounds.include(p)
		}
	}
	if len(indices) <= 8 {
		node.indices = append([]int(nil), indices...)
		return node
	}
	size := sub(node.bounds.max, node.bounds.min)
	axisIndex := 0
	for i := 1; i < 3; i++ {
		if component(size, i) > component(size, axisIndex) {
			axisIndex = i
		}
	}
	center := func(i int) float64 {
		tri := m.triangle(i)
		return component(add(add(tri[0], tri[1]), tri[2]), axisIndex)
	}
	sort.SliceStable(indices, func(i, j int) bool { return center(indices[i]) < center(indices[j]) })
	half := len(indices) / 2
	node.left = m.build(indices[:half])
	node.right = m.build(indices[half:])
	return node
}
func (m *TriangleMesh) query(query bounds) []int {
	var result []int
	var visit func(*triangleNode)
	visit = func(node *triangleNode) {
		if node == nil || !node.bounds.overlaps(query) {
			return
		}
		result = append(result, node.indices...)
		visit(node.left)
		visit(node.right)
	}
	visit(m.root)
	sort.Ints(result)
	return result
}
func closestTriangle(p, a, b, c vec) vec {
	ab, ac, ap := sub(b, a), sub(c, a), sub(p, a)
	d1, d2 := dot(ab, ap), dot(ac, ap)
	if d1 <= 0 && d2 <= 0 {
		return a
	}
	bp := sub(p, b)
	d3, d4 := dot(ab, bp), dot(ac, bp)
	if d3 >= 0 && d4 <= d3 {
		return b
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		return add(a, mul(ab, d1/(d1-d3)))
	}
	cp := sub(p, c)
	d5, d6 := dot(ab, cp), dot(ac, cp)
	if d6 >= 0 && d5 <= d6 {
		return c
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		return add(a, mul(ac, d2/(d2-d6)))
	}
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		return add(b, mul(sub(c, b), (d4-d3)/((d4-d3)+(d5-d6))))
	}
	denom := 1 / (va + vb + vc)
	return add(a, add(mul(ab, vb*denom), mul(ac, vc*denom)))
}
func primitiveMesh(a, b *body) []Contact {
	mesh := b.state.Collider.Shape.mesh
	worldBounds := bodyBounds(a).expanded(contactSlop)
	local := emptyBounds()
	for i := 0; i < 8; i++ {
		p := worldBounds.min
		if i&1 != 0 {
			p.X = worldBounds.max.X
		}
		if i&2 != 0 {
			p.Y = worldBounds.max.Y
		}
		if i&4 != 0 {
			p.Z = worldBounds.max.Z
		}
		local.include(inverseRotate(b.state.Rotation, sub(p, b.state.Position)))
	}
	type patch struct {
		normal   vec
		offset   float64
		contacts []Contact
	}
	var patches []patch
	for _, index := range mesh.query(local) {
		tri := mesh.triangle(index)
		for j := range tri {
			tri[j] = add(b.state.Position, b.state.Rotation.RotateVector(tri[j]))
		}
		planeNormal := unit(cross(sub(tri[1], tri[0]), sub(tri[2], tri[0])))
		offset := dot(planeNormal, tri[0])
		patchIndex := -1
		for i, p := range patches {
			alignment := dot(p.normal, planeNormal)
			if math.Abs(alignment) > .999999 && math.Abs(p.offset-offset*math.Copysign(1, alignment)) < 1e-6 {
				patchIndex = i
				break
			}
		}
		if patchIndex < 0 {
			patchIndex = len(patches)
			patches = append(patches, patch{normal: planeNormal, offset: offset})
		}
		group := &patches[patchIndex]
		feature := uint64(index+1) << 32
		if a.state.Collider.Shape.kind == SphereShape {
			p := closestTriangle(a.state.Position, tri[0], tri[1], tri[2])
			delta := sub(p, a.state.Position)
			distance := length(delta)
			radius := a.state.Collider.Shape.radius
			if distance > radius+contactSlop {
				continue
			}
			n := unit(delta)
			if distance < 1e-10 {
				n = mul(planeNormal, -1)
				if dot(n, a.state.Velocity) < 0 {
					n = mul(n, -1)
				}
			}
			group.contacts = append(group.contacts, contact(a, b, add(a.state.Position, mul(n, radius)), p, n, feature))
		} else if a.state.Collider.Shape.kind == HullShape {
			group.contacts = append(group.contacts, hullTriangle(a, b, tri, feature)...)
		} else {
			group.contacts = append(group.contacts, boxTriangle(a, b, tri, feature)...)
		}
	}
	// A coplanar seam is not a physical edge. Prefer face-support contacts over
	// shallower edge contacts from neighboring triangles on the same surface.
	var out []Contact
	for _, p := range patches {
		var faces []Contact
		for _, c := range p.contacts {
			if math.Abs(dot(c.Normal, p.normal)) > .99999 {
				faces = append(faces, c)
			}
		}
		contacts := p.contacts
		if len(faces) > 0 {
			contacts = faces
		}
		contacts = reduceContacts(contacts)
		if a.state.Collider.Shape.kind == SphereShape && len(contacts) > 1 {
			contacts = contacts[:1]
		}
		out = append(out, contacts...)
	}
	return reduceContacts(out)
}
func boxTriangle(a, b *body, tri [3]vec, feature uint64) []Contact {
	axes := boxAxes(a)
	edges := [3]vec{sub(tri[1], tri[0]), sub(tri[2], tri[1]), sub(tri[0], tri[2])}
	normal := unit(cross(edges[0], edges[1]))
	best := math.Inf(1)
	bestAxis := normal
	kind, index, edgeIndex := 0, 0, 0
	test := func(n vec, k, i, j int) bool {
		if length(n) < 1e-8 {
			return true
		}
		n = unit(n)
		center := dot(a.state.Position, n)
		radius := projectionRadius(a, n)
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range tri {
			v := dot(p, n)
			lo = math.Min(lo, v)
			hi = math.Max(hi, v)
		}
		if lo > center+radius+contactSlop || hi < center-radius-contactSlop {
			return false
		}
		positive := center + radius - lo
		negative := hi - center + radius
		depth := positive
		if negative < positive {
			depth = negative
			n = mul(n, -1)
		}
		if depth < best-1e-8 {
			best = depth
			bestAxis = n
			kind = k
			index = i
			edgeIndex = j
		}
		return true
	}
	if !test(normal, 0, 0, 0) {
		return nil
	}
	for i := 0; i < 3; i++ {
		if !test(axes[i], 1, i, 0) {
			return nil
		}
		for j := 0; j < 3; j++ {
			if !test(cross(axes[i], edges[j]), 2, i, j) {
				return nil
			}
		}
	}
	n := bestAxis
	if kind == 2 {
		p, q := supportEdge(a, index, n)
		pa, pb := closestSegments(p, q, tri[edgeIndex], tri[(edgeIndex+1)%3])
		return []Contact{contact(a, b, pa, pb, n, feature+uint64(100+index*3+edgeIndex))}
	}
	var out []Contact
	if kind == 0 {
		poly := incidentFace(a, mul(n, -1))
		for j := 0; j < 3; j++ {
			side := unit(cross(normal, edges[j]))
			if dot(side, sub(tri[(j+2)%3], tri[j])) > 0 {
				side = mul(side, -1)
			}
			poly = clipPolygon(poly, side, dot(side, tri[j]), uint64(j+1))
		}
		for _, v := range poly {
			sep := dot(sub(tri[0], v.p), n)
			if sep <= contactSlop {
				out = append(out, contact(a, b, v.p, add(v.p, mul(n, sep)), n, feature^v.id))
			}
		}
	} else {
		poly := []clipVertex{{p: tri[0], id: 1}, {p: tri[1], id: 2}, {p: tri[2], id: 3}}
		h := a.state.Collider.Shape.half
		for j := 0; j < 3; j++ {
			if j == index {
				continue
			}
			for _, s := range []float64{-1, 1} {
				side := mul(axes[j], s)
				poly = clipPolygon(poly, side, dot(side, a.state.Position)+component(h, j), uint64(j*2+3+int(s)))
			}
		}
		face := add(a.state.Position, mul(n, component(h, index)))
		for _, v := range poly {
			sep := dot(sub(v.p, face), n)
			if sep <= contactSlop {
				out = append(out, contact(a, b, sub(v.p, mul(n, sep)), v.p, n, feature^v.id))
			}
		}
	}
	return reduceContacts(out)
}
