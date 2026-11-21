package physics

type contactKey struct {
	a, b    BodyID
	feature uint64
}
type cachedImpulse struct {
	normal, tangent, point vec
	impulse                float64
}

func (w *World) invalidate(id BodyID) {
	for key := range w.cache {
		if key.a == id || key.b == id {
			delete(w.cache, key)
		}
	}
}
