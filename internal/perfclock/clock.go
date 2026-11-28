// Package perfclock supplies a process-local clock for short benchmark spans.
package perfclock

import "time"

func Since(start time.Time) time.Duration { return Now().Sub(start) }
