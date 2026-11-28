import argparse
from contextlib import ExitStack, nullcontext
import sys
import tempfile
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

from benchmark_opengl import Adapter, matrix, matches, probe, windows_preference_routes, windows_wgl_routes


class BenchmarkMatrixTests(unittest.TestCase):
    def test_full_matrix_uses_each_adapter_per_stage(self):
        args = argparse.Namespace(profile="render", grid=[(160, 60)], scene=["physics"])
        for count, expected in ((0, 8), (1, 15), (2, 24)):
            with self.subTest(adapters=count):
                adapters = [
                    Adapter(str(i), "Same GPU Model", Path("benchmark"), [{}])
                    for i in range(count)
                ]
                cases = list(matrix(args, adapters))
                self.assertEqual(len(set(cases)), expected)
                self.assertEqual(sum(not c.raster_gpu and not c.filter_gpu for c in cases), 8)
                self.assertEqual(sum(bool(c.raster_gpu and c.filter_gpu) for c in cases), count**2)
                if count == 2:
                    self.assertIn(("0", "1"), {(c.raster_gpu, c.filter_gpu) for c in cases})
                    self.assertIn(("1", "0"), {(c.raster_gpu, c.filter_gpu) for c in cases})

    def test_windows_probe_keeps_verified_helper_for_each_stage(self):
        adapter = Adapter("pci-a", "AMD Radeon Graphics", Path("default.exe"), [{}])
        args = argparse.Namespace(require_renderer=None, allow_software=False)
        with patch("benchmark_opengl.tool_json", return_value=["amd:7", "invalid"]):
            routes = windows_wgl_routes(Path("minimum-power.exe"), {})
        self.assertEqual([env["GRAPHICS_WGL_GPU"] for _, env in routes], ["amd:7", ""])
        routes.insert(0, (adapter.binary, {"GRAPHICS_WGL_GPU": ""}))
        observed = ['{"driver":"NVIDIA GPU"}', '{"driver":"AMD Radeon(TM) Graphics"}']
        with patch("benchmark_opengl.run", side_effect=observed):
            probe(adapter, "filter", Path("."), {}, args, False, routes)
        self.assertEqual(adapter.stage_binaries["filter"], "minimum-power.exe")
        self.assertEqual(adapter.verified_env["filter"], {"GRAPHICS_WGL_GPU": "amd:7"})
        self.assertEqual(adapter.stages["filter"], "AMD Radeon(TM) Graphics")

    def test_windows_preferences_are_confined_to_temporary_helpers(self):
        registry = Mock()
        registry.CreateKeyEx.return_value = nullcontext("key")
        with tempfile.TemporaryDirectory() as directory:
            original = Path(directory) / "benchmark.exe"
            original.write_bytes(b"fixture")
            with patch.dict(sys.modules, winreg=registry), patch("benchmark_opengl.tool_json", return_value=[]):
                with ExitStack() as cleanup:
                    routes = windows_preference_routes(original, {}, cleanup)
                    helpers = [path for path, _ in routes]
                    self.assertEqual([p.stem for p in helpers], ["minimum-power", "high-performance"])
                    self.assertTrue(all(p.is_file() and p != original for p in helpers))
                    assigned = [call.args for call in registry.SetValueEx.call_args_list]
                    self.assertEqual([a[1] for a in assigned], list(map(str, helpers)))
                    self.assertEqual([a[-1] for a in assigned], ["GpuPreference=1;", "GpuPreference=2;"])
                removed = [call.args[1] for call in registry.DeleteValue.call_args_list]
                self.assertCountEqual(removed, list(map(str, helpers)))
                self.assertEqual(list(original.parent.iterdir()), [original])

    def test_isolated_fxaa_does_not_repeat_raster(self):
        args = argparse.Namespace(profile="render", grid=[(160, 60)], scene=["fxaa"])
        gpu = Adapter("nvidia", "NVIDIA", Path("benchmark"), [{}])
        cases = list(matrix(args, [gpu]))
        self.assertEqual({(c.raster, c.filter) for c in cases}, {("single", "cpu"), ("single", "gl")})

    def test_renderer_match_accounts_for_vendor_naming(self):
        self.assertTrue(matches("NVIDIA GeForce RTX 4050 Laptop GPU/PCIe/SSE2", "RTX 4050"))
        self.assertTrue(matches("AMD Radeon(TM) Graphics", "AMD Radeon Graphics"))
        self.assertFalse(matches("AMD Radeon(TM) Graphics", "RTX 4050"))


if __name__ == "__main__":
    unittest.main()
