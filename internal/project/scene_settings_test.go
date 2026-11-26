package project

import "testing"

func TestSettingsSnapshotIsolation(t *testing.T) {
	g := 0.
	off := false
	d := NewDocument()
	d.Simulation = &SimulationSettings{GravityY: &g, Sleeping: &off}
	l := DefaultLighting()
	d.Lighting = &l
	copy := d.Clone()
	*copy.Simulation.GravityY = 10
	copy.Lighting.Lights[0].Name = "Other"
	if g != 0 || l.Lights[0].Name == "Other" {
		t.Fatal("shared mutable settings")
	}
	r := Resource{Version: Version, ID: "a", Scene: d}
	r.Scene.Entities = r.Scene.Entities[:1]
	if r.Validate() == nil {
		t.Fatal("resource accepted global settings")
	}
}
