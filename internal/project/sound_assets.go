package project

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mirstar13/go-3d-graphics/internal/audio"
)

const maxSounds = 128
const maxSoundMemory = 64 << 20

func validCue(cue string) bool {
	if len(cue) == 0 || len(cue) > 64 {
		return false
	}
	for _, c := range cue {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func validateSounds(sounds map[string]string) error {
	if len(sounds) > maxSounds {
		return fmt.Errorf("at most %d audio cues are allowed", maxSounds)
	}
	for cue, name := range sounds {
		if !validCue(cue) || !validAsset(name, ".wav") {
			return fmt.Errorf("invalid audio cue %q or WAV path %q", cue, name)
		}
	}
	return nil
}

func (p *Project) ValidateSoundReferences(scene Document) error {
	for _, e := range scene.Entities {
		if e.PlateSound != "" && p.Manifest.Sounds[e.PlateSound] == "" {
			return fmt.Errorf("entity %s: unknown audio cue %q", e.ID, e.PlateSound)
		}
	}
	return nil
}

func readSound(file *os.File) ([]byte, *audio.Clip, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > audio.MaxWAVBytes {
		return nil, nil, fmt.Errorf("audio requires a regular WAV file no larger than %d MiB", audio.MaxWAVBytes>>20)
	}
	data, err := io.ReadAll(io.LimitReader(file, audio.MaxWAVBytes+1))
	if err != nil {
		return nil, nil, err
	}
	clip, err := audio.DecodeWAV(data)
	return data, clip, err
}

func (p *Project) LoadSound(cue string) (*audio.Clip, error) {
	name := p.Manifest.Sounds[cue]
	if !validAsset(name, ".wav") {
		return nil, fmt.Errorf("unknown audio cue %q", cue)
	}
	clip, err := p.readSoundAsset(name)
	if err != nil {
		return nil, fmt.Errorf("audio cue %q (%s): %w", cue, name, err)
	}
	return clip, nil
}

func (p *Project) readSoundAsset(name string) (*audio.Clip, error) {
	root, err := os.OpenRoot(p.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("audio asset must be a regular WAV file")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	_, clip, err := readSound(file)
	return clip, err
}

// LoadSounds loads each unique asset once with a shared decoded-memory budget.
func (p *Project) LoadSounds() (map[string]*audio.Clip, error) {
	if err := validateSounds(p.Manifest.Sounds); err != nil {
		return nil, err
	}
	if len(p.Manifest.Sounds) == 0 {
		return nil, nil
	}
	result := make(map[string]*audio.Clip, len(p.Manifest.Sounds))
	cache := make(map[string]*audio.Clip)
	budget := maxSoundMemory
	for _, cue := range slices.Sorted(maps.Keys(p.Manifest.Sounds)) {
		name := p.Manifest.Sounds[cue]
		clip := cache[name]
		if clip == nil {
			var err error
			clip, err = p.LoadSound(cue)
			if err != nil {
				return nil, err
			}
			budget -= clip.Bytes()
			if budget < 0 {
				return nil, fmt.Errorf("decoded audio exceeds %d MiB", maxSoundMemory>>20)
			}
			cache[name] = clip
		}
		result[cue] = clip
	}
	return result, nil
}

// ImportSound copies a validated WAV into assets and registers its filename stem as a cue.
func (p *Project) ImportSound(source string) (string, error) {
	fileName := filepath.Base(source)
	cue := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	name := path.Join("assets", fileName)
	if !validCue(cue) || !validAsset(name, ".wav") {
		return "", fmt.Errorf("choose a WAV filename using letters, digits, underscores or hyphens")
	}
	next := p.Manifest
	next.Sounds = maps.Clone(p.Manifest.Sounds)
	if next.Sounds == nil {
		next.Sounds = make(map[string]string)
	}
	if next.Sounds[cue] != "" {
		return "", fmt.Errorf("audio cue %q already exists", cue)
	}
	next.Sounds[cue] = name
	if err := validateSounds(next.Sounds); err != nil {
		return "", err
	}
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("audio import requires a regular WAV file")
	}
	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, _, err := readSound(file)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(p.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeImportedAsset(root, name, data); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(p.Root, "project.yaml"), next); err != nil {
		_ = root.Remove(name)
		return "", err
	}
	p.Manifest = next
	return cue, nil
}
