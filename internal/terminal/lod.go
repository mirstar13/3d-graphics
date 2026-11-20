package terminal

import "github.com/mirstar13/go-3d-graphics/internal/engine"

// renderLODTransition uses complementary screen-door coverage for fade modes.
// Each cell belongs to one mesh, so overlapping surfaces cannot fight for depth.
// Morph mode falls back to this fade when vertex correspondence is unavailable.
func (r *TerminalRenderer) renderLODTransition(group *engine.LODGroupWithTransitions, matrix engine.Matrix4x4, camera *engine.Camera) {
	from, to, alpha := group.GetBlendedMesh()
	if from == nil {
		return
	}
	if to == nil {
		r.RenderMesh(from, matrix, camera)
		return
	}
	if group.TransitionState.Mode == engine.LODTransitionMorph && compatibleTopology(from, to) {
		mesh := *from
		mesh.Vertices = make([]engine.Point, len(from.Vertices))
		for i, p := range from.Vertices {
			q := to.Vertices[i]
			mesh.Vertices[i] = engine.Point{X: p.X + (q.X-p.X)*alpha, Y: p.Y + (q.Y-p.Y)*alpha, Z: p.Z + (q.Z-p.Z)*alpha}
		}
		mesh.Position = engine.Point{X: from.Position.X + (to.Position.X-from.Position.X)*alpha, Y: from.Position.Y + (to.Position.Y-from.Position.Y)*alpha, Z: from.Position.Z + (to.Position.Z-from.Position.Z)*alpha}
		r.RenderMesh(&mesh, matrix, camera)
		return
	}
	local := *r
	local.lodMask = true
	local.lodAlpha = alpha
	local.lodIncoming = false
	local.RenderMesh(from, matrix, camera)
	local.lodIncoming = true
	local.RenderMesh(to, matrix, camera)
}

func compatibleTopology(a, b *engine.Mesh) bool {
	if len(a.Vertices) != len(b.Vertices) || len(a.Indices) != len(b.Indices) {
		return false
	}
	for i, index := range a.Indices {
		if index != b.Indices[i] {
			return false
		}
	}
	return true
}
