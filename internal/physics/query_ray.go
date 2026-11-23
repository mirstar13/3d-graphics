package physics

import "math"

func rayLocal(r Ray, s Shape) (float64, vec, bool) {
	switch s.kind {
	case SphereShape:
		c := dot(r.Origin, r.Origin) - s.radius*s.radius
		if c <= 0 {
			return 0, mul(r.Direction, -1), true
		}
		b := dot(r.Origin, r.Direction)
		disc := b*b - c
		if disc < 0 {
			return 0, vec{}, false
		}
		t := -b - math.Sqrt(disc)
		if t < 0 || t > r.MaxDistance || !finite(t) {
			return 0, vec{}, false
		}
		return t, unit(add(r.Origin, mul(r.Direction, t))), true
	case BoxShape:
		faces := []hullFace{{normal: vec{X: 1}, offset: s.half.X}, {normal: vec{X: -1}, offset: s.half.X}, {normal: vec{Y: 1}, offset: s.half.Y}, {normal: vec{Y: -1}, offset: s.half.Y}, {normal: vec{Z: 1}, offset: s.half.Z}, {normal: vec{Z: -1}, offset: s.half.Z}}
		return rayConvex(r, faces)
	case HullShape:
		return rayConvex(r, s.hull.faces)
	case PlaneShape:
		n := unit(s.normal)
		d := dot(n, r.Origin) - s.offset/length(s.normal)
		if d <= 0 {
			return 0, mul(r.Direction, -1), true
		}
		denominator := dot(n, r.Direction)
		if denominator >= 0 {
			return 0, vec{}, false
		}
		t := -d / denominator
		return t, n, t <= r.MaxDistance
	case MeshShape:
		// Query the segment AABB against the existing immutable triangle BVH.
		box := emptyBounds()
		box.include(r.Origin)
		box.include(add(r.Origin, mul(r.Direction, r.MaxDistance)))
		best := r.MaxDistance
		normal := vec{}
		found := false
		for _, index := range s.mesh.query(box) {
			t, n, ok := rayTriangle(r, s.mesh.triangle(index))
			if ok && (!found || t < best) {
				best = t
				normal = n
				found = true
			}
		}
		return best, normal, found
	}
	return 0, vec{}, false
}
func rayConvex(r Ray, faces []hullFace) (float64, vec, bool) {
	enter, exit := 0.0, r.MaxDistance
	normal := mul(r.Direction, -1)
	inside := true
	for _, face := range faces {
		d := dot(face.normal, r.Origin) - face.offset
		denom := dot(face.normal, r.Direction)
		if d > 0 {
			inside = false
		}
		if math.Abs(denom) < 1e-14 {
			if d > 0 {
				return 0, vec{}, false
			}
			continue
		}
		t := -d / denom
		if denom < 0 {
			if t > enter {
				enter = t
				normal = face.normal
			}
		} else {
			exit = math.Min(exit, t)
		}
		if enter > exit {
			return 0, vec{}, false
		}
	}
	if inside {
		return 0, mul(r.Direction, -1), true
	}
	return enter, normal, enter >= 0 && enter <= r.MaxDistance
}
func rayTriangle(r Ray, tri [3]vec) (float64, vec, bool) {
	a, b := sub(tri[1], tri[0]), sub(tri[2], tri[0])
	p := cross(r.Direction, b)
	det := dot(a, p)
	if math.Abs(det) < 1e-12 {
		return 0, vec{}, false
	}
	inverse := 1 / det
	offset := sub(r.Origin, tri[0])
	u := dot(offset, p) * inverse
	if u < 0 || u > 1 {
		return 0, vec{}, false
	}
	q := cross(offset, a)
	v := dot(r.Direction, q) * inverse
	if v < 0 || u+v > 1 {
		return 0, vec{}, false
	}
	t := dot(b, q) * inverse
	if t < 0 || t > r.MaxDistance {
		return 0, vec{}, false
	}
	normal := unit(cross(a, b))
	if dot(normal, r.Direction) > 0 {
		normal = mul(normal, -1)
	}
	return t, normal, true
}
