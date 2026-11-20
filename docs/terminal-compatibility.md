# Terminal compatibility checks

Available here: Linux x86-64 PTY. Native Windows and macOS consoles are not
available in this workspace, so their interactive behavior remains unverified.
Cross-builds check compilation only.

Build with `go build -race -o /tmp/graphics .` and run
`python3 scripts/check_terminal.py /tmp/graphics` on Linux or macOS.
The script exercises the menu, settings, scene restart, and prompt exit.

For native testing, run `go run .` in Windows Terminal/PowerShell and macOS
Terminal (optionally iTerm2), then repeat these steps:

1. Open Settings, change AA and CPU mode, apply, and launch a scene.
2. Confirm consecutive rendered rows have no blank lines or scrolling.
3. Shrink and grow the terminal while rendering, including a very small window.
4. Press P, return to the menu, and start another scene.
5. Leave a scene using X, Escape, and Ctrl+C in separate sessions.
6. Exit from the menu. Confirm the cursor is visible, typed text echoes, and
   console colors and input mode are restored. Exit should take under two seconds.
7. Repeat with color modes none, 16, 256, and truecolor where supported.

Record OS/version, terminal/version, dimensions, color mode, and elapsed exit
 time for each result. On Windows, verify ANSI output is enabled successfully
and the original console mode is restored after exit.
