## 2024-10-24 - Scanline Rasterization Inner Loop Overhead
**Learning:** The software rasterizer was performing full linear interpolation (subtraction, division, multiplication) for every pixel in the inner scanline loop. This is a classic bottleneck. Standard incremental addition (Forward Differencing) is much faster for linear gradients.
**Action:** Always check inner rasterization loops for invariant calculations that can be moved out or converted to incremental updates.
