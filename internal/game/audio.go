package game

import (
	"fmt"
	"maps"
	"slices"

	"github.com/mirstar13/go-3d-graphics/internal/audio"
)

const maxTickSounds = 32

// ConfigureAudio installs the clip catalog for the session, including headless runs.
func (s *Session) ConfigureAudio(clips map[string]*audio.Clip) error {
	for _, e := range s.entities {
		if cue := e.description.PlateSound; cue != "" && clips[cue] == nil {
			return fmt.Errorf("entity %s: unknown audio cue %q", e.description.ID, cue)
		}
	}
	s.sounds = clips
	s.soundCues = slices.Sorted(maps.Keys(clips))
	return nil
}

func (s *Session) AttachAudio(manager *audio.Manager) {
	if manager != nil && len(s.sounds) > 0 {
		s.soundScope = manager.Scope()
	}
}

func (s *Session) SetPaused(paused bool) {
	s.World.SetPaused(paused)
	s.soundScope.Pause(paused)
}

// Close releases the playback scope and the optional behavior worker.
func (s *Session) Close() {
	s.soundScope.Close()
	s.CloseBehaviors()
}

func (s *Session) SetResultSounds(win, loss string) { s.winSound, s.lossSound = win, loss }

func (s *Session) HasSound(cue string) bool { return s.sounds[cue] != nil }

func (s *Session) queueResultSound() {
	cue := s.winSound
	if s.objective.Outcome == Lost {
		cue = s.lossSound
	}
	if cue != "" {
		_ = s.QueueSound(cue)
	}
}

func (s *Session) QueueSound(cue string) error {
	clip := s.sounds[cue]
	if clip == nil {
		return fmt.Errorf("unknown audio cue %q", cue)
	}
	if len(s.pendingSounds) == maxTickSounds {
		return fmt.Errorf("at most %d scripted sounds per tick", maxTickSounds)
	}
	s.pendingSounds = append(s.pendingSounds, clip)
	return nil
}

func (s *Session) DiscardSounds() {
	clear(s.pendingSounds)
	s.pendingSounds = s.pendingSounds[:0]
}

func (s *Session) publishSounds() {
	if s.lastError == nil && s.soundScope != nil {
		for _, clip := range s.pendingSounds {
			s.soundScope.Play(clip)
		}
	}
	s.DiscardSounds()
}
