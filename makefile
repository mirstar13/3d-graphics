shaders:
	@mkdir -p shaders/compiled
	glslc shaders/vertex.vert -o shaders/compiled/vertex.spv
	glslc shaders/fragment.frag -o shaders/compiled/fragment.spv

.PHONY: shaders