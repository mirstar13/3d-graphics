import argparse
from pathlib import Path
import unittest

from benchmark_opengl import Adapter, matrix, matches


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
