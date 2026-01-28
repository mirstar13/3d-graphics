## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2024-05-23 - [Software Rasterizer Optimization]
**Learning:** `fillTriangleWithPerPixelLighting` was performing per-pixel floating point division and multiple linear interpolations, which is costly for CPU rasterization.
**Action:** Replaced with forward differencing (incremental addition) for gradients (InvZ, Point, UV). Benchmark showed ~30% improvement.
