#!/usr/bin/env python3
"""Compare the engine's CPU and OpenGL render paths on a Linux machine."""

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
SOFTWARE = re.compile(r"llvmpipe|lavapipe|softpipe|swrast|swiftshader|software rasterizer", re.I)
BENCHMARK = re.compile(r"^(Benchmark\w+(?:/[\w]+)+)(?:-\d+)?\s+(.*)$")
SAMPLE = re.compile(r"\s*\d+\s+([\d.]+)\s+ns/op\b")


def run(log, *args):
    with log.open("w", encoding="utf-8") as file:
        result = subprocess.run(
            ["go", *args], cwd=ROOT, stdout=file, stderr=subprocess.STDOUT, check=False,
        )
    output = log.read_text(encoding="utf-8", errors="replace")
    if result.returncode:
        print(output[-6000:], file=sys.stderr)
        raise RuntimeError(f"go {args[0]} exited with status {result.returncode}. See {log}")
    return output


def parse_samples(output):
    samples = {}
    pending = None
    for line in output.splitlines():
        match = BENCHMARK.match(line)
        if match:
            pending, line = match.groups()
        sample = SAMPLE.match(line)
        # The EGL device message may follow the benchmark label on the same
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
    args = parser.parse_args()
    if platform.system() != "Linux" or platform.machine() not in {"x86_64", "aarch64", "arm64"}:
        parser.error("the engine's OpenGL backend requires Linux amd64 or arm64 with surfaceless EGL")
    if not shutil.which("go"):
        parser.error("Go 1.24.6 or newer must be on PATH")
    if args.count < 1 or (args.cpu is not None and args.cpu < 1):
        parser.error("--count and --cpu must be positive")
    cpu = args.cpu or min(4, len(os.sched_getaffinity(0)))
    if args.output is None:
        cache = ROOT / ".cache"
        cache.mkdir(exist_ok=True)
        output = Path(tempfile.mkdtemp(prefix="opengl-bench-", dir=cache))
    else:
        output = args.output.resolve()
        output.mkdir(parents=True, exist_ok=False)
    print(f"Results directory: {output}", flush=True)

    version = run(output / "go-version.txt", "version").strip()
    print(f"{version}, CPU budget {cpu}", flush=True)
    (output / "settings.txt").write_text(
        f"{version}\n{platform.system()} {platform.release()} {platform.machine()}\n"
        f"cpu={cpu}\ncount={args.count}\nbenchtime={args.benchtime}\n", encoding="utf-8",
    )
    print("Checking the EGL driver and CPU/OpenGL rendering agreement...", flush=True)
    parity = run(
        output / "parity.txt", "test", "-tags", "terminalgl", "-count=1", "-v",
        "-run", "^TestOpenGL(RasterDemoParity|VisibilityAndFallback|FXAAMatchesCPU)$",
        "./internal/demo", "./internal/terminal",
    )
    drivers = {}
    for kind, driver in re.findall(r"OpenGL (raster|FXAA) device: ([^\r\n]+)", parity):
        drivers.setdefault(kind, set()).add(driver.strip())
    for kind, names in drivers.items():
        print(f"OpenGL {kind}: {', '.join(sorted(names))}", flush=True)
    if set(drivers) != {"raster", "FXAA"} or "--- SKIP:" in parity:
        raise RuntimeError(f"OpenGL validation was skipped. Check the surfaceless EGL driver in {output / 'parity.txt'}")
    if any(SOFTWARE.search(name) for names in drivers.values() for name in names):
        if not args.allow_software:
            raise RuntimeError("EGL selected a software renderer. Select your GPU driver or use --allow-software.")
        print("Software rendering enabled. These timings do not measure hardware GPU performance.", flush=True)

    flags = ["test", "-tags", "terminalgl", "-run", "^$", "-benchmem",
             f"-count={args.count}", f"-cpu={cpu}", f"-benchtime={args.benchtime}"]
    print("Benchmarking scene rasterization...", flush=True)
    raster = parse_samples(run(output / "raster.txt", *flags, "-bench", "^BenchmarkOpenGLSceneRaster$", "./internal/demo"))
    print("Benchmarking FXAA...", flush=True)
    fxaa = parse_samples(run(output / "fxaa.txt", *flags, "-bench", "^BenchmarkFXAADevice$", "./internal/terminal"))
    compare(output, args.count, raster, fxaa)
    print(f"CSV summary: {output / 'summary.csv'}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError) as error:
        sys.exit(f"benchmark_opengl: {error}")
