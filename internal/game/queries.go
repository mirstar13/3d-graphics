package game

import (
	"github.com/mirstar13/go-3d-graphics/internal/physics"
)

// PickHit maps query geometry to a generation-checked entity. VisualProxy denotes
// a visual-only primitive; it is not a physics collider or a rendered triangle hit.
type PickHit struct {
	Entity      EntityID
	FileID      string
	VisualProxy bool
	physics.QueryHit
}

// Pick selects current authoritative geometry. Exact ties prefer physics, then
// physics slot order; visual-only ties follow entity slot order. It never wakes bodies.
func (s *Session) Pick(ray physics.Ray) (PickHit, bool, error) {
	hit, found, err := s.World.Raycast(ray, physics.QueryFilter{LayerMask: ^uint32(0)})
	if err != nil {
		return PickHit{}, false, err
	}
	var best PickHit
	if found {
		best.QueryHit = hit
		for i, e := range s.entities {
			if e.active && e.description.Body != "none" && e.body == hit.Body {
				best = PickHit{Entity: EntityID{s.id, uint32(i), e.generation}, FileID: e.description.ID, QueryHit: hit}
				break
			}
		}
	}
	for i, e := range s.entities {
		if !e.active || e.description.Body != "none" {
			continue
		}
		shape := physics.Box(e.description.Size)
		if e.description.Shape == "sphere" {
			shape = physics.Sphere(e.description.Size.X)
		}
		h, ok, err := physics.RaycastShape(ray, shape, e.node.Transform.GetWorldPosition(), e.node.Transform.GetWorldRotationQuaternion())
		if err != nil {
			return PickHit{}, false, err
		}
		if ok && (!found || h.Distance < best.Distance) {
			best = PickHit{Entity: EntityID{s.id, uint32(i), e.generation}, FileID: e.description.ID, VisualProxy: true, QueryHit: h}
			found = true
		}
	}
	if best.FileID == "" {
		return PickHit{}, false, nil
	}
	return best, found, nil
}

// BodyEntities snapshots the current mapping from physics bodies to live handles.
func (s *Session) BodyEntities() map[physics.BodyID]EntityID {
	ids := make(map[physics.BodyID]EntityID)
	for i, e := range s.entities {
		if e.active && e.description.Body != "none" {
			ids[e.body] = EntityID{s.id, uint32(i), e.generation}
		}
	}
	return ids
}
