## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2024-05-23 - [Affine Transform Optimization]
**Learning:** `TransformPoint` performs perspective division even for affine Model-to-World matrices (where W=1), costing unnecessary cycles in hot vertex loops.
**Action:** Use `TransformPointAffine` (skipping W calc/division) for known affine transformations like World Matrices to save ~30% per vertex transform.
