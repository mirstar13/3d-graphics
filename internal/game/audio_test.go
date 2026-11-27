package game

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/audio"
	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

func TestPlateSoundEdgesAndPause(t *testing.T) {
	p, err := project.Open("../../examples/switch-room")
	if err != nil {
		t.Fatal(err)
	}
	clips, err := p.LoadSounds()
	if err != nil {
		t.Fatal(err)
	}
	clip := clips["plate_down"]
	gold := make([]byte, clip.Bytes())
	reference := audio.New(nil)
	reference.Scope().Play(clip)
	_, _ = reference.Read(gold)

	plate := project.NewEntity("plate", "Plate")
	plate.Position, plate.Sensor, plate.Body = engine.Point{}, true, "static"
	plate.PlateSignal, plate.PlateSound = "pressed", "plate_down"
	player := project.NewEntity("player", "Player")
	player.Position = engine.Point{X: 5}
	doc := project.Document{Version: 1, Name: "Audio plate", Entities: []project.Entity{plate, player}}
	options := physics.DefaultOptions()
	options.Gravity = engine.Point{}
	for _, tickCount := range []int{1, 3} {
		s, err := NewSession(doc, options)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ConfigureAudio(clips); err != nil {
			t.Fatal(err)
		}
		manager := audio.New(nil)
		s.AttachAudio(manager)
		id, _ := s.Find(player.ID)
		body, _ := s.lookup(id)
		pull := func(expected []byte) {
			t.Helper()
			block := make([]byte, clip.Bytes())
			_, _ = manager.Read(block)
			if !bytes.Equal(block, expected) {
				t.Fatalf("tick %d (%d ticks/frame) produced unexpected PCM", s.World.Stats().Tick, tickCount)
			}
		}
		advance := func(x float64) {
			t.Helper()
			if err := s.World.Teleport(body.body, engine.Point{X: x}, engine.IdentityQuaternion()); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 3; i += tickCount {
				if err := s.Advance(time.Duration(tickCount) * options.FixedStep); err != nil {
					t.Fatal(err)
				}
			}
		}
		silence := make([]byte, len(gold))
		advance(0)
		pull(gold)
		advance(0)
		pull(silence)
		advance(5)
		s.SetPaused(true)
		if err := s.World.Teleport(body.body, engine.Point{}, engine.IdentityQuaternion()); err != nil {
			t.Fatal(err)
		}
		if err := s.Step(); err != nil {
			t.Fatal(err)
		}
		s.SetPaused(false)
		advance(0)
		pull(silence)
		advance(5)
		advance(0)
		pull(gold)
		s.Close()
		manager.Close()
	}
}

func TestImportSoundIntoProject(t *testing.T) {
	p, err := project.Create(filepath.Join(t.TempDir(), "Game"))
	if err != nil {
		t.Fatal(err)
	}
	path := "../../examples/switch-room/assets/plate_down.wav"
	cue, err := p.ImportSound(path)
	if err != nil || cue != "plate_down" {
		t.Fatalf("imported %q: %v", cue, err)
	}
	if _, err := p.ImportSound(path); err == nil {
		t.Fatal("duplicate import succeeded")
	}
	reopened, err := project.Open(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.LoadSound(cue); err != nil {
		t.Fatal(err)
	}
}
