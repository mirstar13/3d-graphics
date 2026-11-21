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
	w.wakeConnected(id, w.contacts)
	for _, slot := range w.joints {
		j := slot.joint
		if j != nil && (j.state.BodyA == id || j.state.BodyB == id) {
			j.impulses = [7]float64{}
		}
	}
	for key := range w.cache {
		if key.a == id || key.b == id {
			delete(w.cache, key)
		}
	}
}
