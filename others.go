package main

// Lighting scenario setups
// These are kept as they're useful for demo scenes

func setupScenario1(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)
	light1 := NewLight(30, 30, -20, ColorWhite, 1.0)
	ls.AddLight(light1)
	return ls
}

func setupScenario2(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)
	lightRed := NewLight(40, 0, 0, ColorRed, 0.8)
	ls.AddLight(lightRed)
	lightGreen := NewLight(-40, 0, 0, ColorGreen, 0.8)
	ls.AddLight(lightGreen)
	lightBlue := NewLight(0, 40, -20, ColorBlue, 0.6)
	ls.AddLight(lightBlue)
	return ls
}

func setupScenario3(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)
	sun := NewLight(50, 20, -30, ColorOrange, 1.2)
	ls.AddLight(sun)
	sky := NewLight(-30, 10, -20, Color{150, 100, 200}, 0.4)
	ls.AddLight(sky)
	ls.AmbientLight = Color{40, 30, 50}
	ls.AmbientIntensity = 0.2
	return ls
}

func setupScenario4(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)
	moon := NewLight(-20, 50, -40, Color{150, 180, 255}, 0.7)
	ls.AddLight(moon)
	ground := NewLight(0, -30, 0, Color{100, 80, 60}, 0.2)
	ls.AddLight(ground)
	ls.AmbientLight = Color{20, 25, 40}
	ls.AmbientIntensity = 0.15
	return ls
}

func setupScenario5(camera *Camera) *LightingSystem {
	ls := NewLightingSystem(camera)
	keyLight := NewLight(40, 30, -30, ColorWhite, 1.0)
	ls.AddLight(keyLight)
	fillLight := NewLight(-30, 10, -20, Color{200, 200, 220}, 0.4)
	ls.AddLight(fillLight)
	backLight := NewLight(0, 20, 40, Color{220, 230, 255}, 0.6)
	ls.AddLight(backLight)
	return ls
}
