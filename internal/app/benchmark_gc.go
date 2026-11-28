package app

import (
	"runtime/metrics"
)

const (
	gcCycles       = "/gc/cycles/total:gc-cycles"
	gcPauses       = "/sched/pauses/total/gc:seconds"
	heapAllocs     = "/gc/heap/allocs:bytes"
	heapObjects    = "/gc/heap/allocs:objects"
	liveHeap       = "/gc/heap/live:bytes"
	gcCPU          = "/cpu/classes/gc/total:cpu-seconds"
	gcAssist       = "/cpu/classes/gc/mark/assist:cpu-seconds"
	runtimeMemory  = "/memory/classes/total:bytes"
	releasedMemory = "/memory/classes/heap/released:bytes"
)

type benchmarkGC struct {
	Cycles, AllocBytes, AllocObjects, LiveBytes, PeakRuntimeBytes uint64
	CPUSeconds, AssistSeconds                                     float64
	PauseCounts                                                   []uint64
	PauseBoundsSeconds                                            []float64
}

type gcMeter struct {
	samples []metrics.Sample
	start   benchmarkGC
	peak    uint64
	bounds  []float64
}

func newGCMeter() *gcMeter {
	m := new(gcMeter)
	for _, name := range []string{gcCycles, gcPauses, heapAllocs, heapObjects, liveHeap, gcCPU, gcAssist, runtimeMemory, releasedMemory} {
		m.samples = append(m.samples, metrics.Sample{Name: name})
	}
	return m
}

func (m *gcMeter) memory() uint64 {
	metrics.Read(m.samples)
	return m.observeMemory()
}

func (m *gcMeter) observeMemory() uint64 {
	used := m.samples[7].Value.Uint64() - m.samples[8].Value.Uint64()
	m.peak = max(m.peak, used)
	return used
}

func (m *gcMeter) snapshot() benchmarkGC {
	metrics.Read(m.samples)
	m.observeMemory()
	p := m.samples[1].Value.Float64Histogram()
	if m.bounds == nil {
		// Export internal boundaries only. The end buckets are unbounded.
		m.bounds = append([]float64(nil), p.Buckets[1:len(p.Buckets)-1]...)
	}
	return benchmarkGC{Cycles: m.samples[0].Value.Uint64(), AllocBytes: m.samples[2].Value.Uint64(),
		AllocObjects: m.samples[3].Value.Uint64(), LiveBytes: m.samples[4].Value.Uint64(),
		CPUSeconds: m.samples[5].Value.Float64(), AssistSeconds: m.samples[6].Value.Float64(),
		PauseCounts: append([]uint64(nil), p.Counts...),
	}
}

func (m *gcMeter) begin() {
	m.peak = 0
	m.start = m.snapshot()
}

func (m *gcMeter) end() benchmarkGC {
	after := m.snapshot()
	after.Cycles -= m.start.Cycles
	after.AllocBytes -= m.start.AllocBytes
	after.AllocObjects -= m.start.AllocObjects
	after.CPUSeconds -= m.start.CPUSeconds
	after.AssistSeconds -= m.start.AssistSeconds
	for i := range after.PauseCounts {
		after.PauseCounts[i] -= m.start.PauseCounts[i]
	}
	after.PauseBoundsSeconds = m.bounds
	after.PeakRuntimeBytes = m.peak
	return after
}
