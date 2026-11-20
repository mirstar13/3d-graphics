package engine

import "testing"

func TestLODTransitionCompletesWithoutRestart(t *testing.T) {
	group := NewLODGroupWithTransitions(LODTransitionCrossFade, .25)
	group.AddLOD(NewMesh(), 10)
	group.AddLOD(NewMesh(), 100)
	camera := NewCameraAt(0, 0, 0)
	group.UpdateWithTransition(Point{Z: 30}, camera, .1)
	if !group.TransitionState.IsTransitioning {
		t.Fatal("did not start")
	}
	group.UpdateWithTransition(Point{Z: 30}, camera, .25)
	if group.CurrentLOD != 1 || group.TransitionState.IsTransitioning {
		t.Fatal("transition restarted")
	}
	group.UpdateWithTransition(Point{Z: 1}, camera, 0)
	if !group.TransitionState.IsTransitioning || group.TransitionState.ToLOD != 0 {
		t.Fatal("reverse transition missing")
	}
}
func TestLODTransitionZeroDurationAndNone(t *testing.T) {
	for _, mode := range []LODTransitionMode{LODTransitionNone, LODTransitionCrossFade} {
		group := NewLODGroupWithTransitions(mode, 0)
		group.AddLOD(NewMesh(), 10)
		group.AddLOD(NewMesh(), 100)
		group.UpdateWithTransition(Point{Z: 30}, NewCameraAt(0, 0, 0), 0)
		if group.CurrentLOD != 1 || group.TransitionState.IsTransitioning {
			t.Fatal("instant transition failed")
		}
	}
}

func TestTransitionBoundsIncludeBothMeshes(t *testing.T) {
	from, to := NewMesh(), NewMesh()
	from.Vertices = []Point{{X: -5, Z: 10}}
	to.Vertices = []Point{{X: 5, Z: 10}}
	to.Position.X = 2
	group := NewLODGroupWithTransitions(LODTransitionCrossFade, 1)
	group.AddLOD(from, 10)
	group.AddLOD(to, 100)
	group.TransitionState.StartTransition(0, 1)
	node := NewSceneNodeWithObject("lod", group)
	bounds := ComputeNodeBounds(node)
	if bounds == nil || bounds.Min.X != -5 || bounds.Max.X != 7 {
		t.Fatal(bounds)
	}
}
