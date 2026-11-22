package physics

import (
	"fmt"
	"math"
	"sort"
)

type hullFace struct {
	normal  vec
	offset  float64
	indices []int
}
type convexHull struct {
	vertices []vec
	faces    []hullFace
	edges    [][2]int
	center   vec
	inertia  InertiaTensor
}

// ConvexHull copies an outward-wound, closed triangular convex surface (4..64
// vertices, at most 128 triangles), merges coplanar faces, and centres it at its
// uniform-density mass centre. It validates authored hulls; it does not compute a
// convex hull from arbitrary points. Shapes are immutable and use discrete collision.
func ConvexHull(vertices []vec, indices []int) (Shape, error) {
	if len(vertices) < 4 || len(vertices) > 64 || len(indices) < 12 || len(indices)%3 != 0 || len(indices) > 384 {
		return Shape{}, fmt.Errorf("hull requires 4..64 vertices and 4..128 triangles")
	}
	h := &convexHull{vertices: append([]vec(nil), vertices...)}
	bounds := emptyBounds()
	for i, v := range vertices {
		if !finitePoint(v) {
			return Shape{}, fmt.Errorf("nonfinite hull vertex")
		}
		bounds.include(v)
		for _, u := range vertices[:i] {
			if length(sub(v, u)) < 1e-9 {
				return Shape{}, fmt.Errorf("duplicate hull vertices")
			}
		}
	}
	scale := length(sub(bounds.max, bounds.min))
	if !finite(scale) || scale < 1e-6 {
		return Shape{}, fmt.Errorf("degenerate hull extent")
	}
	tolerance := scale * 1e-8
	// Integrate near the geometry to avoid cancellation for translated assets.
	origin := mul(add(bounds.min, bounds.max), .5)
	vertices = append([]vec(nil), vertices...)
	for i := range vertices {
		vertices[i] = sub(vertices[i], origin)
	}
	h.vertices = vertices
	type edgeCount struct{ count, direction int }
	edges := map[[2]int]edgeCount{}
	used := make([]bool, len(vertices))
	var volume float64
	var first vec
	var second InertiaTensor
	for t := 0; t < len(indices); t += 3 {
		ids := indices[t : t+3]
		for _, i := range ids {
			if i < 0 || i >= len(vertices) {
				return Shape{}, fmt.Errorf("hull index out of bounds")
			}
			used[i] = true
		}
		a, b, c := vertices[ids[0]], vertices[ids[1]], vertices[ids[2]]
		raw := cross(sub(b, a), sub(c, a))
		if length(raw) < scale*scale*1e-10 {
			return Shape{}, fmt.Errorf("degenerate hull face")
		}
		normal := unit(raw)
		offset := dot(normal, a)
		for _, v := range vertices {
			if dot(normal, v)-offset > tolerance {
				return Shape{}, fmt.Errorf("hull is nonconvex or winding is inward")
			}
		}
		for i := 0; i < 3; i++ {
			u, v := ids[i], ids[(i+1)%3]
			direction := 1
			if u > v {
				u, v = v, u
				direction = -1
			}
			key := [2]int{u, v}
			e := edges[key]
			e.count++
			e.direction += direction
			edges[key] = e
		}
		face := -1
		for i, f := range h.faces {
			if dot(f.normal, normal) > 1-1e-10 && math.Abs(f.offset-offset) < tolerance && math.Abs(dot(f.normal, b)-f.offset) < tolerance && math.Abs(dot(f.normal, c)-f.offset) < tolerance {
				face = i
				break
			}
		}
		if face < 0 {
			h.faces = append(h.faces, hullFace{normal: normal, offset: offset})
			face = len(h.faces) - 1
		}
		h.faces[face].indices = append(h.faces[face].indices, ids...)
		v := dot(a, cross(b, c)) / 6
		volume += v
		sum := add(a, add(b, c))
		first = add(first, mul(sum, v/4))
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				second[i][j] += v / 20 * (component(sum, i)*component(sum, j) + component(a, i)*component(a, j) + component(b, i)*component(b, j) + component(c, i)*component(c, j))
			}
		}
	}
	for _, e := range edges {
		if e.count != 2 || e.direction != 0 {
			return Shape{}, fmt.Errorf("hull is not a closed consistently wound manifold")
		}
	}
	for _, yes := range used {
		if !yes {
			return Shape{}, fmt.Errorf("unused hull vertex")
		}
	}
	if !finite(volume) || volume <= scale*scale*scale*1e-10 {
		return Shape{}, fmt.Errorf("hull has zero or inverted volume")
	}
	h.center = mul(first, 1/volume)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			h.inertia[i][j] = -second[i][j]/volume + component(h.center, i)*component(h.center, j)
			if i == j {
				h.inertia[i][j] += (second[0][0]+second[1][1]+second[2][2])/volume - dot(h.center, h.center)
			}
		}
	}
	if !finitePoint(h.center) {
		return Shape{}, fmt.Errorf("invalid hull mass centre")
	}
	if _, ok := tensorInverse(h.inertia); !ok {
		return Shape{}, fmt.Errorf("invalid hull inertia")
	}
	for i := range h.vertices {
		h.vertices[i] = sub(h.vertices[i], h.center)
	}
	boundaryEdges := map[[2]int]bool{}
	for i := range h.faces {
		f := &h.faces[i]
		f.offset -= dot(f.normal, h.center)
		f.indices = faceBoundary(h.vertices, f.indices, f.normal, tolerance)
		if len(f.indices) < 3 {
			return Shape{}, fmt.Errorf("degenerate hull face boundary")
		}
		for j, a := range f.indices {
			b := f.indices[(j+1)%len(f.indices)]
			if a > b {
				a, b = b, a
			}
			key := [2]int{a, b}
			if !boundaryEdges[key] {
				boundaryEdges[key] = true
				h.edges = append(h.edges, key)
			}
		}
	}
	h.center = add(h.center, origin)
	return Shape{kind: HullShape, hull: h}, nil
}

// faceBoundary removes triangulation diagonals and coplanar interior vertices.
func faceBoundary(vertices []vec, indices []int, n vec, tolerance float64) []int {
	u, v := perpendicular(n), cross(n, perpendicular(n))
	seen := map[int]bool{}
	var order []int
	for _, i := range indices {
		if !seen[i] {
			seen[i] = true
			order = append(order, i)
		}
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := vertices[order[i]], vertices[order[j]]
		x, y := dot(a, u), dot(b, u)
		if x != y {
			return x < y
		}
		return dot(a, v) < dot(b, v)
	})
	turn := func(a, b, c int) float64 {
		return dot(cross(sub(vertices[b], vertices[a]), sub(vertices[c], vertices[b])), n)
	}
	var lower, upper []int
	for _, i := range order {
		for len(lower) >= 2 && turn(lower[len(lower)-2], lower[len(lower)-1], i) <= tolerance*tolerance {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, i)
	}
	for k := len(order) - 1; k >= 0; k-- {
		i := order[k]
		for len(upper) >= 2 && turn(upper[len(upper)-2], upper[len(upper)-1], i) <= tolerance*tolerance {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, i)
	}
	return append(lower[:len(lower)-1], upper[:len(upper)-1]...)
}
