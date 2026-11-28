#!/usr/bin/env python3
"""Compare the engine's CPU and OpenGL render paths using the native desktop graphics driver."""

import argparse
import csv
import json
import os
from pathlib import Path
import platform
import re
import shutil
import shlex
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


def linux_pci_adapters():
    adapters = []
    vendors = {"0x10de": "NVIDIA", "0x1002": "AMD", "0x8086": "Intel"}
    for device in sorted(Path("/sys/bus/pci/devices").glob("*")):
        try:
            if (device / "class").read_text().startswith("0x03"):
                vendor = (device / "vendor").read_text().strip()
                adapters.append((vendors.get(vendor, ""), "pci-" + device.name.replace(":", "_").replace(".", "_")))
        except OSError:
            continue
    return adapters


def discover_adapters(system):
    """Return (adapter name, optional Mesa PCI selector) pairs."""
    if system == "Windows":
        command = ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
                   "[Console]::OutputEncoding = [Text.Encoding]::UTF8; "
                   "Get-CimInstance Win32_VideoController | Select-Object -ExpandProperty Name | ConvertTo-Json -Compress"]
    elif system == "Darwin":
        command = ["system_profiler", "SPDisplaysDataType", "-json"]
    else:
        command = ["lspci", "-Dmm", "-nn"]
    try:
        output = subprocess.check_output(command, text=True, encoding="utf-8", errors="replace",
                                         stderr=subprocess.DEVNULL, timeout=15)
        if system == "Windows":
            names = json.loads(output) if output.strip() else []
            return [(name, None) for name in ([names] if isinstance(names, str) else names or [])
                    if name and not re.search(r"Microsoft Basic|Remote Display|Virtual Display", name, re.I)]
        if system == "Darwin":
            return [(gpu["_name"], None) for gpu in json.loads(output).get("SPDisplaysDataType", [])
                    if gpu.get("_name")]
        adapters = []
        for line in output.splitlines():
            fields = shlex.split(line)
            if len(fields) >= 4 and re.search(r"\[03(?:00|02|80)\]$", fields[1]):
                name = re.sub(r" \[\w{4}:\w{4}\]$", "", fields[3])
                label = re.search(r"\[([^\]]+)\]$", name)
                adapters.append((label[1] if label else name, "pci-" + fields[0].replace(":", "_").replace(".", "_")))
        return adapters
    except (OSError, ValueError, subprocess.SubprocessError):
        if system == "Linux":
            return linux_pci_adapters()
        print("GPU inventory unavailable. Use --require-renderer to check a specific OpenGL renderer.", flush=True)
        return []


def renderer_matches(name, expected):
    def normalize(value):
        return re.sub(r"[^a-z0-9]", "", re.sub(r"\((?:tm|r)\)|[™®]", "", value.casefold()))
    wanted = normalize(expected)
    return bool(wanted) and wanted in normalize(name)


def select_adapter(system, required):
    if required:
        return required
    adapters = discover_adapters(system)
    if len(adapters) < 2:
        return None
    for index, (name, selector) in enumerate(adapters, 1):
        label = name or "Display adapter"
        print(f"  {index}. {label}" + (f" ({selector})" if selector else ""))
    if not sys.stdin.isatty():
        raise RuntimeError("multiple GPUs detected. Supply --require-renderer to run without a prompt.")
    while True:
        try:
            choice = input("Choose a GPU by number: ").strip()
        except (EOFError, KeyboardInterrupt) as error:
            raise RuntimeError("GPU selection canceled") from error
        if choice.isascii() and choice.isdigit() and len(choice) <= len(str(len(adapters))):
            index = int(choice)
            if 1 <= index <= len(adapters):
                name, selector = adapters[index - 1]
                if selector:
                    os.environ["DRI_PRIME"] = selector
                # Model numbers survive driver differences such as added OEM names.
                model = re.search(r"\b(?:RTX|GTX|RX|Arc [AB])\s*\d{3,4}\b", name, re.I)
                return model[0] if model else name
        print(f"Enter a number from 1 to {len(adapters)}.")


def compare(log_dir, count, raster, fxaa, drivers):
    with (log_dir / "summary.csv").open("w", newline="", encoding="utf-8") as file:
        writer = csv.writer(file)
        writer.writerow(["pass", "workload", "viewport", "raster", "filter", "median_ms", "speedup_vs_best_cpu", "opengl_renderer"])
        print(f"\n{'Workload':<21} {'Viewport':>8}  {'Filter':<8} {'Fastest raster / filter':<24} {'ms':>7}")
        for scene, title in SCENES.items():
            for grid in ("160x60", "320x120"):
                prefix = f"BenchmarkOpenGLSceneRaster/{scene}/{grid}/"
                timings = {
                    (mode, filter): time_ms(raster, prefix + mode + "/" + filter, count)
                    for mode in ("single", "tiles", "jobs", "batched", "opengl")
                    for filter in ("none", "cpu_fxaa", "gl_fxaa")
                    if (mode, filter) != ("opengl", "gl_fxaa")
                }
                for group in ("none", "FXAA"):
                    comparable = {k: ms for k, ms in timings.items() if (k[1] == "none") == (group == "none")}
                    best_cpu = min(ms for (mode, filter), ms in comparable.items() if mode != "opengl" and filter != "gl_fxaa")
                    winner = min(comparable, key=comparable.get)
                    label = " / ".join(winner)
                    print(f"{title:<21} {grid:>8}  {group:<8} {label:<24} {comparable[winner]:7.3f}")
                    for (mode, filter), ms in sorted(comparable.items(), key=lambda item: item[1]):
                        gpu = "raster" if mode == "opengl" else "FXAA" if filter == "gl_fxaa" else None
                        names = ", ".join(sorted(drivers[gpu])) if gpu else ""
                        writer.writerow(["Scene pipeline", title, grid, mode, filter, ms, best_cpu / ms, names])
        print("\nIsolated FXAA (CPU ms / OpenGL ms, speedup)")
        for grid in ("120x40", "160x60", "320x120"):
            prefix = f"BenchmarkFXAADevice/{grid}/"
            cpu, gl = (time_ms(fxaa, prefix + mode, count) for mode in ("cpu", "gl"))
            print(f"{grid:>8}  {cpu:.3f} / {gl:.3f}, {cpu / gl:.2f}x")
            writer.writerow(["FXAA only", "Color filter", grid, "", "cpu_fxaa", cpu, 1, ""])
            writer.writerow(["FXAA only", "Color filter", grid, "", "gl_fxaa", gl, cpu / gl,
                             ", ".join(sorted(drivers["FXAA"]))])
    print("\nSpeedups in the CSV compare matching filter groups against their fastest CPU pipeline.")
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

    required = select_adapter(system, args.require_renderer)
    if required:
        print(f"Required OpenGL renderer: {required}", flush=True)
    drivers = {}
    while True:
        print("Checking the native OpenGL driver and CPU/OpenGL rendering agreement...", flush=True)
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
            if required and any(not renderer_matches(name, required) for name in drivers[kind]):
                hint = " In Windows Settings → System → Display → Graphics, assign both printed executables to the selected GPU." if system == "Windows" else ""
                message = f"OpenGL {kind} did not select {required!r}.{hint}"
                if system != "Windows" or args.require_renderer or not sys.stdin.isatty():
                    raise RuntimeError(message)
                print(message, flush=True)
                try:
                    retry = input("Press Enter after applying the graphics preferences, or type q to quit: ")
                except (EOFError, KeyboardInterrupt) as error:
                    raise RuntimeError("GPU selection canceled") from error
                if retry.strip():
                    raise RuntimeError("GPU selection canceled")
                break
        else:
            break
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
        for stage in (("raster", "FXAA") if kind == "raster" else ("FXAA",)):
            if renderer_names(result, stage) != drivers[stage]:
                raise RuntimeError(f"OpenGL {stage} renderer changed between validation and timing. Check {log}")
        samples[kind] = parse_samples(result)
    compare(output, args.count, samples["raster"], samples["FXAA"], drivers)
    print(f"CSV summary: {output / 'summary.csv'}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError) as error:
        sys.exit(f"benchmark_opengl: {error}")
