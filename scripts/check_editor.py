"""Project editor PTY workflow; run after go build -race -o /tmp/graphics ."""
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
        s.send(b"g"); s.wait(b"Added sphere")
        s.send(b"e"); s.wait(b"Edit Name")
        s.send(b"\x7f" * len("New sphere") + b"Player\r"); s.wait(b"Property updated")
        s.send(b"\x13"); s.wait(b"Saved scenes/main.yaml")
        scene_path = os.path.join(root, "scenes", "main.yaml")
        with open(scene_path, "rb") as file:
            authored = file.read()
        assert b"name: Player" in authored
        s.send(b"p"); s.wait(b"[PLAY]")
        time.sleep(.2)
        s.send(b" "); s.wait(b"[PAUSED]")
        s.send(b"i"); s.wait(b"Upward impulse queued")
        s.send(b"n"); s.wait(b"Pending:0")
        s.send(b"\x1b"); s.wait(b"Stopped; authored scene preserved")
        with open(scene_path, "rb") as file:
            assert file.read() == authored, "Play changed saved scene"
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
print("Project create/edit/save/play/pause/impulse/step/stop/discard/reopen/resize PASS")

s=Session([])
try:
    s.wait(b"3D Graphics")
    s.send(b"e");s.wait(b"Project editor")
    s.send(b"q");s.wait(b"3D Graphics")
    s.send(b"\x1b");s.close()
finally:
    s.cleanup()
print("Launcher/editor/launcher/exit PASS")
