package app

import (
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"
)

const frameTimingWindow = 600

type frameTiming struct {
	interval, update, raster, present, write time.Duration
	ticks                                    uint64
	bytes, columns, rows                     int
}

func writeFrameTrace(path string, frames []frameTiming) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	out := csv.NewWriter(file)
	_ = out.Write([]string{"frame", "interval_ns", "update_ns", "raster_ns", "present_ns", "write_ns", "ticks", "frame_bytes", "columns", "rows"})
	for i, f := range frames {
		_ = out.Write([]string{
			strconv.Itoa(i + 1), strconv.FormatInt(int64(f.interval), 10),
			strconv.FormatInt(int64(f.update), 10), strconv.FormatInt(int64(f.raster), 10),
			strconv.FormatInt(int64(f.present), 10), strconv.FormatInt(int64(f.write), 10), strconv.FormatUint(f.ticks, 10),
			strconv.Itoa(f.bytes), strconv.Itoa(f.columns), strconv.Itoa(f.rows),
		})
	}
	out.Flush()
	closeErr := file.Close()
	if err := out.Error(); err != nil {
		return err
	}
	return closeErr
}

func frameTimingSummary(frames []frameTiming) string {
	intervals := make([]time.Duration, len(frames))
	for i, f := range frames {
		intervals[i] = f.interval
	}
	slices.Sort(intervals)
	p := func(percent int) float64 {
		return float64(intervals[(len(intervals)*percent+99)/100-1]) / float64(time.Millisecond)
	}
	return fmt.Sprintf("Frame intervals p50 %.2f ms, p95 %.2f ms, p99 %.2f ms (%d frames)", p(50), p(95), p(99), len(frames))
}
