#!/usr/bin/env python3
"""Compare the engine's CPU and OpenGL render paths using the native desktop graphics driver."""

import argparse
import csv
import os
from pathlib import Path
import platform
import re
import shutil
import statistics
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
SCENES = {"scene1": "Kinetic gallery", "scene6": "Wave field", "scene7": "Physics playground"}
SOFTWARE = re.compile(r"llvmpipe|lavapipe|softpipe|swrast|swiftshader|software rasterizer|gdi generic|apple software renderer", re.I)
BENCHMARK = re.compile(r"^(Benchmark\w+(?:/[\w]+)+)(?:-\d+)?\s+(.*)$")
SAMPLE = re.compile(r"\s*\d+\s+([\d.]+)\s+ns/op\b")


def run(log, *command, cwd=ROOT):
    with log.open("w", encoding="utf-8") as file:
        result = subprocess.run(
            command, cwd=cwd, stdout=file, stderr=subprocess.STDOUT, check=False,
        )
    output = log.read_text(encoding="utf-8", errors="replace")
    if result.returncode:
        print(output[-6000:], file=sys.stderr)
        raise RuntimeError(f"{Path(command[0]).name} exited with status {result.returncode}. See {log}")
    return output


def renderer_names(output, kind):
    return {name.strip() for name in re.findall(rf"OpenGL {kind} device: ([^\r\n]+)", output)}


def parse_samples(output):
    samples = {}
    pending = None
    for line in output.splitlines():
        match = BENCHMARK.match(line)
        if match:
            pending, line = match.groups()
        sample = SAMPLE.match(line)
        # The OpenGL device message may follow the benchmark label on the same
        # line, leaving the timing on the next line.
        if pending and sample:
            samples.setdefault(pending, []).append(float(sample[1]))
            pending = None
    return samples


def time_ms(samples, name, count):
    times = samples.get(name, [])
    if len(times) != count:
        raise RuntimeError(f"{name} returned {len(times)} of {count} timings. Check the raw logs.")
    return statistics.median(times) / 1e6


def compare(log_dir, count, raster, fxaa):
    rows = []
    for scene, title in SCENES.items():
        for grid in ("160x60", "320x120"):
            prefix = f"BenchmarkOpenGLSceneRaster/{scene}/{grid}/"
            single, batched, gl = (
                time_ms(raster, prefix + name, count) for name in ("gl_false", "batched", "gl_true")
            )
            rows.append(("Scene raster", title, grid, single, batched, gl, min(single, batched) / gl))
    for grid in ("120x40", "160x60", "320x120"):
        prefix = f"BenchmarkFXAADevice/{grid}/"
        cpu, gl = (time_ms(fxaa, prefix + name, count) for name in ("cpu", "gl"))
        rows.append(("FXAA", "Color filter", grid, cpu, None, gl, cpu / gl))

    with (log_dir / "summary.csv").open("w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        writer.writerow(["pass", "workload", "viewport", "cpu_ms", "batched_ms", "opengl_ms", "speedup_vs_best_cpu"])
        writer.writerows(rows)

    print(f"\n{'Workload':<21} {'Viewport':>8} {'CPU ms':>9} {'Batched':>9} {'GL ms':>9} {'Speedup':>9}")
    for _, scene, grid, cpu, batched, gl, ratio in rows:
        multicore = f"{batched:.3f}" if batched is not None else "-"
        print(f"{scene:<21} {grid:>8} {cpu:9.3f} {multicore:>9} {gl:9.3f} {ratio:8.2f}x")
    print("\nSpeedup above 1.00x favors OpenGL against the fastest CPU path in that row.")
    print("Timings include GL uploads and readback. Terminal painting and simulation are excluded.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, help="new directory for raw logs and summary.csv")
    parser.add_argument("--count", type=int, default=3, help="repetitions per benchmark (default: 3)")
    parser.add_argument("--benchtime", default="300ms", help="Go duration per repetition (default: 300ms)")
    parser.add_argument("--cpu", type=int, help="Go CPU budget and batched workers (default: up to 4)")
    parser.add_argument("--allow-software", action="store_true", help="continue with a known software renderer")
    parser.add_argument("--prepare-only", action="store_true", help="build the benchmark executables and print their paths")
    parser.add_argument("--require-renderer", help="require this substring in every OpenGL renderer name (case-insensitive)")
    args = parser.parse_args()
    system = platform.system()
    if system not in {"Linux", "Windows", "Darwin"} or platform.machine().lower() not in {"amd64", "x86_64", "aarch64", "arm64"}:
        parser.error("the engine's OpenGL backend requires Linux, Windows, or macOS on amd64 or arm64")
    if not shutil.which("go"):
        parser.error("Go 1.24.6 or newer must be on PATH")
    if args.count < 1 or (args.cpu is not None and args.cpu < 1):
        parser.error("--count and --cpu must be positive")
    available_cpus = len(os.sched_getaffinity(0)) if hasattr(os, "sched_getaffinity") else (os.cpu_count() or 1)
    cpu = args.cpu or min(4, available_cpus)
    if args.output is None:
        cache = ROOT / ".cache"
        cache.mkdir(exist_ok=True)
        output = Path(tempfile.mkdtemp(prefix="opengl-bench-", dir=cache))
    else:
        output = args.output.resolve()
        output.mkdir(parents=True, exist_ok=False)
    print(f"Results directory: {output}", flush=True)

    version = run(output / "go-version.txt", "go", "version").strip()
    print(f"{version}, CPU budget {cpu}", flush=True)
    (output / "settings.txt").write_text(
        f"{version}\n{platform.system()} {platform.release()} {platform.machine()}\n"
        f"cpu={cpu}\ncount={args.count}\nbenchtime={args.benchtime}\n", encoding="utf-8",
    )
    binary_dir = ROOT / ".cache" / "opengl-bin"
    binary_dir.mkdir(parents=True, exist_ok=True)
    extension = ".exe" if system == "Windows" else ""
    packages = {"raster": "internal/demo", "FXAA": "internal/terminal"}
    executables = {}
    for kind, package in packages.items():
        binary = binary_dir / f"opengl-{kind.lower()}{extension}"
        run(output / f"build-{kind.lower()}.txt", "go", "test", "-tags", "terminalgl",
            "-c", "-o", str(binary), f"./{package}")
        print(f"OpenGL {kind} executable: {binary}", flush=True)
        executables[kind] = binary
    if args.prepare_only:
        return

    print("Checking the native OpenGL driver and CPU/OpenGL rendering agreement...", flush=True)
    drivers = {}
    for kind, package in packages.items():
        parity_log = output / f"parity-{kind.lower()}.txt"
        parity = run(
            parity_log, str(executables[kind]), "-test.count=1", "-test.v",
            "-test.run=^TestOpenGL(RasterDemoParity|VisibilityAndFallback|FXAAMatchesCPU)$",
            cwd=ROOT / package,
        )
        drivers[kind] = renderer_names(parity, kind)
        if not drivers[kind] or "--- SKIP:" in parity:
            raise RuntimeError(f"OpenGL {kind} validation was skipped. Check {parity_log}")
        print(f"OpenGL {kind}: {', '.join(sorted(drivers[kind]))}", flush=True)
        if args.require_renderer and any(args.require_renderer.casefold() not in name.casefold() for name in drivers[kind]):
            raise RuntimeError(f"OpenGL {kind} did not select {args.require_renderer!r}. Check the graphics preference for {executables[kind]}")
    if any(SOFTWARE.search(name) for names in drivers.values() for name in names):
        if not args.allow_software:
            raise RuntimeError("OpenGL selected a software renderer. Select your GPU driver or use --allow-software.")
        print("Software rendering enabled. These timings do not measure hardware GPU performance.", flush=True)

    flags = ["-test.run=^$", "-test.benchmem", f"-test.count={args.count}",
             f"-test.cpu={cpu}", f"-test.benchtime={args.benchtime}"]
    samples = {}
    for kind, benchmark in (("raster", "BenchmarkOpenGLSceneRaster"), ("FXAA", "BenchmarkFXAADevice")):
        print(f"Benchmarking {kind}...", flush=True)
        log = output / f"{kind.lower()}.txt"
        result = run(log, str(executables[kind]), *flags, f"-test.bench=^{benchmark}$", cwd=ROOT / packages[kind])
        if renderer_names(result, kind) != drivers[kind]:
            raise RuntimeError(f"OpenGL {kind} renderer changed between validation and timing. Check {log}")
        samples[kind] = parse_samples(result)
    compare(output, args.count, samples["raster"], samples["FXAA"])
    print(f"CSV summary: {output / 'summary.csv'}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError) as error:
        sys.exit(f"benchmark_opengl: {error}")
