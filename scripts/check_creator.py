"""Build and run the public API contract from a separate Go module."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

repo = Path(__file__).resolve().parents[1]
env = {**os.environ, "GOWORK": "off"}
with tempfile.TemporaryDirectory() as directory:
    target = Path(directory) / "consumer"
    target.mkdir()
    fixture = Path(directory) / "examples" / "relay-rooms"
    fixture.mkdir(parents=True)
    shutil.copy(repo / "examples/relay-rooms/project.yaml", fixture)
    shutil.copytree(repo / "examples/relay-rooms/scenes", fixture / "scenes")
    shutil.copy(repo / "creator/creator_test.go", target / "behavior_test.go")
    shutil.copy(repo / "go.sum", target / "go.sum")
    module = "github.com/mirstar13/go-3d-graphics"
    for args in [
        ["mod", "init", "example.org/terminalgame"],
        ["mod", "edit", f"-require={module}@v0.0.0", f"-replace={module}={repo}"],
        ["test", "-mod=mod", "-race", "-v", "./..."],
    ]:
        subprocess.run(["go", *args], cwd=target, env=env, check=True)
