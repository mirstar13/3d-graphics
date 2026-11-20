package terminal

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func enableTerminalOutput() (func(), error) {
	handle := windows.Handle(os.Stdout.Fd())
	var original uint32
	if err := windows.GetConsoleMode(handle, &original); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, original|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return nil, fmt.Errorf("console requires ANSI virtual terminal support: %w", err)
	}
	return func() { windows.SetConsoleMode(handle, original) }, nil
}
