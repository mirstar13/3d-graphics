package perfclock

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var kernel = windows.NewLazySystemDLL("kernel32.dll")
var counter = kernel.NewProc("QueryPerformanceCounter")
var frequency = kernel.NewProc("QueryPerformanceFrequency")

func read(proc *windows.LazyProc) int64 {
	var value int64
	if ok, _, err := proc.Call(uintptr(unsafe.Pointer(&value))); ok == 0 {
		panic(err)
	}
	return value
}

var origin = sync.OnceValues(func() (int64, int64) {
	return read(counter), read(frequency)
})

// Now uses QPC on Windows, including systems with a coarse Go monotonic clock.
// Convert the remainder separately to avoid overflow on long-running hosts.
func Now() time.Time {
	start, hz := origin()
	ticks := read(counter) - start
	return time.Unix(ticks/hz, ticks%hz*1e9/hz)
}
