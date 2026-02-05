## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2026-02-05 - [Rasterizer Forward Differencing]
**Learning:** The terminal renderer's inner rasterization loop was using per-pixel division and full linear interpolation calls, creating significant overhead.
**Action:** Replaced with Forward Differencing to compute attributes incrementally, removing division from the hot path.
