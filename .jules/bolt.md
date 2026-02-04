## 2024-05-22 - [Legacy Field Padding]
**Learning:** Found a legacy `char` byte field in the `Point` struct that was unused but caused the struct to grow from 24 bytes to 32 bytes due to alignment padding.
**Action:** Always check core data structures (like vectors/points) for unused fields or poor alignment, as they are multiplied by the number of vertices.

## 2024-05-23 - [Affine Matrix Optimization]
**Learning:** Matrix transformations (Model-to-World) are often affine (W=1), allowing us to skip the W component calculation and perspective division for a ~40% speedup per vertex.
**Action:** Use `TransformPointAffine` instead of `TransformPoint` whenever the transformation matrix is guaranteed to be affine (like standard translation/rotation/scale matrices).
