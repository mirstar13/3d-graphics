package demo

import (
	"fmt"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
)

const (
	DemoBasicGeometry = iota
	DemoMeshGenerators
	DemoLightingShowcase
	DemoMaterialShowcase
	DemoTransformHierarchy
	DemoLODSystem
	DemoSpatialPartitioning
	DemoCollisionPhysics
	DemoAdvancedRendering
	DemoPerformanceTest
	DemoAdvancedFeatures
	DemoTextureShowcase
	DemoShadowMapping
)

// Names lists scene presets in the same order as the demo identifiers.
var Names = []string{
	"Basic geometry", "Mesh generators", "Lighting showcase", "Material showcase",
	"Transform hierarchy", "LOD system", "Spatial partitioning", "Collision and physics",
	"Advanced rendering", "Performance test", "Advanced features", "Texture showcase", "Shadow mapping",
}

func ConfigureCamera(camera *engine.Camera, demoType int, orientation engine.OrientationType) {
	camera.DZ = 0.0
	camera.Near = 0.5
	camera.FOV = engine.Point{X: 60.0, Y: 30.0, Z: 0}

	switch demoType {
	case DemoBasicGeometry:
		camera.Transform.SetPosition(0, 5, -40*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoMeshGenerators:
		camera.Transform.SetPosition(0, 10, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoLightingShowcase:
		camera.Transform.SetPosition(0, 15, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoMaterialShowcase:
		camera.Transform.SetPosition(0, 12, -60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 250.0

	case DemoTransformHierarchy:
		camera.Transform.SetPosition(0, 25, -70*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoLODSystem:
		camera.Transform.SetPosition(0, 50, -150*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0

	case DemoSpatialPartitioning:
		camera.Transform.SetPosition(0, 40, -100*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoCollisionPhysics:
		camera.Transform.SetPosition(0, 30, -70*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0

	case DemoAdvancedRendering:
		camera.Transform.SetPosition(0, 20, -50*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoPerformanceTest:
		camera.Transform.SetPosition(0, 50, -130*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 500.0

	case DemoAdvancedFeatures:
		camera.Transform.SetPosition(0, 20, -80*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 400.0

	case DemoTextureShowcase:
		camera.Transform.SetPosition(0, 0, 60*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	case DemoShadowMapping:
		camera.Transform.SetPosition(0, 15, 40*float64(orientation))
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 200.0

	default:
		camera.Transform.SetPosition(0, 10, -60)
		camera.Transform.SetRotation(0, 0, 0)
		camera.Far = 300.0
	}
}

func ConfigureCameraController(controller *engine.CameraController, demoType int) {
	switch demoType {
	case DemoBasicGeometry:
		controller.SetOrbitRadius(50.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(10.0)

	case DemoMeshGenerators:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoLightingShowcase:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)

	case DemoMaterialShowcase:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoTransformHierarchy:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(30.0)

	case DemoLODSystem:
		controller.SetOrbitRadius(180.0)
		controller.SetOrbitCenter(0, 0, -40)
		controller.SetOrbitHeight(50.0)

	case DemoSpatialPartitioning:
		controller.SetOrbitRadius(120.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(40.0)

	case DemoCollisionPhysics:
		controller.SetOrbitRadius(90.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(30.0)

	case DemoAdvancedRendering:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, -50)
		controller.SetOrbitHeight(25.0)

	case DemoPerformanceTest:
		controller.SetOrbitRadius(150.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(50.0)

	case DemoAdvancedFeatures:
		controller.SetOrbitRadius(100.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(25.0)

	case DemoTextureShowcase:
		controller.SetOrbitRadius(70.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(15.0)

	case DemoShadowMapping:
		controller.SetOrbitRadius(60.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)

	default:
		controller.SetOrbitRadius(80.0)
		controller.SetOrbitCenter(0, 0, 0)
		controller.SetOrbitHeight(20.0)
	}

	controller.EnableAutoOrbit(true)
}

func SetupLighting(camera *engine.Camera, demoType int) *engine.LightingSystem {
	lighting := GetLightingScenario(demoType, camera)
	scenarioName := GetLightingScenarioName(demoType)
	fmt.Printf("Lighting: %s\n", scenarioName)
	return lighting
}

func BuildScene(scene *engine.Scene, demoType int) {
	switch demoType {
	case DemoBasicGeometry:
		BasicGeometryDemo(scene)
	case DemoMeshGenerators:
		MeshGeneratorsDemo(scene)
	case DemoLightingShowcase:
		LightingShowcaseDemo(scene)
	case DemoMaterialShowcase:
		MaterialShowcaseDemo(scene)
	case DemoTransformHierarchy:
		TransformHierarchyDemo(scene)
	case DemoLODSystem:
		LODSystemDemo(scene)
	case DemoSpatialPartitioning:
		SpatialPartitioningDemo(scene)
	case DemoCollisionPhysics:
		CollisionPhysicsDemo(scene)
	case DemoAdvancedRendering:
		AdvancedRenderingDemo(scene)
	case DemoPerformanceTest:
		PerformanceTestDemo(scene)
	case DemoAdvancedFeatures:
		AdvancedFeaturesDemo(scene)
	case DemoTextureShowcase:
		TextureShowcaseDemo(scene)
	case DemoShadowMapping:
		ShadowMappingDemo(scene)
	default:
		BasicGeometryDemo(scene)
	}
}

func AnimateSceneDemo(scene *engine.Scene, demoType int, time, dt float64) {
	switch demoType {
	case DemoBasicGeometry:
		AnimateBasicGeometry(scene, dt)
	case DemoMeshGenerators:
		AnimateMeshGenerators(scene, dt)
	case DemoLightingShowcase:
		AnimateLightingShowcase(scene, time, dt)
	case DemoMaterialShowcase:
		AnimateMaterialShowcase(scene, dt)
	case DemoTransformHierarchy:
		AnimateTransformHierarchy(scene, time, dt)
	case DemoLODSystem:
		AnimateLODSystem(scene, dt)
	case DemoSpatialPartitioning:
		AnimateSpatialPartitioning(scene, time, dt)
	case DemoCollisionPhysics:
		AnimateCollisionPhysics(scene, time, dt)
	case DemoAdvancedRendering:
		AnimateAdvancedRendering(scene, dt)
	case DemoPerformanceTest:
		AnimatePerformanceTest(scene, dt)
	case DemoAdvancedFeatures:
		AnimateAdvancedFeatures(scene, time, dt)
	case DemoTextureShowcase:
		AnimateAdvancedFeatures(scene, time, dt) // Reuse advanced features animation
	case DemoShadowMapping:
		AnimateAdvancedFeatures(scene, time, dt) // Reuse animation for shadow casters
	}

}
