//go:build !windows

package perfclock

import "time"

func Now() time.Time { return time.Now() }
