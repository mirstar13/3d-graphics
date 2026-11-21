package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSettingsRoundTripAndDefaults(t *testing.T) {
	settings, err := Decode([]byte("render_mode: tiles\nanti_aliasing: fxaa\nfov_x: 75\n"))
	if err != nil {
		t.Fatal(err)
	}
	if settings.RenderMode != "tiles" || settings.AntiAliasing != "fxaa" || settings.FOVX != 75 || settings.FPS != 30 {
		t.Fatalf("incorrect defaults/values: %+v", settings)
	}
	path := filepath.Join(t.TempDir(), "renderer.yaml")
	if err := Save(path, settings); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, settings) {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	before, _ := os.ReadFile(path)
	settings.Workers = 0
	if Save(path, settings) == nil {
		t.Fatal("invalid settings saved")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save modified the file")
	}
}

func TestInvalidSettings(t *testing.T) {
	for _, data := range []string{
		"render_mod: tiles", "render_mode: invalid", "anti_aliasing: invalid",
		"workers: 0", "tile_size: 0", "fps: .nan", "fps: .inf",
		"fov_y: 180", "width: -1", "demo: 14", "color: orange",
		"fps: 30\nfps: 40", "fps: 30\n---\nfps: 40",
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := Decode([]byte(data)); err == nil {
				t.Fatal("invalid YAML accepted")
			}
		})
	}
}

func TestSceneCatalogValidation(t *testing.T) {
	settings := Default()
	settings.Demo = 7
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	settings.Demo = 8
	if settings.Validate() == nil {
		t.Fatal("retired scene ID accepted")
	}
}
