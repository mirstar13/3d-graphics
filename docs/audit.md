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
across single, tile, job, and batched rendering, and all six curated presets rendered
through single-threaded and tiled strategies. Preset checks now use their actual
lighting and shadow policy, several animation times, manual and orbit cameras,
and monochrome output. Interactive checks exercise normal application startup
and frame preparation.

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

## Curated scene revision

The 13 overlapping technical examples were replaced by six focused scenes;
README documents the lineup and changed numeric selection. A shared catalog
owns camera framing, descriptions, lighting policy, construction, and animation.
Animation poses are evaluated from absolute time. Geometry budgets, visible
coverage, CPU parity, monochrome output, courtyard shadows, and runway LOD
changes have regression coverage. Framebuffer previews are exportable for review.

Visual review exposed two additional defects: sphere triangle winding pointed
inward, and TexturedMaterial inherited a solid-color SampleDiffuse method.
Sphere faces now point outward and enabled texture maps are actually sampled.
Dedicated tests verify both corrections.

## Fixed-step simulation foundation

A session-owned physics world now drives animation, lights, node updates, and
LOD transitions at 60 Hz. Cameras and rendering retain frame cadence. Catch-up
is bounded by 250 ms accepted elapsed time and eight ticks per frame, with
explicit discarded-time diagnostics. Gravity integration is available and
unit-tested; demo objects are not yet rigid bodies.

Space pauses, Enter resumes, and N single-steps while paused. Ordered command
queues drain independently of movement-key clearing. Queue limits and dropped
commands are observable. Resume clears pending steps and skips paused elapsed
time; rendering and AA passes cannot advance the world. Paused help is shown
even with debug statistics disabled. See physics-world-plan.md for ownership,
API contracts, test coverage, and remaining rigid-body/collider milestones.

## Rigid-body milestone

Quaternion vector rotation used normalized quaternion multiplication, losing vector
magnitude. The corrected cross-product formula matches rotation matrices and
preserves zero vectors, collider dimensions, and inertia calculations.

Physics coverage includes primitive contacts, resting stacks, moving platforms,
angular integration, deterministic render-rate independence, sensors and handles,
static-mesh BVH queries, rotated terrain, and coplanar triangle seams. Seam handling
groups coplanar contacts to avoid sideways impulses from internal triangle edges.
The seventh preset adds a physics playground; Linux PTY checks exercise C, pause,
single-step, and clean exit. Race tests, vet, staticcheck, and Windows/macOS cross-
builds pass; native Windows/macOS interaction still needs validation.

Initial physics-only benchmarks on Linux amd64, Intel Xeon Platinum 8573C:

| Spheres | Time per tick | Allocated bytes per tick | Allocations per tick |
| --- | --- | --- | --- |
| 16 | 0.075 ms | 53,552 | 154 |
| 64 | 0.367 ms | 214,641 | 462 |
| 128 | 0.957 ms | 431,122 | 856 |

These fixtures exclude rendering and are not hardware-independent performance
promises. Broadphase remains quadratic; reusable contact buffers and spatial
partitioning are candidates if larger scenes justify them.

## Physics stability baseline

The five-box, 30-second stack regression exposed collapse with the original
10 velocity / 4 position iterations. Defaults are now 20 / 8. Box SAT keeps a
supporting face when another face improves depth by less than 0.1 mm, or an edge
axis improves it by less than 1 mm; separation rejection is unchanged. This avoids
unstable single-edge manifolds for nearly parallel supporting faces.

Without sleeping, equal-mass boxes remain stacked with maximum final-window
penetration 8.4 mm and speed below 0.001 m/s. A 10-times-heavier bottom box remains
stacked with penetration below 7.2 mm and speed below 0.06 m/s. These are measured
fixture bounds, not guarantees for arbitrary stacks or mass ratios. Initial sphere
overlap recovery injects no velocity; an impossible kinematic crushing gap remains
finite but cannot satisfy both surfaces.

Broad-phase traversal, cache invalidation, and contact events now live in separate
files. Solver metrics include all enabled pair visits across detection passes and
phase timings; timings never influence simulation. Sparse/dense/settled benchmarks
retain the brute-force baseline for later optimization comparisons.

## Sleeping and sphere CCD

Resting contact islands sleep after 0.5 quiet simulation seconds, with trial limits
0.03 m/s and 0.05 rad/s and maximum 15 mm contact penetration. Static/kinematic
supports do not bridge separate dynamic islands. Wake regressions cover forces,
impulses, edits, removed support, new contacts, and moving platforms. Sleep state
is public in body snapshots and terminal diagnostics; existing stack stability
fixtures explicitly disable sleeping.

Opt-in sphere CCD sweeps static planes, boxes, and mesh triangles with conservative
closest-point advancement, including rounded corners and triangle edges. Bodies
advance together within the fixed tick; forces, hooks, sleep timers, and history
advance once. CCD solves deliberately skip warm starting instead of reapplying
full-tick impulses at unequal subintervals. Transient Begin/End events are retained.
The default eight-impact budget conservatively freezes CCD motion for the remaining
interval on exhaustion, exposed through CCDClamps. The sweep convergence limit can
also cause conservative early stops. Sensors and unsupported pairs remain discrete.
Tests cover 600 m/s thin-wall crossings, bounce, rounded-corner misses, transient
contacts, force lifetime, multiple triangles in one mesh, and budget exhaustion.

## Broad-phase and buffer reuse

Sweep-and-prune is now the default; the brute-force mode remains available through
physics options as a test oracle. Randomized candidate tests include rotated boxes,
filters, sensors, removals, slot reuse, and teleports. Identical CCD simulations
produce matching body/contact/event snapshots with both implementations. Published
snapshots remain independent of reused world scratch buffers.

With 20 benchmark ticks on the reference machine, 128 sparse bodies took about
0.18 ms/tick and 1,152 pair visits; 128 settled bodies took 0.31 ms/tick and the same
pair visits. The original milestone-1 baseline visited 82,560 pairs and took about
0.91/1.74 ms respectively. Those comparisons include the added sleeping behavior
and skipping empty position solves, not only the broad-phase change. Settled
allocations fell from roughly 735 KB to 86 KB per tick. Dense overlapping 128-body
fixtures still cost about 4.7 ms/tick; contact generation/island work remains a
cost, and broad-phase acceleration is not a guarantee for dense scenes.

## Joint constraints

Distance, ball-and-socket, and hinge rows share velocity/position iterations with
contacts. World-local generation handles are invalidated on removal/reset;
attached joints are removed with their body. Joint connectivity participates in
wake/sleep traversal without merging islands through static supports. Hinge axes
must initially align; signed limits lie within -pi..pi relative to creation pose.
Motors clamp accumulated angular impulse to torque times solve interval. Spring
compliance and breakage remain separate extensions.

Regression fixtures cover a 10-second pendulum energy/length bound, hanging-chain
anchor error and wakeup, kinematic attachments, hinge-axis drift, torque/angle
limits, connected-body filtering, and deferred removal. The playground adds a
small pendulum and presentation-only link, staying within its geometry budget.

## Compound primitive bodies

Immutable compound colliders accept up to 32 sphere/box children with explicit
local poses and relative mass weights. Construction computes the mass centre,
rotates each child tensor, and adds parallel-axis contributions. Dynamics now use
full symmetric 3x3 inertia/inverse-inertia tensors; the existing diagonal snapshot
remains for inspection. Overlapping children contribute additive mass, so authored
weights describe the intended distribution rather than an inferred union volume.

Each child retains its own contact manifold and feature namespace; compounds do
not discard distinct supports through a global four-contact reduction. Debug
outlines include children. Application bindings carry an explicit body-to-visual
offset, including interpolation and joint-link anchor conversion. Tests cover
analytic two-mass inertia, rotated off-diagonal terms, tensor inversion, mass
changes, off-centre impulses, joint anchors, mesh contacts, and immutable snapshots.
The playground adds an asymmetrically weighted dumbbell. Compound CCD is rejected.
