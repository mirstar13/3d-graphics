# Code audit

This pass covers camera math, transform caching, geometry loading and clipping,
scene visibility, terminal scheduling, and the application menus. Regression
tests reproduce the corrected failures. The checks cover selected inputs and
scene presets. They do not establish correctness for every engine feature.

## Corrected behavior

| Area | Finding and correction | Regression coverage |
| --- | --- | --- |
| Projection and picking | FOV was used as a pixel scale in projection and as an angle in culling. Projection, picking rays, visibility, and LOD coverage now use degree-based angles. | TestProjectionFrustumAndRayAgree |
| Frustum planes | Two implementations disagreed, and side-plane normals used incorrect trigonometric components. Both entry points now use one implementation with inward unit normals. | TestProjectionFrustumAndRayAgree |
| Transform caches | Children retained stale world and inverse matrices after parent movement. Local caches could also remain stale when the world cache was dirty. Cache refresh now checks parent state and consistently invalidates local changes. | TestParentTransformInvalidatesChild |
| Camera orientation | LookAt built a reflected basis. Euler pitch and yaw mapped to incorrect axes. LookAt and Euler conversion now follow +Z-forward and X/Y/Z pitch/yaw/roll conventions. | TestCameraLookAtAndRotationAxes |
| Transform composition | Nonuniform scale multiplied matrix rows, applying scale after rotation. Composition now applies local scale before rotation. | TestCompositionScalesLocalAxes |
| Child culling | Rejecting a parent's own geometry skipped independently visible descendants. Children are now traversed separately. | TestCullingRetainsVisibleChildOfCulledObject |
| Mesh simplification | Independent vertex/index sampling produced invalid indices and lost UVs. Simplification now samples complete triangles and remaps referenced vertices and attributes. | TestSimplifyMeshKeepsValidTriangles |
| Bounds | Scene and frustum code duplicated bounds calculations. Mesh offsets and primitives now share one world-space calculation. The historical optimal-OBB helper now encloses uneven point clouds correctly. | TestMeshOffsetMatchesTransformedVertices, TestFittedBoundsEncloseUnevenPointCloud |
| Parallel primitives | Jobs omitted quads, circles, and instances. Tile bounds could shrink primitives to their origin. All strategies now use the terminal primitive dispatcher and geometry bounds. | TestParallelPrimitiveParity |
| Instance materials | Rendering temporarily mutated the shared base mesh. Workers now use a local mesh copy for each material override. | TestParallelPrimitiveParity under the race detector |
| Pooled triangles | Texture flags and material references survived reuse. Copy helpers omitted UV attributes. Pool release resets complete triangles and copies preserve attributes. | TestPooledTriangleAndCopiesClearStaleAttributes |
| Near clipping | Clipping dropped UVs and could dereference a missing normal. Polygon clipping now preserves metadata and interpolates UVs at intersections. | TestClippingPreservesTextureAttributes |
| OBJ loading | Parsed UVs were discarded, relative indices failed, and malformed attribute indices were silently accepted. Faces now preserve UV seams and validate absolute/relative indices. | TestOBJPreservesUVsAndRelativeIndices |
| Points and lines | Color-mode points retained a black foreground, far-plane points rendered, and zero-length lines disappeared. Points now receive a visible color and respect depth limits. Visible degenerate lines produce a cell. Line depth interpolation uses inverse depth. | TestPointVisibilityAndDegenerateLine |
| LOD presets | The primitive dispatcher omitted LOD groups. It now draws their selected mesh. | TestScenePresetsRender |

Changing FOV and rotation conventions corrects visible behavior. Existing YAML
FOV values now represent angles. Scenes may have different framing and
orientation from earlier builds.

## Structure and cleanup

The renderer strategy interface and scheduling/filter decorators remain the
extension points. Shared frame preparation resolves lazy caches and shadow
state before workers start. Tiles use local clipping state, and the job
renderer serializes framebuffer writes.

Duplicate frustum builders, bounds calculations, tile scheduling, primitive
dispatch, LOD coverage, and TUI viewport clipping now share implementations.
Package comments describe lifecycle, units, and threading assumptions.

Removed code includes uncalled private math/raster/raycast helpers, GPU cache
types with no callers in the terminal branch, an unimplemented parallel-culling
stub, unused experimental scanline/transform-worker entry points, unused SSAA
buffer allocations and sample patterns, commented-out example code, and an
unreachable statistics branch. Historical GPU implementations are available
in the existing local renderer branches.

## Remaining platform validation

Native Windows and macOS interactive console checks require those operating
systems, which are unavailable in this Linux workspace. See
[terminal compatibility](terminal-compatibility.md) for the test procedure.
Cross-compilation is build coverage only.

## Reproducing the checks

From the repository root, with Go 1.24.6 or later:

    go test -race -count=1 ./...
    go vet ./...
    go doc ./internal/engine
    go doc ./internal/engine.Camera.ProjectPoint
    go doc ./internal/terminal
    go doc ./internal/config

Static analysis for this pass uses Staticcheck 2025.1.1:

    go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
    staticcheck ./...

The race suite includes all 20 CPU-mode/AA combinations, a framebuffer comparison
across single, tile, job, and batched rendering, and all 13 presets rendered
through single-threaded and tiled strategies. The preset checks disable shadow
generation to isolate geometry rendering. Interactive checks exercise normal
application startup and frame preparation.

## Timing, resizing, and normals follow-up

Camera movement, orbit, demo rotation, and light rotation now scale with measured
elapsed seconds, retaining the previous 60 FPS reference speed. Scene updates
receive measured elapsed time. Terminal dimensions are checked between frames;
configured viewport dimensions remain upper limits, while zero follows the
window. Buffers and clipping are replaced together; filters and tiles read the
new dimensions each frame. Custom triangle and quad normals use normalized
inverse-transpose transformation for nonuniform scale.

## Materials, anti-aliasing, and LOD follow-up

- OBJ loading now assigns materials per triangulated face and merges referenced
  material libraries. Unknown materials and missing libraries return errors.
  Rendering, indexed LOD sampling, and clustering preserve face assignments.
  QEM uses indexed sampling for material-bearing meshes until edge collapse
  can preserve material boundaries. Texture-map loading and OBJ material export
  remain outside this import change.
- Projection retains floating-point coordinates for triangle rasterization.
  MSAA uses independent coverage and depth at 2 or 4 subpixel positions, with
  one lighting evaluation per visible triangle/cell. Points and wireframe lines
  retain full-cell coverage. SSAA shades at twice each dimension and downsamples
  colors and monochrome glyphs, retaining nearest covered depth. Buffers are
  reused across frames and replaced on resize. FXAA remains a color filter.
- LOD transition state now completes without restarting. Scene updates advance
  transitions; terminal rendering uses complementary ordered-dither masks to
  avoid overlap depth conflicts. Culling bounds enclose both meshes. Compatible
  morph topology interpolates vertices; incompatible topology cross-fades.
  Shadow maps switch geometry at the fade midpoint, without shadow blending.
- The LOD preset now exercises transitions. Regression tests cover materials,
  fractional AA coverage, depth ordering, CPU-mode parity, transition endpoints,
  completion/reversal, and zero-duration transitions. The reusable PTY script
  covers menu/settings/restart/exit, resize, color modes, and input restoration.

Run the AA microbenchmark with:

    go test ./internal/terminal -run '^$' -bench BenchmarkAntiAliasing -benchmem

This isolates a filled triangle at 100x40 cells with shadows disabled; it is
not a prediction of performance for complex scenes or terminal output.

One Linux x86-64 run on an Intel Xeon Platinum 8573C measured approximately
85 µs/frame without AA, 240 µs for FXAA, 155 µs for MSAA2, 229 µs for MSAA4,
and 434 µs for SSAA in this microbenchmark. SSAA scratch-buffer reuse reduced
steady-state allocation count to the same three small geometry allocations as
the unfiltered path. These figures vary with hardware, geometry, and lighting.
