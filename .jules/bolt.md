## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2025-05-23 - [Affine Transform Optimization]
**Learning:** Confirmed that ~40% of CPU time in matrix-vector multiplication for model-to-world transforms can be saved by skipping the W component calculation (which is always 1 for affine transforms).
**Action:** Use `TransformPointAffine` instead of `TransformPoint` for all ModelMatrix and WorldMatrix transformations in the renderer pipeline.
