#!/usr/bin/env python3
"""Sweep CPU and native OpenGL raster/FXAA pipelines and save local performance reports."""

import argparse
import csv
from dataclasses import dataclass, field
import hashlib
from html import escape
import itertools
import json
import math
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
RASTERS = ("single", "tiles", "jobs", "batched")
RENDER_SCENES = ("kinetic", "wave", "physics")
GAME_SCENES = ("physics", "switch-room")
SOFTWARE = re.compile(r"llvmpipe|lavapipe|softpipe|swrast|swiftshader|software rasterizer|gdi generic|apple software renderer", re.I)


@dataclass
class Adapter:
    id: str
    name: str
    binary: Path
    routes: list
    stages: dict = field(default_factory=dict)
    verified_env: dict = field(default_factory=dict)
    errors: dict = field(default_factory=dict)

    @property
    def label(self):
        return f"{self.name} [{self.id}]"


@dataclass(frozen=True)
class Case:
    scene: str
    columns: int
    rows: int
    viewport: str
    paced: bool
    live: bool
    raster: str
    filter: str
    raster_gpu: str = ""
    filter_gpu: str = ""

    @property
    def group(self):
        return (self.scene, self.columns, self.rows, self.viewport, self.paced, self.live,
                self.filter != "none")

    @property
    def replay(self):
        return self.group[:-1]


def tool_json(command):
    try:
        return json.loads(subprocess.check_output(command, text=True, encoding="utf-8",
                          errors="replace", stderr=subprocess.DEVNULL, timeout=20))
    except (OSError, ValueError, subprocess.SubprocessError):
        return None


def discover_adapters(system, binary):
    if system == "Windows":
        rows = tool_json(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
                         "[Console]::OutputEncoding = [Text.Encoding]::UTF8; "
                         "Get-CimInstance Win32_VideoController | Select-Object Name,PNPDeviceID | ConvertTo-Json -Compress"])
        if isinstance(rows, dict):
            rows = [rows]
        return [(row["PNPDeviceID"], row["Name"], [{}]) for row in rows or []
                if row.get("PNPDeviceID") and row.get("Name") and
                not re.search(r"Microsoft Basic|Remote Display|Virtual Display", row["Name"], re.I)]
    if system == "Darwin":
        ids = tool_json([str(binary), "-inventory"])
        if ids:
            return [(id_, f"CGL renderer {id_}", [{"GRAPHICS_CGL_RENDERER": id_}]) for id_ in ids]
        data = tool_json(["system_profiler", "SPDisplaysDataType", "-json"]) or {}
        return [(row.get("spdisplays_device-id", "") + f"-{i}", row["_name"], [{}])
                for i, row in enumerate(data.get("SPDisplaysDataType", []), 1) if row.get("_name")]
    names = {}
    try:
        listing = subprocess.check_output(["lspci", "-D"], text=True, stderr=subprocess.DEVNULL,
                                          timeout=10)
        for line in listing.splitlines():
            address, _, description = line.partition(" ")
            names[address] = description.partition(": ")[2]
    except (OSError, subprocess.SubprocessError):
        pass
    chips = []
    vendors = {"0x10de": "NVIDIA", "0x1002": "AMD", "0x8086": "Intel"}
    for pci in sorted(Path("/sys/bus/pci/devices").glob("*")):
        try:
            if not (pci / "class").read_text().startswith("0x03"):
                continue
            vendor = vendors.get((pci / "vendor").read_text().strip(), "")
        except OSError:
            continue
        name = names.get(pci.name) or vendor or "PCI display adapter"
        if label := re.search(r"\[([^\]]+)\]", name):
            name = label[1]
        prime = "pci-" + pci.name.replace(":", "_").replace(".", "_")
        chips.append((pci.name, name, [{"GRAPHICS_EGL_PCI": pci.name, "DRI_PRIME": prime},
                                      {"GRAPHICS_EGL_PCI": "", "DRI_PRIME": prime}]))
    return chips


def matches(driver, expected):
    def normalize(value):
        return re.sub(r"[^a-z0-9]", "", re.sub(r"\((?:tm|r)\)|[™®]", "", value.casefold()))
    model = re.search(r"\b(?:RTX|GTX|RX|Arc [AB])\s*\d{3,4}\b", expected, re.I)
    wanted = normalize(model[0] if model else expected)
    return bool(wanted) and wanted in normalize(driver)


def run(log, command, env=None, live=False, timeout=None):
    try:
        with log.open("w", encoding="utf-8") as stream:
            launched = subprocess.run(command, cwd=ROOT, env=env, stderr=stream,
                                      stdout=None if live else stream, timeout=timeout)
    except subprocess.TimeoutExpired as error:
        raise RuntimeError(f"timed out. See {log.name}") from error
    output = log.read_text(encoding="utf-8", errors="replace")
    if launched.returncode:
        last_line = output.strip().splitlines()[-1:] or ["exit status " + str(launched.returncode)]
        raise RuntimeError(f"{last_line[0]} ({log.name})")
    return output


def probe(adapter, stage, output, base_env, args, ambiguous):
    for index, route in enumerate(adapter.routes):
        log = output / f"probe-{adapter.binary.stem}-{stage}-{index}.txt"
        env = {**base_env, **route}
        try:
            result = run(log, [str(adapter.binary), "-probe", "-filter", "gl" if stage == "filter" else "none"],
                         env, timeout=25)
            driver = json.loads(result.splitlines()[-1])["driver"]
            if SOFTWARE.search(driver) and (not args.allow_software or adapter.id != "default"):
                raise RuntimeError(f"selected software renderer {driver}")
            expected = args.require_renderer or (adapter.name if adapter.id != "default" else "")
            # The EGL device path was resolved to the exact PCI address.
            exact = bool(route.get("GRAPHICS_EGL_PCI") or route.get("GRAPHICS_CGL_RENDERER"))
            if ambiguous and not exact:
                raise RuntimeError("identical adapter names and no verifiable device selector")
            if expected and (args.require_renderer or not exact) and not matches(driver, expected):
                raise RuntimeError(f"selected {driver}, expected {expected}")
        except (OSError, ValueError, KeyError, IndexError, RuntimeError) as error:
            adapter.errors[stage] = str(error)
            continue
        adapter.stages[stage], adapter.verified_env[stage] = driver, route
        adapter.errors.pop(stage, None)
        return


def matrix(args, adapters):
    raster = [(mode, "") for mode in RASTERS] + [("gl", gpu.id) for gpu in adapters]
    filters = [(mode, "") for mode in ("none", "cpu")] + [("gl", gpu.id) for gpu in adapters]
    if args.profile == "render":
        grids = args.grid or [(160, 60), (320, 120)]
        scenes = args.scene or RENDER_SCENES
        for scene, (columns, rows), (r, rid), (f, fid) in itertools.product(
                (scene for scene in scenes if scene != "fxaa"), grids, raster, filters):
            yield Case(scene, columns, rows, "fixed", False, False, r, f, rid, fid)
        if not args.scene or "fxaa" in scenes:
            filter_grids = grids if args.grid else [(120, 40), *grids]
            for (columns, rows), (f, fid) in itertools.product(filter_grids, filters[1:]):
                yield Case("fxaa", columns, rows, "fixed", False, False, "single", f, "", fid)
    else:
        grids = args.grid or [terminal_grid()]
        paces = {"both": (False, True), "paced": (True,), "uncapped": (False,)}[args.pacing]
        layouts = ("centered", "full") if args.viewport == "both" else (args.viewport,)
        for scene, (columns, rows), viewport, paced, live, (r, rid), (f, fid) in itertools.product(
                args.scene or GAME_SCENES, grids, layouts, paces, (False, True) if args.live else (False,),
                raster, filters):
            yield Case(scene, columns, rows, viewport, paced, live, r, f, rid, fid)


def terminal_grid():
    try:
        size = os.get_terminal_size()
        return size.columns, size.lines
    except OSError:
        return 161, 61


def quantile(items, fraction):
    return sorted(items)[max(0, math.ceil(len(items)*fraction)-1)] if items else None


def summarize(sample, fps):
    frames = sample["Frames"]
    wall = [f["TotalNS"] / 1e6 for f in frames]
    intervals = [f["IntervalNS"] / 1e6 for f in frames if f["IntervalNS"]]
    gc = [sample["GC"], sample["WorkerGC"]] if sample["WorkerGC"] else [sample["GC"]]
    elapsed = sample["ElapsedNS"] / 1e9
    presented = [f for f in frames if f["Presented"]]
    presentation = [f["PresentNS"] / 1e6 for f in presented if f["PresentNS"]]
    grids = list(dict.fromkeys(f'{f["Columns"]}x{f["Rows"]}' for f in frames))
    result = {
        "observed_grid": ", ".join(grids),
        "grid_changed": len(grids) > 1,
        "presentation": sample["Presentation"],
        "presented_frames": len(presented),
        "coalesced_frames": len(frames) - len(presented) if sample["Presentation"] == "Bubble Tea" else 0,
        "p99_submission_ms": quantile(presentation, .99),
        "frames": len(frames), "fps": len(frames) / elapsed, "elapsed_s": elapsed,
        "median_ms": statistics.median(wall), "p95_ms": quantile(wall, .95), "p99_ms": quantile(wall, .99),
        "p95_interval_ms": quantile(intervals, .95), "p99_interval_ms": quantile(intervals, .99),
        "over_budget_frames": sum(value > 1000/fps for value in wall),
        "late_intervals": sum(value > 1000/fps + .5 for value in intervals),
        "present_deadline_misses": sum(value > 1000/fps for value in presentation),
        "gc_cycles": sum(m["Cycles"] for m in gc), "gc_cpu_ms": sum(m["CPUSeconds"] for m in gc)*1000,
        "gc_worker_capacity_pct_estimate": 100*sum(m["CPUSeconds"] for m in gc)/(elapsed*sample["CPUs"]*len(gc)),
        "gc_assist_ms": sum(m["AssistSeconds"] for m in gc)*1000,
        "allocated_bytes_per_frame": sum(m["AllocBytes"] for m in gc)/len(frames),
        "allocated_objects_per_frame": sum(m["AllocObjects"] for m in gc)/len(frames),
        "peak_runtime_bytes_main": sample["GC"]["PeakRuntimeBytes"],
        "peak_runtime_bytes_worker": gc[1]["PeakRuntimeBytes"] if len(gc) > 1 else None,
        "viewport_drawn": f'{frames[-1]["Width"]}x{frames[-1]["Height"]}',
        "bytes_per_presented_frame": statistics.mean(f["Bytes"] for f in presented) if presented else None,
        "transport": sample["Transport"],
    }
    return result


def execute_case(case, args, index, adapters, binary, env, output, states):
    rgpu, fgpu = adapters.get(case.raster_gpu), adapters.get(case.filter_gpu)
    row = vars(case).copy()
    row["raster_driver"] = rgpu.stages.get("raster", "") if rgpu else ""
    row["filter_driver"] = fgpu.stages.get("filter", "") if fgpu else ""
    errors = [gpu.errors.get(stage) for gpu, stage in ((rgpu, "raster"), (fgpu, "filter")) if gpu]
    if any(errors):
        return [{**row, "status": "unavailable", "reason": " | ".join(filter(None, errors))}]

    active = rgpu or fgpu
    stage = "raster" if rgpu else "filter"
    active_env = {**env, **active.verified_env[stage]} if active else env
    command = [str(active.binary if active else binary),
               "-workload", case.scene, "-profile", args.profile, "-raster", case.raster, "-filter", case.filter,
               "-columns", str(case.columns), "-rows", str(case.rows), "-viewport", case.viewport,
               "-fps", str(args.fps), "-tile", str(args.tile), "-duration", args.benchtime,
               "-frames", str(args.frames), "-warmup", str(args.warmup), "-color", args.color]
    if case.paced:
        command += ["-paced"]
    if case.live:
        command += ["-live"]
    if rgpu and fgpu and (rgpu.id != fgpu.id or rgpu.verified_env["raster"] != fgpu.verified_env["filter"]):
        command += ["-filter-worker", str(fgpu.binary), "-filter-env", json.dumps(fgpu.verified_env["filter"])]
    if args.gc:
        command += ["-memory-bytes", str(args.memory_mib * 1024 * 1024)]
        # Keep the memory limit identical in each half of the pair.
        active_env = {**active_env, "GOMEMLIMIT": args.gomemlimit}
    result = []
    for repetition in range(args.count):
        modes = ("normal", "off") if repetition % 2 == 0 else ("off", "normal")
        for gc in modes if args.gc else ("normal",):
            prefix = f"case-{index:04d}-{repetition+1}-{gc}"
            try:
                path = output / f"{prefix}.json"
                timed_env = {**active_env, "GOGC": "off"} if gc == "off" else active_env
                log = output / f"{prefix}.log"
                run(log, [*command, "-output", str(path)], timed_env, live=case.live, timeout=args.timeout)
                sample = json.loads(path.read_text(encoding="utf-8"))
                if not log.stat().st_size:
                    log.unlink()
                if (sample["RasterDriver"], sample["FilterDriver"]) != (row["raster_driver"], row["filter_driver"]):
                    raise RuntimeError("timed adapter differs from the verified adapter")
                if args.frames and case.scene != "fxaa" and not case.live:
                    state = sample["State"]
                    if states.setdefault(case.replay, state) != state:
                        raise RuntimeError("replay final state differs from the CPU reference")
                stats = summarize(sample, args.fps)
                reason = ""
                if gc == "off" and stats["gc_cycles"]:
                    reason = "GC ran with GOGC=off under the configured memory limit"
                expected_grid = f"{case.columns}x{case.rows}"
                if case.live and stats["observed_grid"] != expected_grid:
                    reason = f"terminal grid differed from planned {expected_grid}; inspect resize samples"
                status = "inconclusive" if reason else "ok"
                result.append({**row, **stats, "gc": gc, "repetition": repetition+1,
                               "status": status, "reason": reason, "raw": path.name})
            except (OSError, ValueError, RuntimeError) as error:
                result.append({**row, "gc": gc, "repetition": repetition+1,
                               "status": "failed", "reason": str(error)})
    if args.detail and not case.live and any(row["status"] == "ok" for row in result):
        path = output / f"case-{index:04d}-detail.json"
        diagnostic = [*command, "-detail", "-output", str(path)]
        if args.trace:
            diagnostic += ["-trace", str(path.with_suffix(".trace")), "-alloc-profile", str(path.with_suffix(".allocs"))]
        try:
            run(path.with_suffix(".log"), diagnostic, active_env, timeout=args.timeout)
            sample = json.loads(path.read_text(encoding="utf-8"))
            if (sample["RasterDriver"], sample["FilterDriver"]) != (row["raster_driver"], row["filter_driver"]):
                raise RuntimeError("diagnostic adapter differs from the verified adapter")
            for item in result:
                item["detail"] = path.name
                if item.get("median_ms"):
                    item["detail_work_change_pct"] = 100*(summarize(sample, args.fps)["median_ms"]/item["median_ms"]-1)
        except (OSError, ValueError, RuntimeError) as error:
            for item in result:
                item["detail_error"] = str(error)
    return result


def write_csv(path, rows):
    if not rows:
        return
    fields = list(dict.fromkeys(key for row in rows for key in row))
    with path.open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(stream, fields)
        writer.writeheader()
        writer.writerows(rows)


def table(headers, rows):
    head = "".join(f"<th>{escape(str(cell))}</th>" for cell in headers)
    body = "".join("<tr>" + "".join(f"<td>{escape(str(cell))}</td>" for cell in row) + "</tr>" for row in rows)
    return f"<table><thead><tr>{head}</tr></thead><tbody>{body}</tbody></table>"


def fmt(value):
    return "—" if value is None else f"{value:.3f}"


def group_title(group):
    scene, columns, rows, viewport, paced, live = group
    return f"{scene} · {columns}×{rows} · {viewport} · {'paced' if paced else 'uncapped'} · {'terminal' if live else 'sink'}"


def detail_html(case, raw, output):
    result = json.loads((output / raw).read_text(encoding="utf-8"))
    frames = result["Frames"]
    elapsed = statistics.median(frame["TotalNS"] for frame in frames)
    pair = (min(frames, key=lambda f: abs(f["TotalNS"] - elapsed)), max(frames, key=lambda f: f["TotalNS"]))
    times = [f["TotalNS"] / 1e6 for f in frames]
    width = 700
    points = " ".join(f"{i * width / max(1, len(times)-1):.2f},{110 - 100 * ms / max(times):.2f}"
                      for i, ms in enumerate(times))
    timeline = (f'<svg viewBox="0 0 {width} 115" role="img" aria-label="Frame times">'
                f'<polyline points="{points}" fill="none" stroke="#44b9b0" stroke-width="2"/></svg>')
    stages = []
    for name, frame in zip(("Typical", "Slowest"), pair):
        raster, fxaa = frame["RasterDevice"], frame["FilterDevice"]
        stages.append((name, frame["Frame"], fmt(frame["TotalNS"]/1e6), *(
            fmt(frame[k]/1e6) for k in ("UpdateNS", "InputNS", "SimulationNS", "LastPhysicsStepNS", "PrepareNS", "RasterNS", "ShadeNS", "FilterNS",
                                       "ExchangeNS", "EncodeNS", "WriteNS")),
            *(fmt(v/1e6 if v is not None else None) for v in
              (raster["UploadNS"], raster["ReadbackNS"], fxaa["UploadNS"], fxaa["ReadbackNS"],
               raster["GPUNS"], fxaa["GPUNS"])),
            fmt((frame["TotalNS"] - sum(frame[k] for k in (
                "UpdateNS", "PrepareNS", "RasterNS", "FilterNS", "EncodeNS", "WriteNS")))/1e6)))
    headers = ("Frame", "ID", "Total", "Update", "Input/camera", "Simulation", "Last physics step", "Prepare", "Raster", "CPU shade", "FXAA",
               "Worker round trip", "Encode", "Write", "Raster upload", "Raster readback/wait",
               "FXAA upload", "FXAA readback/wait", "Raster GPU", "FXAA GPU", "Other")
    return (f'<details><summary>{escape(case)} · {len(times)} frames · peak {max(times):.2f} ms</summary>'
            f'{timeline}{table(headers, stages)}<a href="{escape(raw)}">Raw frame samples</a></details>')


def write_report(output, observations, adapters, fps):
    workloads = {}
    for case, entry in observations.items():
        workloads.setdefault(case.replay, {})[case] = entry
    summary = []
    paired = []
    sections = []
    for group, pipelines in workloads.items():
        medians = {}
        for case, samples in pipelines.items():
            normal = [row for row in samples if row["status"] == "ok" and row["gc"] == "normal"]
            if normal:
                medians[case] = statistics.median(row["median_ms"] for row in normal)
        baselines = {}
        for case, ms in medians.items():
            if not case.raster_gpu and not case.filter_gpu:
                key = case.filter != "none"
                baselines[key] = min(ms, baselines.get(key, ms))
        raster_axes = list(dict.fromkeys((case.raster, case.raster_gpu) for case in pipelines))
        filter_axes = list(dict.fromkeys((case.filter, case.filter_gpu) for case in pipelines))
        heat = {}
        diagnostics = []
        for case, samples in pipelines.items():
            med = medians.get(case)
            baseline = baselines.get(case.filter != "none")
            speedup = baseline / med if med and baseline else None
            normal = [row for row in samples if row["status"] == "ok" and row["gc"] == "normal"]
            row = {key: value for key, value in samples[0].items()
                   if key not in ("raw", "repetition", "gc", "fps", "elapsed_s")}
            row.update(median_ms=med, speedup_vs_best_cpu=speedup)
            if normal:
                row.update(status="ok" if len(normal) == sum(s.get("gc") == "normal" for s in samples)
                           else "partial", reason="",
                           samples=sum(s["frames"] for s in normal), repetitions=len(normal),
                           min_repeat_median_ms=min(s["median_ms"] for s in normal),
                           max_repeat_median_ms=max(s["median_ms"] for s in normal),
                           p99_ms=statistics.median(s["p99_ms"] for s in normal))
                for metric in ("p95_ms", "p95_interval_ms", "p99_interval_ms", "p99_submission_ms",
                               "fps", "bytes_per_presented_frame", "allocated_bytes_per_frame"):
                    values = [s[metric] for s in normal if s[metric] is not None]
                    row[metric] = statistics.median(values) if values else None
            summary.append(row)
            heat[(case.raster, case.raster_gpu, case.filter, case.filter_gpu)] = (row, speedup)
            by_run = {(s["repetition"], s["gc"]): s for s in samples if s.get("repetition")}
            for rep, gc in list(by_run):
                if gc != "normal" or (rep, "off") not in by_run:
                    continue
                on, off = by_run[(rep, "normal")], by_run[(rep, "off")]
                pair = {**vars(case), "repetition": rep, "status": "inconclusive",
                        "normal_status": on["status"], "off_status": off["status"]}
                if on["status"] == off["status"] == "ok":
                    pair.update(status="ok" if on["gc_cycles"] else "no_normal_gc_cycle",
                                normal_gc_cycles=on["gc_cycles"],
                                approximate_throughput_gain_pct=100*(off["fps"]/on["fps"]-1),
                                normal_p99_ms=on["p99_ms"], off_p99_ms=off["p99_ms"],
                                p99_reduction_ms=on["p99_ms"]-off["p99_ms"],
                                tail_sample="limited" if on["frames"] < 100 else "at least 100 frames",
                                normal_gc_cpu_ms=on["gc_cpu_ms"], normal_gc_assist_ms=on["gc_assist_ms"],
                                normal_peak_runtime_mb=on["peak_runtime_bytes_main"]/2**20,
                                off_peak_runtime_mb=off["peak_runtime_bytes_main"]/2**20)
                paired.append(pair)
            detail = row.get("detail")
            label = f'{axis_label((case.raster, case.raster_gpu), adapters)} / {axis_label((case.filter, case.filter_gpu), adapters)}'
            if detail:
                diagnostics.append(detail_html(label, detail, output))
        title = group_title(group)
        headings = "".join(f"<th>{escape(axis_label(f, adapters))}</th>" for f in filter_axes)
        cells = []
        for r in raster_axes:
            row = [f"<th>{escape(axis_label(r, adapters))}</th>"]
            for f in filter_axes:
                timing, speedup = heat.get((*r, *f), ({}, None))
                if speedup is not None:
                    light = min(65, max(30, 35 + 30 * speedup))
                    hue = 165 if speedup >= 1 else 15
                    row.append(f'<td style="background:hsl({hue} 52% {light}%);color:#0b141b" '
                               f'title="{escape(timing["status"])}">{timing["median_ms"]:.3f} ms<br>{speedup:.2f}×</td>')
                else:
                    row.append(f'<td title="{escape(timing.get("reason", ""))}">{escape(timing.get("status", "—"))}</td>')
            cells.append("<tr>" + "".join(row) + "</tr>")
        sections.append(f"<h2>{escape(title)}</h2><table><thead><tr><th>Raster / FXAA</th>{headings}</tr></thead>"
                        f"<tbody>{''.join(cells)}</tbody></table>{''.join(diagnostics)}")
        for fxaa in (False, True):
            candidates = {case: ms for case, ms in medians.items() if (case.filter != "none") == fxaa}
            if not candidates:
                continue
            winner = min(candidates, key=candidates.get)
            label = f'{axis_label((winner.raster, winner.raster_gpu), adapters)} / {axis_label((winner.filter, winner.filter_gpu), adapters)}'
            print(f"{title} | {label} | {candidates[winner]:.3f} ms")
    write_csv(output / "summary.csv", summary)
    write_csv(output / "repetitions.csv", [row for entries in observations.values() for row in entries])
    write_csv(output / "gc-pairs.csv", paired)
    gc_link = '<a href="gc-pairs.csv">GC pairs</a>' if paired else ""
    page = f"""<!doctype html><html lang="en"><meta charset="utf-8">
<title>Terminal renderer benchmark</title>
<style>
body{{background:#131b25;color:#e6edf6;font:16px system-ui;margin:2rem}} a{{color:#8eddd2}}
table{{border-collapse:collapse;margin-bottom:1rem}} th,td{{border:1px solid #526074;padding:.45rem;text-align:left}}
svg{{display:block;width:min(900px,100%);background:#10151a}} details{{margin:.6rem 0}} summary{{cursor:pointer}}
</style><h1>Terminal renderer benchmark</h1>
<p><a href="summary.csv">Summary</a> · <a href="repetitions.csv">Repetitions</a> ·
<a href="adapters.json">Adapter probes</a> {gc_link}</p>
<p>Heatmaps show median frame work and speed relative to the fastest CPU pipeline in the same FXAA group.
The frame budget is {1000/fps:.2f} ms. Readback combines waiting and transfer. GPU elapsed and worker
round trips overlap the host stage totals. Input and simulation are part of update. All stage columns are milliseconds. Live timing ends at
terminal submission. Paced Switch Room uses Bubble Tea and reports frames it coalesced.
Uncapped Switch Room submits views synchronously to measure throughput.</p>
{''.join(sections)}</html>"""
    (output / "report.html").write_text(page, encoding="utf-8")


def axis_label(axis, adapters):
    mode, adapter = axis
    if not adapter:
        return mode
    gpu = adapters[adapter]
    if sum(other.name == gpu.name for other in adapters.values()) > 1:
        return f"{gpu.name} ({gpu.binary.stem.split('-')[-1]})"
    return gpu.name


def parse_grid(value):
    try:
        w, h = map(int, value.lower().split("x"))
        if 3 <= w <= 4096 and 3 <= h <= 4096:
            return w, h
    except ValueError:
        pass
    raise argparse.ArgumentTypeError("expected WIDTHxHEIGHT between 3 and 4096 cells per axis")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, help="new directory for local results")
    parser.add_argument("--count", type=int, default=3, help="repetitions per case")
    parser.add_argument("--benchtime", default="300ms", help="minimum time per render repetition")
    parser.add_argument("--cpu", type=int, help="Go CPU budget per process (default up to four)")
    parser.add_argument("--tile", type=int, default=32, help="tile width and height for tiled and batched raster")
    parser.add_argument("--frames", type=int, help="fixed measured frames (gameplay defaults to 240)")
    parser.add_argument("--warmup", type=int, default=8)
    parser.add_argument("--fps", type=int, default=60, help="frame budget and deterministic gameplay cadence")
    parser.add_argument("--grid", type=parse_grid, action="append", help="repeatable grid WIDTHxHEIGHT")
    parser.add_argument("--scene", action="append", choices=(*RENDER_SCENES, "switch-room", "fxaa"))
    parser.add_argument("--profile", choices=("render", "gameplay"), default="render")
    parser.add_argument("--pacing", choices=("both", "paced", "uncapped"), default="both")
    parser.add_argument("--viewport", choices=("both", "centered", "full"), default="both")
    parser.add_argument("--color", choices=("auto", "none", "16", "256", "truecolor"), default="truecolor")
    parser.add_argument("--live", action="store_true", help="gameplay only, repeat against the current terminal")
    parser.add_argument("--detail", action="store_true", help="run one extra diagnostic repetition per pipeline")
    parser.add_argument("--trace", action="store_true", help="add Go traces and allocation profiles to diagnostics")
    parser.add_argument("--gc", action="store_true", help="run paired normal and GOGC=off replays")
    parser.add_argument("--memory-mib", type=int, default=512, help="per-process ceiling in the GC experiment")
    parser.add_argument("--gomemlimit", default=os.environ.get("GOMEMLIMIT", "off"), help="soft Go limit for both GC modes")
    parser.add_argument("--allow-software", action="store_true", help="include verified software GL")
    parser.add_argument("--cpu-only", action="store_true", help="skip hardware probing")
    parser.add_argument("--prepare-only", action="store_true", help="build adapter executables and print their paths")
    parser.add_argument("--dry-run", action="store_true", help="probe and save the planned cases, then exit")
    parser.add_argument("--require-renderer", help="limit the GPU sweep to a matching renderer")
    parser.add_argument("--timeout", type=float, default=180, help="seconds allowed per case repetition")
    args = parser.parse_args()
    if min(args.count, args.tile, args.cpu if args.cpu is not None else 1, args.memory_mib, args.timeout) <= 0:
        parser.error("count, tile, cpu, memory-mib and timeout must be positive")
    if args.warmup < 0 or not 1 <= args.fps <= 1000 or (args.frames is not None and args.frames <= 0):
        parser.error("warmup must be nonnegative, fps 1–1000, and frames positive when supplied")
    if args.live and (args.profile != "gameplay" or not sys.stdout.isatty() or args.grid):
        parser.error("--live requires a terminal and --profile gameplay, and measures the current grid")
    if args.profile == "gameplay" and args.scene and "fxaa" in args.scene:
        parser.error("isolated FXAA is a render workload")
    if args.profile == "render" and args.scene and "switch-room" in args.scene:
        parser.error("Switch Room requires --profile gameplay")
    if args.gc and os.environ.get("GOGC", "").casefold() == "off":
        parser.error("set GOGC to a numeric target for the normal half of the GC comparison")
    if args.cpu_only and args.require_renderer:
        parser.error("--cpu-only and --require-renderer select different targets")
    args.frames = args.frames or (240 if args.profile == "gameplay" or args.gc else 0)
    args.detail |= args.trace
    if not shutil.which("go"):
        parser.error("Go 1.24.6 or newer must be on PATH")
    cpu_count = len(os.sched_getaffinity(0)) if hasattr(os, "sched_getaffinity") else os.cpu_count() or 1
    env = {**os.environ, "GOMAXPROCS": str(args.cpu or min(4, cpu_count))}
    for key in ("GRAPHICS_EGL_PCI", "GRAPHICS_CGL_RENDERER", "DRI_PRIME"):
        env.pop(key, None)
    cache = ROOT / ".cache"
    cache.mkdir(exist_ok=True)
    output = args.output.resolve() if args.output else Path(tempfile.mkdtemp(prefix="opengl-bench-", dir=cache))
    print(f"Results directory: {output}", flush=True)
    if args.output:
        output.mkdir(parents=True, exist_ok=False)
    system = platform.system()
    version = run(output / "go-version.txt", ["go", "version"]).strip()
    settings = {**vars(args), "output": str(output), "go": version, "platform": platform.platform(),
                "effective_cpus_per_process": env["GOMAXPROCS"], "normal_gogc": env.get("GOGC", "100"),
                "inherited_gomemlimit": env.get("GOMEMLIMIT", "off")}
    (output / "settings.json").write_text(json.dumps(settings, indent=2), encoding="utf-8")
    binary_dir = cache / "opengl-bin"
    binary_dir.mkdir(exist_ok=True)
    suffix = ".exe" if system == "Windows" else ""
    binary = binary_dir / ("terminal-benchmark" + suffix)
    command = ["go", "build", "-o", str(binary)]
    if not args.cpu_only:
        command += ["-tags", "terminalgl"]
    run(output / "build.log", [*command, "./cmd/terminal-benchmark"])
    inventory = [] if args.cpu_only else discover_adapters(system, binary)
    if args.require_renderer:
        matching = [row for row in inventory if matches(row[1], args.require_renderer)]
        if matching:
            inventory = matching
        elif system != "Darwin" or not any(route.get("GRAPHICS_CGL_RENDERER") for _, _, routes in inventory for route in routes):
            inventory = []
    if not args.cpu_only and (not inventory or args.allow_software):
        inventory.append(("default", "Default OpenGL", [{}]))
    adapters = {}
    for id_, name, routes in inventory:
        digest = hashlib.sha256(id_.encode()).hexdigest()[:12]
        executable = binary_dir / (f"benchmark-gpu-{digest}{suffix}")
        shutil.copyfile(binary, executable)
        executable.chmod(binary.stat().st_mode)
        adapter = Adapter(id_, name, executable, routes)
        adapters[id_] = adapter
        print(f"{adapter.label}: {executable}", flush=True)
    if system == "Windows" and adapters:
        print("Assign each printed executable to its named GPU in Windows Settings → System → Display → Graphics.", flush=True)
    if args.prepare_only:
        return 0
    for adapter in adapters.values():
        ambiguous = sum(gpu.name == adapter.name for gpu in adapters.values()) > 1
        for stage in ("raster", "filter"):
            probe(adapter, stage, output, env, args, ambiguous)
        if adapter.name.startswith("CGL renderer ") and adapter.stages:
            adapter.name = next(iter(adapter.stages.values()))
        print(f"{adapter.label} · raster {adapter.stages.get('raster') or adapter.errors['raster']}"
              f" · FXAA {adapter.stages.get('filter') or adapter.errors['filter']}", flush=True)
    # Include the default route for a verified software driver or if inventory
    # cannot supply a working route. Avoid timing hardware twice.
    fallback = adapters.get("default")
    if fallback and len(adapters) > 1:
        if not fallback.stages or any(other is not fallback and other.stages == fallback.stages
                                      for other in adapters.values()):
            del adapters["default"]
    adapter_report = [dict(vars(gpu), binary=str(gpu.binary)) for gpu in adapters.values()]
    (output / "adapters.json").write_text(json.dumps(adapter_report, indent=2), encoding="utf-8")
    planned = list(dict.fromkeys(matrix(args, adapters.values())))
    write_csv(output / "plan.csv", [
        {**vars(case), "probe_errors": " | ".join(
            adapters[id_].errors[stage] for id_, stage in ((case.raster_gpu, "raster"), (case.filter_gpu, "filter"))
            if id_ and stage in adapters[id_].errors)} for case in planned])
    print(f"Planned {len(planned)} cases, {args.count} repetitions each", flush=True)
    missing_required = args.require_renderer and not any(gpu.stages for gpu in adapters.values())
    if args.dry_run:
        return 1 if missing_required else 0
    results, states = {}, {}
    for i, case in enumerate(planned, 1):
        results[case] = execute_case(case, args, i, adapters, binary, env, output, states)
        if i % 12 == 0 or i == len(planned):
            print(f"Completed {i}/{len(planned)} cases", flush=True)
    write_report(output, results, adapters, args.fps)
    statuses = [row["status"] for entries in results.values() for row in entries]
    print(f'{statuses.count("ok")} successful, {statuses.count("unavailable")} unavailable, '
          f'{statuses.count("failed")} failed, {statuses.count("inconclusive")} inconclusive')
    print(f"Report: {output / 'report.html'}")
    return int(bool(missing_required or "failed" in statuses))


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError) as error:
        sys.exit(f"benchmark_opengl: {error}")
