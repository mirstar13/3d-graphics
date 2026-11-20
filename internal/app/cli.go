package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"

	"github.com/mirstar13/go-3d-graphics/internal/config"
	"github.com/mirstar13/go-3d-graphics/internal/terminal"
)

type commandOptions struct {
	path           string
	settingsScreen bool
	cpuProfile     string
	memoryProfile  string
}

func readCommand(args []string, output io.Writer) (config.Settings, commandOptions, error) {
	settings := config.Default()
	options := commandOptions{}
	flags := flag.NewFlagSet("3d-graphics", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&options.path, "config", "renderer.yaml", "YAML settings file")
	flags.BoolVar(&options.settingsScreen, "settings", false, "open the settings editor")
	flags.StringVar(&options.cpuProfile, "cpuprofile", "", "write CPU profile to file")
	flags.StringVar(&options.memoryProfile, "memprofile", "", "write memory profile to file")
	flags.StringVar(&settings.Renderer, "renderer", settings.Renderer, "registered rendering backend")
	flags.IntVar(&settings.Demo, "demo", settings.Demo, "demo number (1-13)")
	flags.StringVar(&settings.RenderMode, "mode", settings.RenderMode, "single, tiles, jobs, or batched")
	flags.StringVar(&settings.AntiAliasing, "aa", settings.AntiAliasing, "none, fxaa, msaa2, msaa4, or ssaa")
	flags.IntVar(&settings.Width, "width", settings.Width, "viewport columns (0 fits terminal)")
	flags.IntVar(&settings.Height, "height", settings.Height, "viewport rows (0 fits terminal)")
	flags.Float64Var(&settings.FPS, "fps", settings.FPS, "target frames per second")
	flags.IntVar(&settings.Workers, "workers", settings.Workers, "CPU rendering workers")
	flags.IntVar(&settings.TileSize, "tile-size", settings.TileSize, "CPU tile size")
	flags.StringVar(&settings.Color, "color", settings.Color, "auto, none, 16, 256, or truecolor")
	flags.Float64Var(&settings.FOVX, "fov-x", settings.FOVX, "horizontal camera FOV value")
	flags.Float64Var(&settings.FOVY, "fov-y", settings.FOVY, "vertical camera FOV value")
	flags.BoolVar(&settings.AutoOrbit, "auto-orbit", settings.AutoOrbit, "automatically orbit the scene")
	flags.BoolVar(&settings.ShowDebug, "debug", settings.ShowDebug, "show frame and camera status")
	flags.BoolVar(&settings.Profiling, "profiling", settings.Profiling, "collect frame timings")
	if err := flags.Parse(args); err != nil {
		return settings, options, err
	}
	if flags.NArg() > 0 {
		return settings, options, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	explicit := map[string]string{}
	flags.Visit(func(f *flag.Flag) { explicit[f.Name] = f.Value.String() })
	loaded, err := config.Load(options.path)
	if err != nil {
		if _, specified := explicit["config"]; specified || !os.IsNotExist(err) {
			return settings, options, fmt.Errorf("load %s: %w", options.path, err)
		}
		loaded = config.Default()
	}
	settings = loaded
	// Flag values retain pointers to settings fields, so only explicitly supplied
	// flags are applied again after loading the file.
	for name, value := range explicit {
		if err := flags.Set(name, value); err != nil {
			return settings, options, err
		}
	}
	return settings, options, settings.Validate()
}

// Run starts the renderer from YAML settings and explicit command-line overrides.
func Run() {
	settings, options, err := readCommand(os.Args[1:], os.Stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		return
	}
	if options.cpuProfile != "" {
		file, err := os.Create(options.cpuProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer file.Close()
		if err := pprof.StartCPUProfile(file); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer pprof.StopCPUProfile()
	}
	if options.memoryProfile != "" {
		defer func() {
			file, err := os.Create(options.memoryProfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			defer file.Close()
			if err := pprof.WriteHeapProfile(file); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}
	openSettings := options.settingsScreen
	for {
		// Validate terminal access before starting the launcher.
		runtimeConfig := settings.EngineConfig()
		if runtimeConfig.Backend == "terminal" {
			if err := terminal.ConfigureTerminal(&runtimeConfig, settings.Color); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}
		updated, start, err := showStartMenu(settings, options.path, openSettings)
		openSettings = false
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		settings = updated
		if !start {
			return
		}
		for {
			runtimeConfig = settings.EngineConfig()
			if runtimeConfig.Backend == "terminal" {
				if err := terminal.ConfigureTerminal(&runtimeConfig, settings.Color); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
			}
			if !runEngine(settings.Demo-1, runtimeConfig) {
				break
			}
			updated, restart, err := editSettings(settings, options.path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			settings = updated
			if !restart {
				break
			}
		}
	}
}
