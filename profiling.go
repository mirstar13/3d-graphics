package main

import (
	"fmt"
	"time"
)

// ProfilerStats holds aggregated performance statistics
type ProfilerStats struct {
	MinTime   float64 // in seconds
	MaxTime   float64 // in seconds
	AvgTime   float64 // in seconds
	TotalTime float64 // in seconds (average frame time)
}

func (s ProfilerStats) String() string {
	return fmt.Sprintf("Avg: %.4fms | Min: %.4fms | Max: %.4fms",
		s.AvgTime*1000, s.MinTime*1000, s.MaxTime*1000)
}

// Profiler measures frame execution times
type Profiler struct {
	start       time.Time
	frameStart  time.Time
	frameTimes  []time.Duration
	windowSize  int
	frameCount  int
	totalTime   time.Duration
	updateTime  time.Duration
	renderTime  time.Duration
	presentTime time.Duration

	// Phase tracking
	phaseStart time.Time
}

// NewProfiler creates a new profiler
func NewProfiler(windowSize int) *Profiler {
	return &Profiler{
		start:      time.Now(),
		windowSize: windowSize,
		frameTimes: make([]time.Duration, 0, windowSize),
	}
}

// BeginFrame marks the start of a frame
func (p *Profiler) BeginFrame() {
	p.frameStart = time.Now()
}

// EndFrame marks the end of a frame and records the duration
func (p *Profiler) EndFrame() {
	duration := time.Since(p.frameStart)
	p.frameTimes = append(p.frameTimes, duration)
	p.totalTime += duration
	p.frameCount++

	// Keep window size constant
	if len(p.frameTimes) > p.windowSize {
		// Remove oldest
		p.totalTime -= p.frameTimes[0]
		p.frameTimes = p.frameTimes[1:]
	}
}

// GetAverageStats calculates statistics from the current window
func (p *Profiler) GetAverageStats() ProfilerStats {
	if len(p.frameTimes) == 0 {
		return ProfilerStats{}
	}

	var minDt, maxDt time.Duration
	var sumDt time.Duration

	minDt = p.frameTimes[0]
	maxDt = p.frameTimes[0]

	for _, dt := range p.frameTimes {
		if dt < minDt {
			minDt = dt
		}
		if dt > maxDt {
			maxDt = dt
		}
		sumDt += dt
	}

	avg := float64(sumDt.Nanoseconds()) / float64(len(p.frameTimes)) / 1e9 // Convert to seconds

	return ProfilerStats{
		MinTime:   float64(minDt.Nanoseconds()) / 1e9,
		MaxTime:   float64(maxDt.Nanoseconds()) / 1e9,
		AvgTime:   avg,
		TotalTime: avg, // For the main loop, TotalTime usually implies "Frame Time"
	}
}

// Phase measurement helpers
func (p *Profiler) BeginUpdate()  { p.phaseStart = time.Now() }
func (p *Profiler) EndUpdate()    { p.updateTime = time.Since(p.phaseStart) }
func (p *Profiler) BeginRender()  { p.phaseStart = time.Now() }
func (p *Profiler) EndRender()    { p.renderTime = time.Since(p.phaseStart) }
func (p *Profiler) BeginPresent() { p.phaseStart = time.Now() }
func (p *Profiler) EndPresent()   { p.presentTime = time.Since(p.phaseStart) }
