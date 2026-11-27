package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"
)

func TestWAVAndMixerLifecycle(t *testing.T) {
	wav := []byte("RIFF\x28\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x44\xac\x00\x00\x88\x58\x01\x00\x02\x00\x10\x00data\x04\x00\x00\x00\x00\x40\x00\x60")
	clip, err := DecodeWAV(wav)
	if err != nil {
		t.Fatal(err)
	}
	if clip.Bytes() != 12 || clip.samples[0] != 16384 || clip.samples[1] != 16384 || clip.samples[2] != 23910 || clip.samples[3] != 23910 {
		t.Fatalf("unexpected resampled stereo PCM: %v", clip.samples)
	}
	if _, err := DecodeWAV(wav[:len(wav)-1]); err == nil {
		t.Fatal("accepted a truncated WAV")
	}
	m := New(nil)
	t.Cleanup(m.Close)
	scope := m.Scope()
	block := make([]byte, 4)
	pull := func(expected int16) {
		t.Helper()
		n, err := m.Read(block)
		if err != nil || n != 4 ||
			int16(binary.LittleEndian.Uint16(block)) != expected ||
			int16(binary.LittleEndian.Uint16(block[2:])) != expected {
			t.Fatalf("got PCM %v, n=%d err=%v, expected %d in both channels", block, n, err, expected)
		}
	}
	pull(0)
	scope.Play(clip)
	scope.Play(clip)
	pull(32767)
	scope.Pause(true)
	scope.Play(clip)
	pull(0)
	scope.Pause(false)
	pull(32767)
	pull(32767)
	pull(0)
	for range Voices + 1 {
		scope.Play(clip)
	}
	if m.Dropped() != 1 {
		t.Fatalf("expected one dropped voice, got %d", m.Dropped())
	}
	current := m.Scope()
	scope.Play(clip)
	scope.Close()
	pull(0)
	current.Play(clip)
	m.ToggleMute()
	pull(0)
	m.ToggleMute()
	pull(23910)
	current.Close()
	pull(0)
}

type failingOutput struct{ err error }

func (o failingOutput) Run(context.Context, io.Reader) error { return o.err }

func TestOutputFailureTransitionsToSilence(t *testing.T) {
	failure := errors.New("disconnected")
	m := New(failingOutput{failure})
	t.Cleanup(m.Close)
	s := m.Scope()
	s.Play(&Clip{samples: []int16{1, 1}})
	select {
	case <-m.Finished():
		if !errors.Is(m.Err(), failure) {
			t.Fatal(m.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("device failure was not reported")
	}
	s.Play(&Clip{samples: []int16{1, 1}})
	block := []byte{1, 1, 1, 1}
	if _, err := m.Read(block); err != nil || binary.LittleEndian.Uint32(block) != 0 {
		t.Fatalf("failed output still had audible samples: %v, %v", block, err)
	}
}

func BenchmarkMix(b *testing.B) {
	for _, count := range []int{0, 1, 8, 32} {
		b.Run(map[int]string{0: "idle", 1: "1_voice", 8: "8_voices", 32: "32_voices"}[count], func(b *testing.B) {
			m := New(nil)
			scope := m.Scope()
			block := make([]byte, BlockFrames*4)
			clip := &Clip{samples: make([]int16, BlockFrames*2)}
			for i := range clip.samples {
				clip.samples[i] = 111
			}
			b.ReportAllocs()
			for b.Loop() {
				for range count {
					scope.Play(clip)
				}
				_, _ = m.Read(block)
			}
		})
	}
}
