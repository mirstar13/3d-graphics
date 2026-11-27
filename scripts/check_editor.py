"""Project editor PTY workflow; run after go build -race -o /tmp/graphics ."""
import glob, shutil, subprocess, zipfile
import os, pty, fcntl, termios, struct, select, time, signal, sys, tempfile

BINARY = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "/tmp/graphics")

class Session:
    def __init__(self, args):
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            os.environ.update(TERM="xterm-256color", COLORTERM="truecolor")
            fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", 35, 110, 0, 0))
            os.execv(BINARY, [BINARY] + args)
        self.output = b""
        self.closed = False
    def wait(self, needle, seconds=12):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            if needle in self.output:
                return
            if select.select([self.fd], [], [], .05)[0]:
                try:
                    data = os.read(self.fd, 65536)
                except OSError:
                    break
                self.output += data
                if b"\x1b[6n" in data:
                    os.write(self.fd, b"\x1b[1;1R")
        raise AssertionError((needle, self.output[-4000:]))
    def send(self, data):
        self.output = b""
        os.write(self.fd, data)
    def close(self):
        end = time.monotonic() + 8
        while time.monotonic() < end:
            if select.select([self.fd], [], [], .05)[0]:
                try:
                    self.output += os.read(self.fd, 65536)
                except OSError:
                    pass
            pid, status = os.waitpid(self.pid, os.WNOHANG)
            if pid:
                assert os.waitstatus_to_exitcode(status) == 0, self.output[-4000:]
                assert b"DATA RACE" not in self.output
                flags = termios.tcgetattr(self.fd)[3]
                assert flags & termios.ECHO and flags & termios.ICANON
                os.close(self.fd)
                self.closed = True
                return
        raise AssertionError("editor exit timed out")
    def cleanup(self):
        if not self.closed:
            try:
                os.kill(self.pid, signal.SIGKILL)
                os.waitpid(self.pid, 0)
            except ProcessLookupError:
                pass
            os.close(self.fd)

with tempfile.TemporaryDirectory() as parent:
    root = os.path.join(parent, "Example")
    s = Session(["-editor"])
    try:
        s.wait(b"Project editor")
        s.send(b"n"); s.wait(b"new project directory")
        s.send(root.encode() + b"\r"); s.wait(b"Opened Example")
        s.send(b"\x02"); s.wait(b"Built build/game", seconds=90)
        source_path = os.path.join(root, "main.go")
        with open(source_path, "rb") as file:
            starter = file.read()
        with open(source_path, "ab") as file:
            file.write(b"\nvar _ = missingGameplaySymbol\n")
        s.send(b"\x02"); s.wait(b"undefined: missingGameplaySymbol", seconds=30)
        with open(source_path, "wb") as file:
            file.write(starter)
        s.send(b"\x02"); s.wait(b"Built build/game", seconds=30)
        s.send(b"\x1b"); s.wait(b"Hierarchy")
        s.send(b"\x1b[17~"); s.wait(b"Game exited successfully")
        s.send(b"\x1b"); s.wait(b"Hierarchy")
        # Keep an unsaved entity through a crash, a running restart and a stop.
        s.send(b"g"); s.wait(b"Added sphere")
        with open(source_path, "w") as file:
            file.write("""package main
import ("fmt"; "os"; "time")
func main() {
    data, _ := os.ReadFile("starts")
    if err := os.WriteFile("starts", append(data, 'x'), 0600); err != nil { panic(err) }
    fmt.Println("project game started")
    if len(data) == 0 { panic("sample behavior failure") }
    for { time.Sleep(time.Second) }
}
""")
        s.send(b"\x02"); s.wait(b"Built build/game", seconds=30)
        s.send(b"\x1b[17~"); s.wait(b"sample behavior failure")
        starts = os.path.join(root, "starts")
        def wait_starts(count):
            end = time.monotonic() + 8
            while time.monotonic() < end:
                if os.path.getsize(starts) == count:
                    return
                time.sleep(.03)
            raise AssertionError("game did not restart")
        s.send(b"\x1b[17~"); wait_starts(2)
        s.send(b"\x1b[17~"); wait_starts(3)
        s.send(b"\x1b"); s.wait(b"Game stopped")
        s.send(b"\x1b"); s.wait(b"New sphere")
        with open(source_path, "wb") as file:
            file.write(starter)
        s.send(b"e"); s.wait(b"Edit Name")
        s.send(b"\x7f" * len("New sphere") + b"Player\r"); s.wait(b"Property updated")
        s.send(b"\x13"); s.wait(b"Saved scenes/main.yaml")
        scene_path = os.path.join(root, "scenes", "main.yaml")
        with open(scene_path, "rb") as file:
            authored = file.read()
        assert b"name: Player" in authored
        s.send(b"p"); s.wait(b"[PLAY]")
        time.sleep(.2)
        s.send(b"\x1b[17~"); s.wait(b"[PAUSED]")
        s.send(b"i"); s.wait(b"Upward impulse queued")
        s.send(b"n"); s.wait(b"Pending:0")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        with open(scene_path, "rb") as file:
            assert file.read() == authored, "Play changed saved scene"
        # Parent a visual child through the inspector and keyboard picker.
        s.send(b"\x1b[B\x1b[B"); s.wait(b"> Body")
        s.send(b"e"); s.wait(b"Edit Body")
        s.send(b"\x7f" * len("dynamic") + b"none\r"); s.wait(b"Property updated")
        s.send(b"h"); s.wait(b"Parent picker")
        s.send(b"\x1b[B\x1b[B"); s.wait(b"> Box [box]")
        s.send(b"\r"); s.wait(b"Parent updated")
        s.send(b"\x13"); s.wait(b"Saved scenes/main.yaml")
        with open(scene_path, "rb") as file:
            assert b"parent: box" in file.read()
        s.send(b"b"); s.wait(b"Added box")
        s.send(b"q"); s.wait(b"Unsaved edits")
        s.send(b"d"); s.close()
    finally:
        s.cleanup()
    # A different process loads the saved project, not the discarded extra object.
    s = Session(["-project", root])
    try:
        s.wait(b"Player")
        assert b"New box" not in s.output
        fcntl.ioctl(s.fd, termios.TIOCSWINSZ, struct.pack("HHHH", 14, 40, 0, 0))
        s.send(b"\t"); s.wait(b"Inspector")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Example")
        s.send(b"\x1b[B"); s.wait(b"> Box")
        s.send(b"r"); s.wait(b"Resource name")
        s.send(b"\x7f" * 3 + b"Assembly\r"); s.wait(b"Created resource")
        for _ in range(2):
            s.send(b"a"); s.wait(b"Scene resources")
            s.send(b"\x1b[B"); s.wait(b"> Assembly")
            s.send(b"\r"); s.wait(b"Instantiated Assembly")
        s.send(b"e"); s.wait(b"Edit Name")
        s.send(b"\x7f" * 3 + b"Custom\r"); s.wait(b"Property updated")
        s.send(b"\x13"); s.wait(b"Saved scenes/main.yaml")
        s.send(b"a"); s.wait(b"Scene resources")
        s.send(b"\x1b[B"); s.wait(b"> Assembly")
        s.send(b"e"); s.wait(b"Editing source: Assembly")
        s.send(b"e"); s.wait(b"Edit Name")
        s.send(b"\x7f" * 3 + b"SourceBox\r"); s.wait(b"Property updated")
        s.send(b"\x13"); s.wait(b"Saved resources/")
        s.send(b"a"); s.wait(b"Scene resources")
        s.send(b"\r"); s.wait(b"Opened Example")
        s.send(b"\x12"); s.wait(b"Links refreshed")
        s.send(b"\x1b[B" * 5); s.wait(b"> Custom [linked]")
        s.send(b"\x14"); s.wait(b"Property override reset")
        s.send(b"\x1a"); s.wait(b"Undo")
        s.send(b"\x15"); s.wait(b"Instance unpacked")
        s.send(b"\x13"); s.wait(b"Saved scenes/main.yaml")
        s.send(b"p"); s.wait(b"[PLAY]")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        s.send(b"q"); s.close()
        with open(scene_path, "rb") as file:
            saved = file.read()
        assert b"instances:" in saved and b"name: Custom" in saved
    finally:
        s.cleanup()
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Example")
        s.send(b"\x1b[B" * 3); s.wait(b"SourceBox [linked]")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Example")
        s.send(b"k"); s.wait(b"Project key bindings")
        s.send(b"\r"); s.wait(b"Bind Play")
        s.send(b"\x7f" * len("p,f5") + b"v\r"); s.wait(b"Project bindings saved")
        s.send(b"\x1b"); s.wait(b"v Play")
        s.send(b"\t\t"); s.wait(b"focus: Viewport")
        s.send(b"f"); s.wait(b"Picked ")
        s.send(b"0"); s.wait(b"Selection cleared")
        s.send(b"]]"); s.wait(b"> Box")
        s.send(b"v"); s.wait(b"[PLAY]")
        s.send(b"\x1b[17~"); s.wait(b"[PAUSED]")
        s.send(b"i"); s.wait(b"Upward impulse queued")
        s.send(b"n"); s.wait(b"Pending:0")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        fcntl.ioctl(s.fd, termios.TIOCSWINSZ, struct.pack("HHHH", 14, 40, 0, 0))
        s.send(b"\x1b[C"); s.wait(b"Viewport")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    s = Session(["-project", root])
    try:
        s.wait(b"v Play")
        s.send(b"v"); s.wait(b"[PLAY]")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        s.send(b"k"); s.wait(b"Project key bindings")
        s.send(b"d"); s.wait(b"Default bindings saved")
        s.send(b"\x1b"); s.wait(b"p/f5 Play")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Example")
        s.send(b"\x05"); s.wait(b"Scene settings")
        s.send(b"l"); s.wait(b"> Light 2")
        s.send(b"d"); s.wait(b"> Key")
        s.send(b"r"); s.wait(b"> Ambient")
        s.send(b"\x1b"); s.wait(b"Hierarchy")
        with open(scene_path, "rb") as file:
            authored_before_tools = file.read()
        s.send(b"p"); s.wait(b"[PLAY]")
        s.send(b"\x0c"); s.wait(b"Play tools (paused)")
        s.send(b"d"); s.wait(b"authored objects are protected")
        s.send(b"\r"); s.wait(b"Edit Impulse X")
        s.send(b"\x7f2\r"); s.wait(b"Tool value updated")
        s.send(b"g"); s.wait(b"Spawn preview: box")
        s.send(b"\x1b"); s.wait(b"[PAUSED]")
        s.send(b"\x0c"); s.wait(b"Play tools (paused)")
        s.send(b"g"); s.wait(b"Spawn preview: box")
        s.send(b"\r"); s.wait(b"Spawn queued")
        s.send(b"\x1b"); s.wait(b"[PAUSED]")
        s.send(b"n"); s.wait(b"Runtime box 2")
        s.send(b"i"); s.wait(b"Impulse queued for next tick")
        s.send(b"n"); s.wait(b"Pending:0")
        s.send(b"\x0c"); s.wait(b"Play tools (paused)")
        s.send(b"d"); s.wait(b"Runtime deletion queued")
        s.send(b"\x1b"); s.wait(b"[PAUSED]")
        s.send(b"n"); s.wait(b"(no selection)")
        s.send(b"\x0c"); s.wait(b"Play tools (paused)")
        s.send(b"r"); s.wait(b"Play reset; pending commands")
        s.send(b"\x1b"); s.wait(b"Tick:0 Pending:0")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        s.send(b"q"); s.close()
        with open(scene_path, "rb") as file:
            assert file.read() == authored_before_tools, "Play tools modified saved scene"
    finally:
        s.cleanup()
    # The exported executable must use the packaged files, even from another cwd.
    with open(os.path.join(root, "assets", "message.txt"), "w") as file:
        file.write("Packaged asset loaded")
    with open(source_path, "w") as file:
        file.write("""package main
import ("fmt"; "os"; "github.com/mirstar13/go-3d-graphics/creator")
func main() {
    g, err := creator.Open(".", nil); if err != nil { panic(err) }
    if err := g.Step(); err != nil { panic(err) }
    data, err := os.ReadFile("assets/message.txt"); if err != nil { panic(err) }
    fmt.Println(string(data))
}
""")
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Example")
        s.send(b"\x1b[20~"); s.wait(b"Exported build/game-", seconds=90)
        archive, = glob.glob(os.path.join(root, "build", "game-*.zip"))
        with open(archive, "rb") as file:
            successful = file.read()
        with open(source_path, "a") as file:
            file.write("\nvar _ = invalidExportSource\n")
        s.send(b"\x1b[20~"); s.wait(b"invalidExportSource", seconds=30)
        with open(archive, "rb") as file:
            assert file.read() == successful, "failed export replaced the package"
        s.send(b"\x1b"); s.wait(b"Hierarchy")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    release = os.path.join(parent, "Relocated game")
    with zipfile.ZipFile(archive) as package:
        assert "main.go" not in package.namelist() and "go.mod" not in package.namelist()
        package.extractall(release)
        for executable in ("play.sh", "game"):
            mode = package.getinfo(executable).external_attr >> 16
            assert mode & 0o111
            os.chmod(os.path.join(release, executable), mode & 0o777)
    os.rename(root, root + "-unavailable")
    result = subprocess.run([os.path.join(release, "play.sh")], cwd=parent,
                            capture_output=True, check=True, timeout=15)
    assert b"Packaged asset loaded" in result.stdout
print("Standalone export, failed replacement and relocated launcher PASS")
print("Built game exit/crash/restart/stop and unsaved editor state PASS")
print("Play tools configure/preview/cancel/spawn/impulse/delete/reset/isolation PASS")
print("Bindings edit/persist/reset, viewport picking/clear/cycle, paused actions PASS")
print("Linked resources extract/instantiate/source edit/override/reset/unpack/reopen PASS")
print("Project create/edit/save/play/pause/impulse/step/stop/discard/reopen/resize PASS")

s=Session([])
try:
    s.wait(b"3D Graphics")
    s.send(b"\x1b[B\x1b[B");s.wait(b"> New scene / project")
    s.send(b"\r");s.wait(b"new project directory")
    s.send(b"\x1b");time.sleep(.2)
    s.send(b"q");s.wait(b"3D Graphics")
    s.send(b"\r");s.wait(b"Kinetic gallery")
    s.send(b"\x1b[A"*3);s.wait(b"> Pulse Dock")
    s.send(b"\r");s.wait(b"Arrows move | Space jump")
    s.send(b"q");s.wait(b"3D Graphics")
    s.send(b"\r");s.wait(b"Kinetic gallery")
    s.send(b"\x1b[A");s.wait(b"> Switch Room")
    s.send(b"\r");s.wait(b"WASD walk")
    s.send(b"g");s.wait(b"Holding crate")
    s.send(b"g");s.wait(b"Released")
    s.send(b"f");s.wait(b"Pushed crate")
    s.send(b"q");s.wait(b"3D Graphics")
    s.send(b"\x1b");s.close()
finally:
    s.cleanup()
print("Launcher/editor/launcher/exit PASS")

with tempfile.TemporaryDirectory() as root:
    with open(os.path.join(root, "project.yaml"), "w") as file:
        file.write("version: 1\nname: Trial\nentry_scene: scene.yaml\n")
    with open(os.path.join(root, "scene.yaml"), "w") as file:
        file.write("""version: 1
name: Finish
simulation: {gravityy: 0}
objective: {text: Cross the finish, signal: finish, tick_limit: 90}
entities:
  - id: player
    name: Player
    shape: sphere
    body: dynamic
    size: {x: 0.5, y: 0.5, z: 0.5}
    mass: 1
  - id: finish
    name: Finish
    shape: box
    body: static
    sensor: true
    plate_signal: finish
    position: {x: 2, y: 0, z: 0}
    size: {x: 0.5, y: 1, z: 1}
    mass: 1
""")
    s = Session(["-play-project", root, "-player", "player"])
    try:
        s.wait(b"Objective: Cross the finish")
        s.wait(b"Lost | R restart")
        s.send(b"r"); s.wait(b"Ticks left:")
        s.send(b"\x1b[C"); s.wait(b"Won | R restart")
        s.send(b"r"); s.wait(b"Ticks left:")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
print("Player HUD, timeout, restart, movement and victory PASS")

# Work on a copy so the sample remains reusable after editing and export.
with tempfile.TemporaryDirectory() as parent:
    source = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "examples", "pulse-dock")
    root = shutil.copytree(source, os.path.join(parent, "Pulse Dock"), ignore=shutil.ignore_patterns("build"))
    s = Session(["-project", root])
    try:
        s.wait(b"Opened Pulse Dock")
        s.send(b"\x1b[B\x1b[B"); s.wait(b"> Docking beacon")
        s.send(b"\t"); s.wait(b"Inspector")
        s.send(b"\x1b[B" * 3 + b"e"); s.wait(b"Edit Position X")
        s.send(b"\x7f0\r"); s.wait(b"Property updated")
        s.send(b"\x13"); s.wait(b"Saved scenes/dock.yaml")
        s.send(b"p"); s.wait(b"Won | R restart")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        s.send(b"\x1b[20~"); s.wait(b"Exported build/game-", seconds=90)
        s.send(b"\x1b"); s.wait(b"Hierarchy")
        s.send(b"q"); s.close()
    finally:
        s.cleanup()
    archive, = glob.glob(os.path.join(root, "build", "game-*.zip"))
    release = os.path.join(parent, "Release")
    with zipfile.ZipFile(archive) as package:
        package.extractall(release)
        for executable in ("play.sh", "game"):
            os.chmod(os.path.join(release, executable), package.getinfo(executable).external_attr >> 16 & 0o777)
    os.rename(root, root + "-unavailable")
    run = subprocess.run([os.path.join(release, "play.sh")], cwd=parent, input=b".\nr\n.\nq\n",
                         capture_output=True, check=True, timeout=15)
    assert run.stdout.count(b"Won | R restart") == 2, run.stdout
print("Pulse Dock sample edit, save, Play, export and standalone restart PASS")
