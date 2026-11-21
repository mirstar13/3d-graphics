"""Linux/macOS PTY smoke test. Run after go build -race -o /tmp/graphics .
Usage: python3 scripts/check_terminal.py /tmp/graphics
"""
import os,pty,fcntl,termios,struct,select,time,signal,sys,re
BINARY=os.path.abspath(sys.argv[1] if len(sys.argv)>1 else "/tmp/graphics")

class Session:
    def __init__(self,args):
        self.pid,self.fd=pty.fork()
        if self.pid==0:
            os.environ.update(TERM="xterm-256color",COLORTERM="truecolor")
            fcntl.ioctl(0,termios.TIOCSWINSZ,struct.pack("HHHH",30,100,0,0))
            os.execv(BINARY,[BINARY]+args)
        self.output=b""
    def send(self,data): os.write(self.fd,data)
    def wait(self,needle,seconds=15):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            if needle in self.output: return
            if select.select([self.fd],[],[],.05)[0]:
                try: data=os.read(self.fd,65536)
                except OSError: break
                self.output+=data
                if b"\x1b[6n" in data: self.send(b"\x1b[1;1R")
        raise AssertionError((needle,self.output[-1600:]))
    def exit(self):
        end=time.monotonic()+8
        while time.monotonic()<end:
            if select.select([self.fd],[],[],.05)[0]:
                try:self.output+=os.read(self.fd,65536)
                except OSError:time.sleep(.05)
            pid,status=os.waitpid(self.pid,os.WNOHANG)
            if pid:
                assert os.waitstatus_to_exitcode(status)==0,self.output[-4000:]
                assert b"DATA RACE" not in self.output,self.output[-4000:]
                flags=termios.tcgetattr(self.fd)[3]
                assert flags & termios.ECHO and flags & termios.ICANON, "input mode not restored"
                assert b"\x1b[?25h" in self.output, "cursor not restored"
                os.close(self.fd)
                return
        os.kill(self.pid,signal.SIGKILL);os.waitpid(self.pid,0)
        raise AssertionError(("exit timed out",self.output[-2000:]))


s=Session([])
s.wait(b"3D Graphics")
assert b"New scene" in s.output and b"Settings" in s.output and b"Exit" in s.output
s.output=b"";s.send(b"\x1b[B\r")
s.wait(b"Renderer settings")
s.send(b"\x1b[B\x1b[C")
time.sleep(.15)
s.output=b"";s.send(b"\r")
s.wait(b"3D Graphics")
# Return to New scene, open and cancel the picker.
s.output=b"";s.send(b"\x1b[A\r")
s.wait(b"Kinetic gallery")
s.output=b"";s.send(b"\x1b")
s.wait(b"3D Graphics")
# Launch the default scene with the settings applied above.
s.output=b"";s.send(b"\r")
s.wait(b"Kinetic gallery")
s.output=b"";s.send(b"\r")
s.wait(b"FPS:")
# Paused time remains fixed, N advances exactly once, Enter resumes.
s.output=b"";s.send(b" ");s.wait(b"PAUSED")
match=re.search(rb"PAUSED 60Hz Tick:(\d+)",s.output)
assert match,s.output[-1000:]
paused_tick=int(match.group(1))
time.sleep(.15)
s.output=b"";s.send(b"n");s.wait(f"PAUSED 60Hz Tick:{paused_tick+1} |".encode())
s.output=b"";s.send(b"\r");s.wait(b"RUNNING")
# Resize while workers and filters are active, then verify the new status row.
for rows,columns in [(18,65),(40,120)]:
    s.output=b""
    fcntl.ioctl(s.fd,termios.TIOCSWINSZ,struct.pack("HHHH",rows,columns,0,0))
    s.wait(f"\x1b[{rows};1H".encode())
s.output=b"";s.send(b"x")
s.wait(b"3D Graphics")
# A second rendering session, then return via P and settings Escape.
s.output=b"";s.send(b"\r")
s.wait(b"Kinetic gallery")
s.output=b"";s.send(b"\r")
s.wait(b"FPS:")
s.output=b"";s.send(b"p")
s.wait(b"Renderer settings")
s.output=b"";s.send(b"\x1b")
s.wait(b"3D Graphics")
s.send(b"\x1b[B\x1b[B\r")
s.exit()
print("Menu/settings/apply/scene/back/restart/exit PASS",flush=True)

s=Session(["-settings"])
s.wait(b"Renderer settings")
s.output=b"";s.send(b"\x1b")
s.wait(b"3D Graphics")
s.send(b"\x1b")
s.exit()
print("-settings shortcut/back/exit PASS",flush=True)

for color,key in [("none",b"\x1b"),("16",b"\x03"),("256",b"x"),("truecolor",b"x")]:
    s=Session(["-color",color,"-aa","msaa4","-mode","tiles"])
    s.wait(b"3D Graphics")
    s.output=b"";s.send(b"\r");s.wait(b"Kinetic gallery")
    s.output=b"";s.send(b"\r");s.wait(b"FPS:")
    s.output=b"";s.send(key);s.wait(b"3D Graphics")
    s.send(b"\x1b");s.exit()
print("Color modes, Escape/Ctrl+C/X, console restoration PASS",flush=True)

# Pause help must remain available when ordinary debug display is hidden.
s=Session(["-debug=false"])
s.wait(b"3D Graphics")
s.output=b"";s.send(b"\r");s.wait(b"Kinetic gallery")
s.output=b"";s.send(b"\r");s.wait(b"\x1b[?25l")
# Wait briefly for scene input startup after Bubble Tea shuts down.
time.sleep(.25)
s.output=b"";s.send(b" ");s.wait(b"Enter resume")
s.output=b"";s.send(b"p");s.wait(b"Renderer settings")
s.output=b"";s.send(b"\x1b");s.wait(b"3D Graphics")
s.send(b"\x1b");s.exit()
print("Simulation pause/step/resume and hidden-debug help PASS",flush=True)

# Run the new body/collider scene with post-AA overlays and fixed stepping.
s=Session(["-demo","7","-auto-orbit=false","-aa","msaa4","-mode","tiles"])
s.wait(b"3D Graphics")
s.output=b"";s.send(b"\r");s.wait(b"Physics playground")
s.output=b"";s.send(b"\r");s.wait(b"Bodies:")
s.output=b"";s.send(b"c");s.wait(b"C:ON")
s.output=b"";s.send(b" ");s.wait(b"PAUSED")
match=re.search(rb"PAUSED 60Hz Tick:(\d+)",s.output)
assert match,s.output[-1000:]
s.output=b"";s.send(b"n");s.wait(f"PAUSED 60Hz Tick:{int(match.group(1))+1} |".encode())
s.output=b"";s.send(b"c");s.wait(b"C:OFF")
s.output=b"";s.send(b"x");s.wait(b"3D Graphics")
s.send(b"\x1b");s.exit()
print("Physics playground, collider overlays, pause/step/exit PASS",flush=True)

# Physics settings remain editable and persist through the scrolling TUI.
import tempfile
with tempfile.TemporaryDirectory() as directory:
    path=os.path.join(directory,"renderer.yaml")
    with open(path,"w") as file: file.write("demo: 7\n")
    s=Session(["-settings","-config",path])
    s.wait(b"Renderer settings")
    s.output=b"";s.send(b"\x1b[B"*20);s.wait(b"Sphere CCD")
    s.output=b"";s.send(b"\x1b[C");time.sleep(.1)
    s.send(b"s");time.sleep(.2)
    with open(path) as file: saved=file.read()
    assert "ccd: true" in saved,saved
    s.output=b"";s.send(b"\r");s.wait(b"3D Graphics")
    s.send(b"\x1b");s.exit()
print("Physics settings scroll/edit/save/apply PASS",flush=True)
