## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2024-05-22 - [Scanline Forward Differencing]
**Learning:** Found that replacing per-pixel interpolation (division + lerp) with forward differencing (setup slope + incremental addition) in the software rasterizer loop improved performance by ~2%. Unwrapping struct fields into local variables provided a further speedup by reducing field access overhead in the hot loop.
**Action:** When optimizing hot inner loops in Go, prefer simple scalar variables and incremental addition over struct field access and repeated calculations, especially divisions.
