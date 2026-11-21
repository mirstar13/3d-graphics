package config

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/mirstar13/go-3d-graphics/internal/demo"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"gopkg.in/yaml.v3"
)

// RenderModes lists the CPU strategies in engine.RenderMode order.
var RenderModes = []string{"single", "tiles", "jobs", "batched"}

// AAModes lists the supported filters in engine.AAMode order.
var AAModes = []string{"none", "fxaa", "msaa2", "msaa4", "ssaa"}

// ColorModes lists accepted terminal color capabilities.
var ColorModes = []string{"auto", "none", "16", "256", "truecolor"}

// Settings is the persisted configuration. Zero viewport dimensions mean automatic sizing.
type Settings struct {
	Physics      PhysicsSettings `yaml:"physics"`
	Demo         int             `yaml:"demo"`
	Renderer     string          `yaml:"renderer"`
	RenderMode   string          `yaml:"render_mode"`
	AntiAliasing string          `yaml:"anti_aliasing"`
	Workers      int             `yaml:"workers"`
	TileSize     int             `yaml:"tile_size"`
	Width        int             `yaml:"width"`
	Height       int             `yaml:"height"`
	FPS          float64         `yaml:"fps"`
	Color        string          `yaml:"color"`
	ShowDebug    bool            `yaml:"show_debug"`
	Profiling    bool            `yaml:"profiling"`
	FOVX         float64         `yaml:"fov_x"`
	FOVY         float64         `yaml:"fov_y"`
	AutoOrbit    bool            `yaml:"auto_orbit"`
}

// Default returns a complete configuration using the available CPU parallelism.
func Default() Settings {
	return Settings{Physics: defaultPhysics(),
		Demo: 1, Renderer: "terminal", RenderMode: "single", AntiAliasing: "none",
		Workers: runtime.GOMAXPROCS(0), TileSize: 32, FPS: 30, Color: "auto",
		ShowDebug: true, Profiling: true, FOVX: 60, FOVY: 30, AutoOrbit: true,
	}
}

// Decode overlays one YAML document on defaults and validates the result.
func Decode(data []byte) (Settings, error) {
	settings := Default()
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&settings); err != nil && err != io.EOF {
		return settings, fmt.Errorf("decode settings: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return settings, fmt.Errorf("settings must contain exactly one YAML document")
	}
	return settings, settings.Validate()
}

// Load reads settings from path, reporting file and validation errors.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, err
	}
	return Decode(data)
}

// Validate checks every setting used to size buffers or select a renderer strategy.
func (s Settings) Validate() error {
	switch {
	case s.Demo < 1 || s.Demo > len(demo.Presets):
		return fmt.Errorf("demo must be between 1 and %d; select a scene from New scene", len(demo.Presets))
	case s.Renderer == "":
		return fmt.Errorf("renderer must have a name")
	case !slices.Contains(RenderModes, s.RenderMode):
		return fmt.Errorf("render_mode must be single, tiles, jobs, or batched")
	case !slices.Contains(AAModes, s.AntiAliasing):
		return fmt.Errorf("anti_aliasing must be none, fxaa, msaa2, msaa4, or ssaa")
	case !slices.Contains(ColorModes, s.Color):
		return fmt.Errorf("color must be auto, none, 16, 256, or truecolor")
	case s.Workers < 1 || s.Workers > 1024:
		return fmt.Errorf("workers must be between 1 and 1024")
	case s.TileSize < 1 || s.TileSize > 1024:
		return fmt.Errorf("tile_size must be between 1 and 1024")
	case s.Width < 0 || s.Width > 4096 || s.Height < 0 || s.Height > 4096:
		return fmt.Errorf("width and height must be between 0 (automatic) and 4096")
	case !finiteRange(s.FPS, 1, 240):
		return fmt.Errorf("fps must be between 1 and 240")
	case !finiteRange(s.FOVX, 1, 179) || !finiteRange(s.FOVY, 1, 179):
		return fmt.Errorf("fov_x and fov_y must be between 1 and 179")
	}
	return s.Physics.validate()
}

func finiteRange(value, min, max float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= min && value <= max
}

// EngineConfig converts validated settings into runtime configuration.
func (s Settings) EngineConfig() engine.EngineConfig {
	return engine.EngineConfig{
		Backend: s.Renderer, RenderMode: engine.RenderMode(slices.Index(RenderModes, s.RenderMode)),
		AAMode: engine.AAMode(slices.Index(AAModes, s.AntiAliasing)),
		Width:  s.Width, Height: s.Height, FPS: s.FPS, NumWorkers: s.Workers,
		TileSize: s.TileSize, ColorMode: s.Color, UseColor: s.Color != "none",
		ShowDebugInfo: s.ShowDebug, EnableProfiling: s.Profiling,
		FOVX: s.FOVX, FOVY: s.FOVY, AutoOrbit: s.AutoOrbit,
	}
}

// Save replaces the file only after a complete, validated write.
func Save(path string, settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".renderer-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
