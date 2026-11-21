package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPhysicsNestedDefaultsAndRoundTrip(t *testing.T) {
	s, err := Decode([]byte("physics:\n  gravity:\n    y: -3\n  ccd: true\n  interpolation: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Physics.Gravity.Y != -3 || s.Physics.VelocityIterations != 20 || !s.Physics.Sleeping || !s.Physics.CCD || s.Physics.Interpolation {
		t.Fatal(s.Physics)
	}
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatal(got, err)
	}
	for _, data := range []string{"physics:\n  gravity: {y: .nan}", "physics:\n  unknown: true", "physics:\n  velocity_iterations: 0", "physics:\n  ccd_max_impacts: 0", "physics:\n  sleep_delay: .inf"} {
		if _, err := Decode([]byte(data)); err == nil {
			t.Fatal("invalid config accepted", data)
		}
	}
}
