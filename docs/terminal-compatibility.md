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

## Persistent corruption in VS Code on Windows

Reported environment: Windows 10, PowerShell (`pwsh`), VS Code integrated terminal.
After running scenes for a while, the image becomes corrupted. Restarting the app
in the same terminal does not repair it; creating a new terminal does. The report
is not associated with resizing or changing font size.

The reporter observed no corruption so far after disabling GPU acceleration.
This supports the VS Code GPU-rendering hypothesis, but a longer comparison is
still needed; it is a workaround, not an isolated driver/root-cause diagnosis.

Diagnosis is pending a native reproduction. Persistence across app restarts makes
terminal-session state, VS Code's terminal renderer, and the Windows console/ConPTY
path stronger suspects than body state or framebuffer data owned by one process.
It does not by itself distinguish these causes or prove a GPU/driver fault.

Code inspection found:

- Frames use foreground-color sequences and Unicode full-block characters (U+2588).
  The application does not intentionally change the terminal palette.
- Render workers finish before presentation. Ordinary terminal-mode profiling does
  not print concurrently into the frame.
- Startup enters the alternate screen and clears it, but does not normalize origin
  mode, scrolling margins, or inherited text attributes. Rows use CR/LF progression.
- Write/flush errors are currently ignored. This is an output-handling gap, but
  there is no evidence yet that a write error caused the reported corruption.

A separate experiment with the pyte 0.8.2 terminal emulator reproduced misplaced
rows by setting a restricted scrolling region and origin mode before feeding the
current startup/frame sequences. Replaying startup did not repair those modes.
This demonstrates an initialization weakness, not a reproduction of the Windows
screenshot: the application has not been observed enabling those modes, and the
emulator does not model VS Code GPU rendering or Windows ConPTY.

### Native isolation procedure

Keep the scene, CPU mode, AA, dimensions, and duration the same between runs.

1. In VS Code settings JSON, set `"terminal.integrated.gpuAcceleration": "off"`.
   Create a fresh integrated terminal and run beyond the usual time-to-failure.
   If this consistently prevents the problem, keep it as a workaround and compare
   against GPU acceleration enabled. Record VS Code and graphics-driver versions.
   A single short successful run is not confirmation.
2. Run the same build in standalone Windows Terminal using the same PowerShell.
   Failure only inside VS Code narrows the investigation to its terminal path;
   failure in both requires checking shared Windows console/output behavior too.
3. While an affected terminal still exists, exit the application and run the
   static grid below. Compare it with a fresh terminal. Corruption in this simple
   output rules out scene geometry/physics as necessary to reproduce the symptom.
4. Compare `-color none` and `-color truecolor` with the same scene. Monochrome
   mostly uses ASCII while color rendering uses full blocks, so a difference may
   implicate color handling, Unicode glyph rendering, or output volume. It does
   not prove that the lighting calculation is wrong.

PowerShell static grid (no animation, color changes, or application buffers):

```powershell
'ASCII: ' + ('0123456789' * 6)
1..8 | ForEach-Object { ('{0:D2} ' -f $_) + ('X' * 60) }
'Unicode full blocks:'
1..8 | ForEach-Object { ('{0:D2} ' -f $_) + ([string][char]0x2588 * 60) }
```

Record Windows build (`winver`), VS Code version (Help > About), PowerShell version
(`$PSVersionTable.PSVersion`), active code page (`chcp`), scene/settings, and typical
time to failure. Note whether the menu, shell text, and static grid also corrupt.
Avoid treating `Clear-Host` as a full reset: clearing cells need not reset terminal
modes or repair terminal rendering resources.

Potential application hardening, independently of the eventual root cause: normalize
required terminal modes at session entry, address each frame row explicitly, reset
text attributes before drawing, and surface output failures through the application
lifecycle. Validate these changes with inherited-state and native terminal tests;
do not describe them as a confirmed fix for this report without reproduction.
