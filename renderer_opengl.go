package main

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// OpenGLRenderer renders using OpenGL 4.1
type OpenGLRenderer struct {
	window        *glfw.Window
	width         int
	height        int
	renderContext *RenderContext

	// OpenGL resources
	program      uint32
	lineProgram  uint32
	pbrProgram   uint32 // PBR shader program
	vao          uint32
	vbo          uint32
	lineVAO      uint32
	lineVBO      uint32
	pbrVAO       uint32 // Separate VAO for PBR rendering
	pbrVBO       uint32 // Separate VBO for PBR rendering
	uniformModel int32
	uniformView  int32
	uniformProj  int32

	// Line shader uniforms
	lineUniformModel int32
	lineUniformView  int32
	lineUniformProj  int32

	// PBR shader uniforms
	pbrUniformModel       int32
	pbrUniformView        int32
	pbrUniformProj        int32
	pbrUniformMetallic    int32
	pbrUniformRoughness   int32
	pbrUniformAlbedo      int32
	pbrUniformCameraPos   int32
	pbrUniformLightPos    int32
	pbrUniformLightColor  int32
	pbrUniformLightSpaceMatrix int32
	pbrUniformShadowMap   int32
	pbrUniformUseShadows  int32

	// Texture support
	textureProgram        uint32
	textureVAO            uint32
	textureVBO            uint32
	textureUniformModel   int32
	textureUniformView    int32
	textureUniformProj    int32
	textureUniformSampler int32
	textureUniformUseTexture int32
	texturedVertices      []TexturedVertex
	textureCache          map[*Texture]uint32 // Cache OpenGL texture IDs
	activeTexture         *Texture             // Current texture being rendered

	// Shadow mapping support
	shadowProgram         uint32   // Depth-only shader for shadow pass
	shadowFBO             uint32   // Framebuffer for shadow map
	shadowDepthTexture    uint32   // Depth texture
	shadowResolution      int      // Shadow map resolution (e.g., 2048)
	shadowUniformModel    int32
	shadowUniformLightSpaceMatrix int32
	enableShadows         bool
	shadowLightMatrix     Matrix4x4 // Light space transformation matrix

	// Vertex data
	maxVertices     int
	currentVertices []VulkanVertex // Interleaved: pos(3) + color(3)
	pbrVertices     []PBRVertex    // Interleaved: pos(3) + normal(3) + color(3)
	lineVertices    []float32
	usePBRPath      bool // Flag to use PBR rendering path

	// Settings
	UseColor       bool
	ShowDebugInfo  bool
	LightingSystem *LightingSystem
	Camera         *Camera

	// Clipping (not used in OpenGL but required by interface)
	clipMinX, clipMinY, clipMaxX, clipMaxY int

	vboCache *MeshBufferCache

	initialized bool
	frameCount  int
}

// PBRVertex represents a vertex with position, normal, and color for PBR rendering
type PBRVertex struct {
	Pos    [3]float32
	Normal [3]float32
	Color  [3]float32
}

// TexturedVertex represents a vertex with position, UV, and color
type TexturedVertex struct {
	Pos   [3]float32
	UV    [2]float32
	Color [3]float32
}

const (
	vertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aColor;

out vec3 FragColor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;

void main() {
    gl_Position = proj * view * model * vec4(aPos, 1.0);
    FragColor = aColor;
}
` + "\x00"

	fragmentShaderSource = `
#version 410 core
in vec3 FragColor;
out vec4 color;

void main() {
    color = vec4(FragColor, 1.0);
}
` + "\x00"

	lineVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aColor;

out vec3 FragColor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;

void main() {
    gl_Position = proj * view * model * vec4(aPos, 1.0);
    FragColor = aColor;
}
` + "\x00"

	lineFragmentShaderSource = `
#version 410 core
in vec3 FragColor;
out vec4 color;

void main() {
    color = vec4(FragColor, 1.0);
}
` + "\x00"

	pbrVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aNormal;
layout (location = 2) in vec3 aColor;

out vec3 FragPos;
out vec3 Normal;
out vec3 BaseColor;
out vec4 FragPosLightSpace;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;
uniform mat4 lightSpaceMatrix;

void main() {
    vec4 worldPos = model * vec4(aPos, 1.0);
    FragPos = worldPos.xyz;
    Normal = mat3(transpose(inverse(model))) * aNormal;
    BaseColor = aColor;
    FragPosLightSpace = lightSpaceMatrix * worldPos;
    gl_Position = proj * view * worldPos;
}
` + "\x00"

	pbrFragmentShaderSource = `
#version 410 core
in vec3 FragPos;
in vec3 Normal;
in vec3 BaseColor;
in vec4 FragPosLightSpace;

out vec4 FragColor;

uniform vec3 cameraPos;
uniform vec3 lightPos;
uniform vec3 lightColor;
uniform float metallic;
uniform float roughness;
uniform vec3 albedo;
uniform sampler2D shadowMap;
uniform bool useShadows;

const float PI = 3.14159265359;

// Simplified PBR (Cook-Torrance BRDF)
vec3 fresnelSchlick(float cosTheta, vec3 F0) {
    return F0 + (1.0 - F0) * pow(1.0 - cosTheta, 5.0);
}

float distributionGGX(vec3 N, vec3 H, float roughness) {
    float a = roughness * roughness;
    float a2 = a * a;
    float NdotH = max(dot(N, H), 0.0);
    float NdotH2 = NdotH * NdotH;
    
    float nom = a2;
    float denom = (NdotH2 * (a2 - 1.0) + 1.0);
    denom = PI * denom * denom;
    
    return nom / max(denom, 0.001);
}

float geometrySchlickGGX(float NdotV, float roughness) {
    float r = (roughness + 1.0);
    float k = (r * r) / 8.0;
    
    float nom = NdotV;
    float denom = NdotV * (1.0 - k) + k;
    
    return nom / max(denom, 0.001);
}

float geometrySmith(vec3 N, vec3 V, vec3 L, float roughness) {
    float NdotV = max(dot(N, V), 0.0);
    float NdotL = max(dot(N, L), 0.0);
    float ggx2 = geometrySchlickGGX(NdotV, roughness);
    float ggx1 = geometrySchlickGGX(NdotL, roughness);
    
    return ggx1 * ggx2;
}

float shadowCalculation(vec4 fragPosLightSpace) {
    if (!useShadows) {
        return 1.0;
    }
    
    // Perspective divide
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    
    // Transform to [0,1] range
    projCoords = projCoords * 0.5 + 0.5;
    
    // Outside shadow map check
    if (projCoords.z > 1.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || 
        projCoords.y < 0.0 || projCoords.y > 1.0) {
        return 1.0;
    }
    
    // Get closest depth from shadow map
    float closestDepth = texture(shadowMap, projCoords.xy).r;
    float currentDepth = projCoords.z;
    
    // Shadow bias to prevent shadow acne
    float bias = 0.005;
    
    // PCF (Percentage Closer Filtering)
    float shadow = 0.0;
    vec2 texelSize = 1.0 / textureSize(shadowMap, 0);
    for(int x = -1; x <= 1; ++x) {
        for(int y = -1; y <= 1; ++y) {
            float pcfDepth = texture(shadowMap, projCoords.xy + vec2(x, y) * texelSize).r;
            shadow += currentDepth - bias > pcfDepth ? 0.0 : 1.0;
        }
    }
    shadow /= 9.0;
    
    return shadow;
}

void main() {
    vec3 N = normalize(Normal);
    vec3 V = normalize(cameraPos - FragPos);
    
    // Use albedo uniform if non-zero, otherwise use vertex color
    vec3 materialAlbedo = (albedo.r + albedo.g + albedo.b > 0.01) ? albedo : BaseColor;
    
    // Calculate reflectance at normal incidence
    vec3 F0 = vec3(0.04); 
    F0 = mix(F0, materialAlbedo, metallic);
    
    // Lighting calculation
    vec3 L = normalize(lightPos - FragPos);
    vec3 H = normalize(V + L);
    float distance = length(lightPos - FragPos);
    float attenuation = 1.0 / (distance * distance * 0.01);
    vec3 radiance = lightColor * attenuation;
    
    // BRDF
    float NDF = distributionGGX(N, H, roughness);
    float G = geometrySmith(N, V, L, roughness);
    vec3 F = fresnelSchlick(max(dot(H, V), 0.0), F0);
    
    vec3 numerator = NDF * G * F;
    float denominator = 4.0 * max(dot(N, V), 0.0) * max(dot(N, L), 0.0);
    vec3 specular = numerator / max(denominator, 0.001);
    
    vec3 kS = F;
    vec3 kD = vec3(1.0) - kS;
    kD *= 1.0 - metallic;
    
    float NdotL = max(dot(N, L), 0.0);
    
    // Calculate shadow
    float shadow = shadowCalculation(FragPosLightSpace);
    
    vec3 Lo = (kD * materialAlbedo / PI + specular) * radiance * NdotL * shadow;
    
    // Ambient
    vec3 ambient = vec3(0.03) * materialAlbedo;
    vec3 color = ambient + Lo;
    
    // Tone mapping
    color = color / (color + vec3(1.0));
    // Gamma correction
    color = pow(color, vec3(1.0/2.2));
    
    FragColor = vec4(color, 1.0);
}
` + "\x00"

	textureVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aUV;
layout (location = 2) in vec3 aColor;

out vec2 TexCoord;
out vec3 VertColor;

uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;

void main() {
    gl_Position = proj * view * model * vec4(aPos, 1.0);
    TexCoord = aUV;
    VertColor = aColor;
}
` + "\x00"

	textureFragmentShaderSource = `
#version 410 core
in vec2 TexCoord;
in vec3 VertColor;

out vec4 FragColor;

uniform sampler2D textureSampler;
uniform bool useTexture;

void main() {
    if (useTexture) {
        vec4 texColor = texture(textureSampler, TexCoord);
        FragColor = texColor * vec4(VertColor, 1.0);
    } else {
        FragColor = vec4(VertColor, 1.0);
    }
}
` + "\x00"

	shadowVertexShaderSource = `
#version 410 core
layout (location = 0) in vec3 aPos;

uniform mat4 lightSpaceMatrix;
uniform mat4 model;

void main() {
    gl_Position = lightSpaceMatrix * model * vec4(aPos, 1.0);
}
` + "\x00"

	shadowFragmentShaderSource = `
#version 410 core

void main() {
    // Depth is written automatically
}
` + "\x00"
)

func NewOpenGLRenderer(width, height int) *OpenGLRenderer {
	return &OpenGLRenderer{
		width:  width,
		height: height,
		renderContext: &RenderContext{
			ViewFrustum: &ViewFrustum{},
		},
		UseColor:         true,
		ShowDebugInfo:    false,
		maxVertices:      100000,
		currentVertices:  make([]VulkanVertex, 0, 600000),
		pbrVertices:      make([]PBRVertex, 0, 900000),
		texturedVertices: make([]TexturedVertex, 0, 800000),
		lineVertices:     make([]float32, 0, 60000),
		vboCache:         NewMeshBufferCache(),
		textureCache:     make(map[*Texture]uint32),
		shadowResolution: 2048,
		enableShadows:    true,
		usePBRPath:       false,
	}
}

func (r *OpenGLRenderer) Initialize() error {
	if r.initialized {
		return nil
	}

	fmt.Println("[OpenGL] Initializing...")

	// Lock to OS thread (required for OpenGL)
	runtime.LockOSThread()

	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize GLFW: %v", err)
	}

	// Set OpenGL version
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	// Create window
	window, err := glfw.CreateWindow(r.width, r.height, "Go 3D Engine (OpenGL)", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}
	r.window = window

	r.window.MakeContextCurrent()

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL: %v", err)
	}

	gl.Disable(gl.CULL_FACE) // Disable face culling for now

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Printf("[OpenGL] Version: %s\n", version)

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	// gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// Create shader programs
	if err := r.createShaderProgram(); err != nil {
		return err
	}

	if err := r.createLineShaderProgram(); err != nil {
		return err
	}

	if err := r.createPBRShaderProgram(); err != nil {
		return err
	}

	if err := r.createTextureShaderProgram(); err != nil {
		return err
	}

	if err := r.createShadowShaderProgram(); err != nil {
		return err
	}

	// Create shadow map FBO
	if err := r.createShadowMapFBO(); err != nil {
		return err
	}

	// Create vertex buffers
	if err := r.createBuffers(); err != nil {
		return err
	}

	// Set viewport
	gl.Viewport(0, 0, int32(r.width), int32(r.height))

	fmt.Println("[OpenGL] Initialization complete")
	r.initialized = true
	return nil
}

func (r *OpenGLRenderer) createShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link program: %v", log)
	}

	r.program = program

	// Get uniform locations
	r.uniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.uniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.uniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))

	return nil
}

func (r *OpenGLRenderer) createLineShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(lineVertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("line vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(lineFragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("line fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link line program: %v", log)
	}

	r.lineProgram = program

	// Get uniform locations
	r.lineUniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.lineUniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.lineUniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))

	return nil
}

func (r *OpenGLRenderer) createPBRShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(pbrVertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("PBR vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(pbrFragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("PBR fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link PBR program: %v", log)
	}

	r.pbrProgram = program

	// Get uniform locations
	r.pbrUniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.pbrUniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.pbrUniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))
	r.pbrUniformMetallic = gl.GetUniformLocation(program, gl.Str("metallic\x00"))
	r.pbrUniformRoughness = gl.GetUniformLocation(program, gl.Str("roughness\x00"))
	r.pbrUniformAlbedo = gl.GetUniformLocation(program, gl.Str("albedo\x00"))
	r.pbrUniformCameraPos = gl.GetUniformLocation(program, gl.Str("cameraPos\x00"))
	r.pbrUniformLightPos = gl.GetUniformLocation(program, gl.Str("lightPos\x00"))
	r.pbrUniformLightColor = gl.GetUniformLocation(program, gl.Str("lightColor\x00"))
	r.pbrUniformLightSpaceMatrix = gl.GetUniformLocation(program, gl.Str("lightSpaceMatrix\x00"))
	r.pbrUniformShadowMap = gl.GetUniformLocation(program, gl.Str("shadowMap\x00"))
	r.pbrUniformUseShadows = gl.GetUniformLocation(program, gl.Str("useShadows\x00"))

	fmt.Println("[OpenGL] PBR shader program created successfully")
	return nil
}

func (r *OpenGLRenderer) createTextureShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(textureVertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("texture vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(textureFragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("texture fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link texture program: %v", log)
	}

	r.textureProgram = program

	// Get uniform locations
	r.textureUniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.textureUniformView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	r.textureUniformProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))
	r.textureUniformSampler = gl.GetUniformLocation(program, gl.Str("textureSampler\x00"))
	r.textureUniformUseTexture = gl.GetUniformLocation(program, gl.Str("useTexture\x00"))

	fmt.Println("[OpenGL] Texture shader program created successfully")
	return nil
}

func (r *OpenGLRenderer) createShadowShaderProgram() error {
	// Compile vertex shader
	vertexShader, err := r.compileShader(shadowVertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return fmt.Errorf("shadow vertex shader: %v", err)
	}
	defer gl.DeleteShader(vertexShader)

	// Compile fragment shader
	fragmentShader, err := r.compileShader(shadowFragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return fmt.Errorf("shadow fragment shader: %v", err)
	}
	defer gl.DeleteShader(fragmentShader)

	// Link program
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// Check for linking errors
	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return fmt.Errorf("failed to link shadow program: %v", log)
	}

	r.shadowProgram = program

	// Get uniform locations
	r.shadowUniformModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	r.shadowUniformLightSpaceMatrix = gl.GetUniformLocation(program, gl.Str("lightSpaceMatrix\x00"))

	fmt.Println("[OpenGL] Shadow shader program created successfully")
	return nil
}

func (r *OpenGLRenderer) createShadowMapFBO() error {
	// Generate framebuffer
	gl.GenFramebuffers(1, &r.shadowFBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.shadowFBO)

	// Create depth texture
	gl.GenTextures(1, &r.shadowDepthTexture)
	gl.BindTexture(gl.TEXTURE_2D, r.shadowDepthTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, int32(r.shadowResolution), int32(r.shadowResolution),
		0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)

	// Set texture parameters
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	
	// Set border color to white (1.0) so areas outside shadow map are not in shadow
	borderColor := []float32{1.0, 1.0, 1.0, 1.0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

	// Attach depth texture to framebuffer
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, r.shadowDepthTexture, 0)

	// We don't need color attachment for shadow map
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)

	// Check framebuffer completeness
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("shadow framebuffer is not complete")
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	fmt.Printf("[OpenGL] Shadow map FBO created (%dx%d)\n", r.shadowResolution, r.shadowResolution)
	return nil
}

func (r *OpenGLRenderer) compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	// Check for compilation errors
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}

	return shader, nil
}

func (r *OpenGLRenderer) createBuffers() error {
	// Generate VAO for triangles
	gl.GenVertexArrays(1, &r.vao)
	gl.BindVertexArray(r.vao)

	// Generate VBO
	gl.GenBuffers(1, &r.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)

	// Allocate buffer (dynamic)
	bufferSize := r.maxVertices * 6 * 4 // 6 floats per vertex, 4 bytes per float
	gl.BufferData(gl.ARRAY_BUFFER, bufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Color attribute (location 1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	// Generate VAO for lines
	gl.GenVertexArrays(1, &r.lineVAO)
	gl.BindVertexArray(r.lineVAO)

	// Generate line VBO
	gl.GenBuffers(1, &r.lineVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.lineVBO)

	// Allocate buffer for lines
	lineBufferSize := 10000 * 6 * 4 // 10k vertices * 6 floats * 4 bytes
	gl.BufferData(gl.ARRAY_BUFFER, lineBufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Color attribute (location 1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	// Generate VAO for PBR rendering (with normals)
	var pbrVAO, pbrVBO uint32
	gl.GenVertexArrays(1, &pbrVAO)
	gl.BindVertexArray(pbrVAO)

	gl.GenBuffers(1, &pbrVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, pbrVBO)

	// Allocate buffer for PBR vertices (pos(3) + normal(3) + color(3) = 9 floats)
	pbrBufferSize := r.maxVertices * 9 * 4
	gl.BufferData(gl.ARRAY_BUFFER, pbrBufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 9*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Normal attribute (location 1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 9*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	// Color attribute (location 2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 9*4, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(2)

	gl.BindVertexArray(0)

	// Store PBR VAO/VBO
	r.pbrVAO = pbrVAO
	r.pbrVBO = pbrVBO
	
	// Create VAO and VBO for textured rendering
	var textureVAO, textureVBO uint32
	gl.GenVertexArrays(1, &textureVAO)
	gl.BindVertexArray(textureVAO)

	gl.GenBuffers(1, &textureVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, textureVBO)

	// Allocate buffer for textured vertices (pos(3) + uv(2) + color(3) = 8 floats)
	textureBufferSize := r.maxVertices * 8 * 4
	gl.BufferData(gl.ARRAY_BUFFER, textureBufferSize, nil, gl.DYNAMIC_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// UV attribute (location 1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	// Color attribute (location 2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	gl.BindVertexArray(0)

	// Store texture VAO/VBO
	r.textureVAO = textureVAO
	r.textureVBO = textureVBO

	return nil
}

func (r *OpenGLRenderer) Shutdown() {
	if !r.initialized {
		return
	}

	fmt.Println("[OpenGL] Shutting down...")

	// Delete OpenGL resources
	gl.DeleteBuffers(1, &r.vbo)
	gl.DeleteBuffers(1, &r.lineVBO)
	gl.DeleteBuffers(1, &r.pbrVBO)
	gl.DeleteBuffers(1, &r.textureVBO)
	gl.DeleteVertexArrays(1, &r.vao)
	gl.DeleteVertexArrays(1, &r.lineVAO)
	gl.DeleteVertexArrays(1, &r.pbrVAO)
	gl.DeleteVertexArrays(1, &r.textureVAO)
	gl.DeleteProgram(r.program)
	gl.DeleteProgram(r.lineProgram)
	gl.DeleteProgram(r.pbrProgram)
	gl.DeleteProgram(r.textureProgram)
	gl.DeleteProgram(r.shadowProgram)
	
	// Delete cached textures
	for _, texID := range r.textureCache {
		gl.DeleteTextures(1, &texID)
	}
	r.textureCache = make(map[*Texture]uint32)
	
	// Delete shadow resources
	if r.shadowDepthTexture != 0 {
		gl.DeleteTextures(1, &r.shadowDepthTexture)
	}
	if r.shadowFBO != 0 {
		gl.DeleteFramebuffers(1, &r.shadowFBO)
	}

	r.window.Destroy()
	glfw.Terminate()
	r.initialized = false
}

func (r *OpenGLRenderer) BeginFrame() {
	if !r.initialized {
		return
	}

	glfw.PollEvents()

	// Clear buffers
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (r *OpenGLRenderer) EndFrame() {
	// No-op for OpenGL (rendering happens in Present)
}

func (r *OpenGLRenderer) Present() {
	if !r.initialized {
		return
	}

	// FIXED: Limit buffer size
	const MAX_VERTICES = 1000000

	// Render regular vertices
	if len(r.currentVertices) > MAX_VERTICES {
		r.currentVertices = r.currentVertices[:MAX_VERTICES]
	}

	if len(r.currentVertices) > 0 {
		// Upload vertex data
		gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
		dataSize := len(r.currentVertices) * 24
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, dataSize, gl.Ptr(r.currentVertices))

		// Use shader program
		gl.UseProgram(r.program)

		// Update matrices
		r.updateMatrices(r.uniformModel, r.uniformView, r.uniformProj)

		// Draw
		gl.BindVertexArray(r.vao)
		vertexCount := int32(len(r.currentVertices))
		gl.DrawArrays(gl.TRIANGLES, 0, vertexCount)
		gl.BindVertexArray(0)
	}

	// Render PBR vertices
	if len(r.pbrVertices) > 0 {
		if len(r.pbrVertices) > MAX_VERTICES {
			r.pbrVertices = r.pbrVertices[:MAX_VERTICES]
		}

		// Upload PBR vertex data
		gl.BindBuffer(gl.ARRAY_BUFFER, r.pbrVBO)
		pbrDataSize := len(r.pbrVertices) * 36 // 9 floats * 4 bytes
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, pbrDataSize, gl.Ptr(r.pbrVertices))

		// Use PBR shader program
		gl.UseProgram(r.pbrProgram)

		// Update matrices
		r.updateMatrices(r.pbrUniformModel, r.pbrUniformView, r.pbrUniformProj)

		// Set PBR uniforms (use first light if available)
		if r.LightingSystem != nil && len(r.LightingSystem.Lights) > 0 {
			light := r.LightingSystem.Lights[0]
			gl.Uniform3f(r.pbrUniformLightPos, float32(light.Position.X), float32(light.Position.Y), float32(light.Position.Z))
			gl.Uniform3f(r.pbrUniformLightColor, float32(light.Color.R)/255.0, float32(light.Color.G)/255.0, float32(light.Color.B)/255.0)
		} else {
			// Default light
			gl.Uniform3f(r.pbrUniformLightPos, 0, 100, 0)
			gl.Uniform3f(r.pbrUniformLightColor, 1.0, 1.0, 1.0)
		}

		// Set camera position
		if r.Camera != nil {
			camPos := r.Camera.GetPosition()
			gl.Uniform3f(r.pbrUniformCameraPos, float32(camPos.X), float32(camPos.Y), float32(camPos.Z))
		}

		// Default PBR parameters (will be overridden per-material later)
		gl.Uniform1f(r.pbrUniformMetallic, 0.5)
		gl.Uniform1f(r.pbrUniformRoughness, 0.5)
		gl.Uniform3f(r.pbrUniformAlbedo, 0, 0, 0) // 0 means use vertex color

		// Set shadow uniforms
		if r.enableShadows {
			// Upload light space matrix
			var m [16]float32
			for i := 0; i < 16; i++ {
				m[i] = float32(r.shadowLightMatrix.M[i])
			}
			gl.UniformMatrix4fv(r.pbrUniformLightSpaceMatrix, 1, false, &m[0])

			// Bind shadow map texture
			gl.ActiveTexture(gl.TEXTURE1)
			gl.BindTexture(gl.TEXTURE_2D, r.shadowDepthTexture)
			gl.Uniform1i(r.pbrUniformShadowMap, 1)
			gl.Uniform1i(r.pbrUniformUseShadows, 1)
		} else {
			gl.Uniform1i(r.pbrUniformUseShadows, 0)
		}

		// Draw PBR geometry
		gl.BindVertexArray(r.pbrVAO)
		pbrVertexCount := int32(len(r.pbrVertices))
		gl.DrawArrays(gl.TRIANGLES, 0, pbrVertexCount)
		gl.BindVertexArray(0)
		
		// Unbind shadow map
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, 0)
	}

	// Render textured vertices
	if len(r.texturedVertices) > 0 {
		if len(r.texturedVertices) > MAX_VERTICES {
			r.texturedVertices = r.texturedVertices[:MAX_VERTICES]
		}

		// Upload textured vertex data
		gl.BindBuffer(gl.ARRAY_BUFFER, r.textureVBO)
		textureDataSize := len(r.texturedVertices) * 32 // 8 floats * 4 bytes
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, textureDataSize, gl.Ptr(r.texturedVertices))

		// Use texture shader program
		gl.UseProgram(r.textureProgram)

		// Update matrices
		r.updateMatrices(r.textureUniformModel, r.textureUniformView, r.textureUniformProj)

		// Bind and activate texture if available
		if r.activeTexture != nil {
			texID := r.uploadTexture(r.activeTexture)
			gl.ActiveTexture(gl.TEXTURE0)
			gl.BindTexture(gl.TEXTURE_2D, texID)
			gl.Uniform1i(r.textureUniformSampler, 0)
			gl.Uniform1i(r.textureUniformUseTexture, 1) // Enable texture
		} else {
			gl.Uniform1i(r.textureUniformUseTexture, 0) // Disable texture
		}

		// Draw textured geometry
		gl.BindVertexArray(r.textureVAO)
		textureVertexCount := int32(len(r.texturedVertices))
		gl.DrawArrays(gl.TRIANGLES, 0, textureVertexCount)
		gl.BindVertexArray(0)
		
		// Unbind texture
		gl.BindTexture(gl.TEXTURE_2D, 0)
	}

	// Render lines (if any)
	if len(r.lineVertices) > 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.lineVBO)
		lineDataSize := len(r.lineVertices) * 4
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, lineDataSize, gl.Ptr(r.lineVertices))

		gl.UseProgram(r.lineProgram)
		r.updateMatrices(r.lineUniformModel, r.lineUniformView, r.lineUniformProj)

		gl.BindVertexArray(r.lineVAO)
		lineVertexCount := int32(len(r.lineVertices) / 6)
		gl.DrawArrays(gl.LINES, 0, lineVertexCount)
		gl.BindVertexArray(0)
	}

	r.window.SwapBuffers()
	r.frameCount++

	// Clear vertices after rendering
	r.currentVertices = r.currentVertices[:0]
	r.pbrVertices = r.pbrVertices[:0]
	r.lineVertices = r.lineVertices[:0]

	if r.frameCount%60 == 0 && r.ShowDebugInfo {
		r.window.SetTitle(fmt.Sprintf("Go 3D Engine (OpenGL) - Frame %d", r.frameCount))
	}
}

func (r *OpenGLRenderer) updateMatrices(modelUniform, viewUniform, projUniform int32) {
	// Model matrix (identity - transforms are baked into vertices)
	modelMatrix := IdentityMatrix()
	r.uploadMatrix(modelUniform, modelMatrix)

	// View matrix (inverse of camera transform)
	viewMatrix := r.Camera.Transform.GetInverseMatrix()
	r.uploadMatrix(viewUniform, viewMatrix)

	// Projection matrix
	projMatrix := r.buildProjectionMatrix(r.Camera)
	r.uploadMatrix(projUniform, projMatrix)
}

func (r *OpenGLRenderer) buildProjectionMatrix(camera *Camera) Matrix4x4 {
	// Use a proper FOV for OpenGL (the custom renderer uses FOV as a scaling factor, not an angle)
	// We'll use 60 degrees vertical FOV for a good view
	fovYDegrees := 60.0
	fovY := fovYDegrees * math.Pi / 180.0
	aspect := float64(r.width) / float64(r.height)
	near := camera.Near
	far := camera.Far

	f := 1.0 / math.Tan(fovY/2.0)

	// Standard OpenGL perspective projection matrix
	return Matrix4x4{M: [16]float64{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}}
}

func (r *OpenGLRenderer) uploadMatrix(uniform int32, matrix Matrix4x4) {
	// Convert to float32 array
	var m [16]float32
	for i := 0; i < 16; i++ {
		m[i] = float32(matrix.M[i])
	}
	gl.UniformMatrix4fv(uniform, 1, true, &m[0])
}

func (r *OpenGLRenderer) RenderScene(scene *Scene) {
	if !r.initialized {
		return
	}

	if r.LightingSystem != nil {
		r.LightingSystem.SetCamera(scene.Camera)
	}

	// First pass: Render shadow map
	if r.enableShadows {
		r.renderShadowPass(scene)
	}

	// Clear vertex buffer at start of scene rendering
	r.currentVertices = r.currentVertices[:0]
	r.pbrVertices = r.pbrVertices[:0]
	r.texturedVertices = r.texturedVertices[:0]

	// Collect all geometry
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNode(node, worldMatrix, scene.Camera)
	}

	// Update matrices now that we have a camera
	if r.Camera != nil {
		gl.UseProgram(r.program)
		r.updateMatrices(r.uniformModel, r.uniformView, r.uniformProj)
	}
}

func (r *OpenGLRenderer) renderNode(node *SceneNode, worldMatrix Matrix4x4, camera *Camera) {
	switch obj := node.Object.(type) {
	case *Triangle:
		r.RenderTriangle(obj, worldMatrix, camera)
	case *Quad:
		r.renderQuad(obj, worldMatrix, camera)
	case *Mesh:
		// Check if mesh has textured material
		if obj.Material != nil {
			if texMat, ok := obj.Material.(*TexturedMaterial); ok && texMat.UseTextures && texMat.DiffuseTexture != nil {
				// Render with texture
				r.RenderTexturedMesh(obj, worldMatrix, camera, texMat.DiffuseTexture)
				return
			}
		}
		// Fall back to regular mesh rendering
		r.RenderMesh(obj, worldMatrix, camera)
	case *InstancedMesh:
		r.RenderInstancedMesh(obj, worldMatrix, camera)
	case *Line:
		r.RenderLine(obj, worldMatrix, camera)
	case *Point:
		r.RenderPoint(obj, worldMatrix, camera)
	case *LODGroup:
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil && len(currentMesh.Vertices) > 0 && len(currentMesh.Indices) > 0 {
			r.RenderMesh(currentMesh, worldMatrix, camera)
		}
	case *LODGroupWithTransitions:
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil && len(currentMesh.Vertices) > 0 && len(currentMesh.Indices) > 0 {
			r.RenderMesh(currentMesh, worldMatrix, camera)
		}
	}
}

func (r *OpenGLRenderer) RenderTriangle(tri *Triangle, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices to world space
	p0 := worldMatrix.TransformPoint(tri.P0)
	p1 := worldMatrix.TransformPoint(tri.P1)
	p2 := worldMatrix.TransformPoint(tri.P2)

	// Get color
	color := tri.Material.GetDiffuseColor(0, 0)

	// Check if wireframe mode
	if tri.Material.IsWireframe() {
		// Render as lines (edges only)
		rf := float32(color.R) / 255.0
		gf := float32(color.G) / 255.0
		bf := float32(color.B) / 255.0

		// Add three edges
		r.addLineVertex(p0, rf, gf, bf)
		r.addLineVertex(p1, rf, gf, bf)

		r.addLineVertex(p1, rf, gf, bf)
		r.addLineVertex(p2, rf, gf, bf)

		r.addLineVertex(p2, rf, gf, bf)
		r.addLineVertex(p0, rf, gf, bf)
		return
	}

	// Apply simple lighting if available
	if r.LightingSystem != nil {
		// Calculate normal
		normal := CalculateSurfaceNormal(&tri.P0, &tri.P1, &tri.P2, tri.Normal, tri.UseSetNormal)
		worldNormal := worldMatrix.TransformDirection(normal)

		// Simple diffuse lighting
		intensity := 0.3 // Ambient
		for _, light := range r.LightingSystem.Lights {
			if !light.IsEnabled {
				continue
			}

			centerX := (p0.X + p1.X + p2.X) / 3.0
			centerY := (p0.Y + p1.Y + p2.Y) / 3.0
			centerZ := (p0.Z + p1.Z + p2.Z) / 3.0

			lx := light.Position.X - centerX
			ly := light.Position.Y - centerY
			lz := light.Position.Z - centerZ
			lx, ly, lz = normalizeVector(lx, ly, lz)

			diff := dotProduct(worldNormal.X, worldNormal.Y, worldNormal.Z, lx, ly, lz)
			if diff > 0 {
				intensity += diff * light.Intensity * 0.7
			}
		}

		if intensity > 1.0 {
			intensity = 1.0
		}

		color = Color{
			R: uint8(float64(color.R) * intensity),
			G: uint8(float64(color.G) * intensity),
			B: uint8(float64(color.B) * intensity),
		}
	}

	rf := float32(color.R) / 255.0
	gf := float32(color.G) / 255.0
	bf := float32(color.B) / 255.0

	// Add vertices (interleaved: position + color)
	r.addVertex(p0, rf, gf, bf)
	r.addVertex(p1, rf, gf, bf)
	r.addVertex(p2, rf, gf, bf)
}

func (r *OpenGLRenderer) RenderMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera) {
	// If the mesh has no vertices or faces, there's nothing to render.
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return
	}

	// Determine if we should use the PBR rendering path.
	isPBR := false
	if _, ok := mesh.Material.(*PBRMaterial); ok {
		isPBR = true
	}

	// Check if the mesh has pre-calculated normals. This should be true for all meshes now.
	hasNormals := len(mesh.Normals) == len(mesh.Vertices)
	if !hasNormals {
		// Log a warning or handle error, as this indicates a problem in mesh creation/loading.
		// For now, we'll skip rendering triangles that are missing normals.
		return
	}

	// Iterate over each triangle in the mesh.
	for i := 0; i < len(mesh.Indices)-2; i += 3 {
		// Get the indices for the triangle's vertices.
		idx0, idx1, idx2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]

		// Bounds check for vertex indices.
		if idx0 >= len(mesh.Vertices) || idx1 >= len(mesh.Vertices) || idx2 >= len(mesh.Vertices) {
			continue
		}

		// Get local-space vertex positions and apply mesh's local position offset.
		v0, v1, v2 := mesh.Vertices[idx0], mesh.Vertices[idx1], mesh.Vertices[idx2]
		p0_local := Point{X: v0.X + mesh.Position.X, Y: v0.Y + mesh.Position.Y, Z: v0.Z + mesh.Position.Z}
		p1_local := Point{X: v1.X + mesh.Position.X, Y: v1.Y + mesh.Position.Y, Z: v1.Z + mesh.Position.Z}
		p2_local := Point{X: v2.X + mesh.Position.X, Y: v2.Y + mesh.Position.Y, Z: v2.Z + mesh.Position.Z}

		// Transform vertex positions to world space.
		finalP0 := worldMatrix.TransformPoint(p0_local)
		finalP1 := worldMatrix.TransformPoint(p1_local)
		finalP2 := worldMatrix.TransformPoint(p2_local)

		// Get the material color for the triangle.
		color := mesh.Material.GetDiffuseColor(0, 0)
		rf, gf, bf := float32(color.R)/255.0, float32(color.G)/255.0, float32(color.B)/255.0

		// Handle wireframe rendering.
		if mesh.Material.IsWireframe() {
			wireColor := mesh.Material.GetWireframeColor()
			wrf, wgf, wbf := float32(wireColor.R)/255.0, float32(wireColor.G)/255.0, float32(wireColor.B)/255.0
			r.addLineVertex(finalP0, wrf, wgf, wbf)
			r.addLineVertex(finalP1, wrf, wgf, wbf)
			r.addLineVertex(finalP1, wrf, wgf, wbf)
			r.addLineVertex(finalP2, wrf, wgf, wbf)
			r.addLineVertex(finalP2, wrf, wgf, wbf)
			r.addLineVertex(finalP0, wrf, wgf, wbf)
			continue
		}

		if isPBR {
			// PBR Path: Use smooth, per-vertex normals.
			n0, n1, n2 := mesh.Normals[idx0], mesh.Normals[idx1], mesh.Normals[idx2]
			worldNormal0 := worldMatrix.TransformDirection(n0)
			worldNormal1 := worldMatrix.TransformDirection(n1)
			worldNormal2 := worldMatrix.TransformDirection(n2)

			r.addPBRVertex(finalP0, worldNormal0, rf, gf, bf)
			r.addPBRVertex(finalP1, worldNormal1, rf, gf, bf)
			r.addPBRVertex(finalP2, worldNormal2, rf, gf, bf)
		} else {
			// Simple Lighting Path: Use an averaged normal from pre-calculated data
			// to simulate the original flat shading without recalculating from vertices.
			n0, n1, n2 := mesh.Normals[idx0], mesh.Normals[idx1], mesh.Normals[idx2]

			// Average vertex normals to get a face normal
			avgNormal := Point{
				X: (n0.X + n1.X + n2.X) / 3.0,
				Y: (n0.Y + n1.Y + n2.Y) / 3.0,
				Z: (n0.Z + n1.Z + n2.Z) / 3.0,
			}

			// Transform the averaged normal to world space.
			worldNormal := worldMatrix.TransformDirection(avgNormal)

			// Apply simple CPU-based lighting.
			if r.LightingSystem != nil {
				intensity := 0.2 // Ambient base
				for _, light := range r.LightingSystem.Lights {
					if !light.IsEnabled {
						continue
					}

					// Center of the triangle in world space for light calculation.
					centerX := (finalP0.X + finalP1.X + finalP2.X) / 3.0
					centerY := (finalP0.Y + finalP1.Y + finalP2.Y) / 3.0
					centerZ := (finalP0.Z + finalP1.Z + finalP2.Z) / 3.0

					lx, ly, lz := light.Position.X-centerX, light.Position.Y-centerY, light.Position.Z-centerZ
					lx, ly, lz = normalizeVector(lx, ly, lz)

					diff := dotProduct(worldNormal.X, worldNormal.Y, worldNormal.Z, lx, ly, lz)
					if diff > 0 {
						intensity += diff * light.Intensity * 0.8
					}
				}

				if intensity > 1.0 {
					intensity = 1.0
				}

				// Modulate color by light intensity.
				color.R, color.G, color.B = uint8(float64(color.R)*intensity), uint8(float64(color.G)*intensity), uint8(float64(color.B)*intensity)
				rf, gf, bf = float32(color.R)/255.0, float32(color.G)/255.0, float32(color.B)/255.0
			}

			// Add vertices with the calculated flat color to the buffer.
			r.addVertex(finalP0, rf, gf, bf)
			r.addVertex(finalP1, rf, gf, bf)
			r.addVertex(finalP2, rf, gf, bf)
		}
	}
}

// RenderInstancedMesh renders multiple instances of the same mesh efficiently
func (r *OpenGLRenderer) RenderInstancedMesh(instMesh *InstancedMesh, worldMatrix Matrix4x4, camera *Camera) {
	if !instMesh.Enabled || instMesh.BaseMesh == nil || len(instMesh.Instances) == 0 {
		return
	}

	// Render each instance (for now, simple approach)
	// TODO: Use glDrawElementsInstanced for true hardware instancing
	for _, instance := range instMesh.Instances {
		// Combine world matrix with instance transform
		finalMatrix := worldMatrix.Multiply(instance.Transform)
		
		// Temporarily override material color if instance has custom color
		originalMat := instMesh.BaseMesh.Material
		if instance.Color.R != 0 || instance.Color.G != 0 || instance.Color.B != 0 {
			tempMat := NewMaterial()
			tempMat.DiffuseColor = instance.Color
			// Copy other properties from original if it exists
			if originalMat != nil {
				tempMat.SpecularColor = originalMat.GetSpecularColor()
				tempMat.Shininess = originalMat.GetShininess()
				tempMat.SpecularStrength = originalMat.GetSpecularStrength()
				tempMat.AmbientStrength = originalMat.GetAmbientStrength()
			}
			instMesh.BaseMesh.Material = &tempMat
		}
		
		// Render the mesh with instance transform
		r.RenderMesh(instMesh.BaseMesh, finalMatrix, camera)
		
		// Restore original material
		instMesh.BaseMesh.Material = originalMat
	}
}

func (r *OpenGLRenderer) renderQuad(quad *Quad, worldMatrix Matrix4x4, camera *Camera) {
	triangles := ConvertQuadToTriangles(quad)
	for _, tri := range triangles {
		r.RenderTriangle(tri, worldMatrix, camera)
	}
}

func (r *OpenGLRenderer) RenderLine(line *Line, worldMatrix Matrix4x4, camera *Camera) {
	// Transform vertices to world space
	p0 := worldMatrix.TransformPoint(line.Start)
	p1 := worldMatrix.TransformPoint(line.End)

	// Use white color for lines
	rf := float32(1.0)
	gf := float32(1.0)
	bf := float32(1.0)

	// Add line vertices
	r.addLineVertex(p0, rf, gf, bf)
	r.addLineVertex(p1, rf, gf, bf)
}

func (r *OpenGLRenderer) RenderPoint(point *Point, worldMatrix Matrix4x4, camera *Camera) {
	// Transform point to world space
	p := worldMatrix.TransformPoint(*point)

	// Render as small sphere approximation (octahedron)
	size := 0.5
	color := Color{255, 255, 255}
	rf := float32(color.R) / 255.0
	gf := float32(color.G) / 255.0
	bf := float32(color.B) / 255.0

	// Create 8 triangles forming an octahedron
	top := Point{X: p.X, Y: p.Y + size, Z: p.Z}
	bottom := Point{X: p.X, Y: p.Y - size, Z: p.Z}
	front := Point{X: p.X, Y: p.Y, Z: p.Z + size}
	back := Point{X: p.X, Y: p.Y, Z: p.Z - size}
	left := Point{X: p.X - size, Y: p.Y, Z: p.Z}
	right := Point{X: p.X + size, Y: p.Y, Z: p.Z}

	// Top pyramid
	r.addVertex(top, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)

	r.addVertex(top, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)

	// Bottom pyramid
	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)
	r.addVertex(right, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
	r.addVertex(back, rf, gf, bf)

	r.addVertex(bottom, rf, gf, bf)
	r.addVertex(front, rf, gf, bf)
	r.addVertex(left, rf, gf, bf)
}

func (r *OpenGLRenderer) addVertex(p Point, red, green, blue float32) {
	r.currentVertices = append(r.currentVertices,
		VulkanVertex{
			Pos:   [3]float32{float32(p.X), float32(p.Y), float32(p.Z)},
			Color: [3]float32{red, green, blue},
		},
	)
}

func (r *OpenGLRenderer) addPBRVertex(p Point, normal Point, red, green, blue float32) {
	r.pbrVertices = append(r.pbrVertices,
		PBRVertex{
			Pos:    [3]float32{float32(p.X), float32(p.Y), float32(p.Z)},
			Normal: [3]float32{float32(normal.X), float32(normal.Y), float32(normal.Z)},
			Color:  [3]float32{red, green, blue},
		},
	)
}

func (r *OpenGLRenderer) addLineVertex(p Point, red, green, blue float32) {
	r.lineVertices = append(r.lineVertices,
		float32(p.X), float32(p.Y), float32(p.Z), // Position
		red, green, blue, // Color
	)
}

func (r *OpenGLRenderer) SetLightingSystem(ls *LightingSystem) {
	r.LightingSystem = ls
	r.renderContext.LightingSystem = ls
}

func (r *OpenGLRenderer) SetCamera(camera *Camera) {
	r.Camera = camera
	r.renderContext.Camera = camera
}

func (r *OpenGLRenderer) GetDimensions() (int, int) {
	return r.width, r.height
}

func (r *OpenGLRenderer) SetUseColor(useColor bool) {
	r.UseColor = useColor
}

func (r *OpenGLRenderer) SetShowDebugInfo(show bool) {
	r.ShowDebugInfo = show
}

func (r *OpenGLRenderer) SetClipBounds(minX, minY, maxX, maxY int) {
	// Store for interface compliance, but not used in OpenGL
	r.clipMinX = minX
	r.clipMinY = minY
	r.clipMaxX = maxX
	r.clipMaxY = maxY
}

func (r *OpenGLRenderer) GetRenderContext() *RenderContext {
	return r.renderContext
}

// ShouldClose checks if window should close
func (r *OpenGLRenderer) ShouldClose() bool {
	if r.window == nil {
		return true
	}
	return r.window.ShouldClose()
}

// GetWindow returns the GLFW window (for input handling)
func (r *OpenGLRenderer) GetWindow() *glfw.Window {
	return r.window
}

// uploadTexture uploads a texture to OpenGL and returns the texture ID
func (r *OpenGLRenderer) uploadTexture(texture *Texture) uint32 {
	// Check if already cached
	if texID, exists := r.textureCache[texture]; exists {
		return texID
	}

	var texID uint32
	gl.GenTextures(1, &texID)
	gl.BindTexture(gl.TEXTURE_2D, texID)

	// Convert texture data to RGBA format for OpenGL
	rgbaData := make([]uint8, texture.Width*texture.Height*4)
	for i, col := range texture.Data {
		rgbaData[i*4+0] = col.R
		rgbaData[i*4+1] = col.G
		rgbaData[i*4+2] = col.B
		rgbaData[i*4+3] = 255 // Full alpha
	}

	// Upload texture data
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(texture.Width), int32(texture.Height),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgbaData))

	// Set texture parameters
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	gl.BindTexture(gl.TEXTURE_2D, 0)

	// Cache the texture
	r.textureCache[texture] = texID

	return texID
}

// addTexturedVertex adds a vertex with UV coordinates to the textured vertex buffer
func (r *OpenGLRenderer) addTexturedVertex(pos Point, u, v float32, red, green, blue float32) {
	vertex := TexturedVertex{
		Pos:   [3]float32{float32(pos.X), float32(pos.Y), float32(pos.Z)},
		UV:    [2]float32{u, v},
		Color: [3]float32{red, green, blue},
	}
	r.texturedVertices = append(r.texturedVertices, vertex)
}

// RenderTexturedMesh renders a mesh with texture support
func (r *OpenGLRenderer) RenderTexturedMesh(mesh *Mesh, worldMatrix Matrix4x4, camera *Camera, texture *Texture) {
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return
	}

	// Check if mesh has UV coordinates
	hasUVs := len(mesh.UVs) == len(mesh.Vertices)
	if !hasUVs {
		// Fall back to regular rendering
		r.RenderMesh(mesh, worldMatrix, camera)
		return
	}

	// Set active texture for this batch
	r.activeTexture = texture

	// Get material color
	color := ColorWhite
	if mesh.Material != nil {
		if mat, ok := mesh.Material.(*Material); ok {
			color = mat.DiffuseColor
		} else if texMat, ok := mesh.Material.(*TexturedMaterial); ok {
			color = texMat.DiffuseColor
		}
	}

	rf := float32(color.R) / 255.0
	gf := float32(color.G) / 255.0
	bf := float32(color.B) / 255.0

	// Render each triangle
	for i := 0; i < len(mesh.Indices); i += 3 {
		if i+2 >= len(mesh.Indices) {
			break
		}

		idx0 := mesh.Indices[i]
		idx1 := mesh.Indices[i+1]
		idx2 := mesh.Indices[i+2]

		if idx0 >= len(mesh.Vertices) || idx1 >= len(mesh.Vertices) || idx2 >= len(mesh.Vertices) {
			continue
		}

		// Transform vertices
		p0 := worldMatrix.TransformPoint(mesh.Vertices[idx0])
		p1 := worldMatrix.TransformPoint(mesh.Vertices[idx1])
		p2 := worldMatrix.TransformPoint(mesh.Vertices[idx2])

		// Get UV coordinates
		uv0 := mesh.UVs[idx0]
		uv1 := mesh.UVs[idx1]
		uv2 := mesh.UVs[idx2]

		// Add textured vertices
		r.addTexturedVertex(p0, float32(uv0.U), float32(uv0.V), rf, gf, bf)
		r.addTexturedVertex(p1, float32(uv1.U), float32(uv1.V), rf, gf, bf)
		r.addTexturedVertex(p2, float32(uv2.U), float32(uv2.V), rf, gf, bf)
	}
}

// calculateLightSpaceMatrix calculates the light view-projection matrix for shadow mapping
func (r *OpenGLRenderer) calculateLightSpaceMatrix(lightPos Point, sceneCenter Point) Matrix4x4 {
	// Create light view matrix (look at scene center from light position)
	viewMatrix := CreateLookAtMatrix(lightPos, sceneCenter, Point{X: 0, Y: 1, Z: 0})
	
	// Create orthographic projection for shadow map
	// Adjust size based on scene bounds
	size := 50.0
	near := 0.1
	far := 200.0
	projMatrix := CreateOrthographicMatrix(-size, size, -size, size, near, far)
	
	// Combine matrices
	return projMatrix.Multiply(viewMatrix)
}

// renderShadowPass renders the scene from light's perspective to generate shadow map
func (r *OpenGLRenderer) renderShadowPass(scene *Scene) {
	if !r.enableShadows || r.LightingSystem == nil || len(r.LightingSystem.Lights) == 0 {
		return
	}

	// Use first light for shadows
	light := r.LightingSystem.Lights[0]
	sceneCenter := Point{X: 0, Y: 0, Z: 0} // Could calculate from scene bounds

	// Calculate light space matrix
	r.shadowLightMatrix = r.calculateLightSpaceMatrix(light.Position, sceneCenter)

	// Bind shadow FBO
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.shadowFBO)
	gl.Viewport(0, 0, int32(r.shadowResolution), int32(r.shadowResolution))
	gl.Clear(gl.DEPTH_BUFFER_BIT)

	// Use shadow shader
	gl.UseProgram(r.shadowProgram)

	// Upload light space matrix
	var m [16]float32
	for i := 0; i < 16; i++ {
		m[i] = float32(r.shadowLightMatrix.M[i])
	}
	gl.UniformMatrix4fv(r.shadowUniformLightSpaceMatrix, 1, false, &m[0])

	// Render all scene nodes (depth only)
	nodes := scene.GetRenderableNodes()
	for _, node := range nodes {
		worldMatrix := node.Transform.GetWorldMatrix()
		r.renderNodeShadow(node, worldMatrix)
	}

	// Restore default framebuffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, int32(r.width), int32(r.height))
}

// renderNodeShadow renders a node for shadow pass (depth only)
func (r *OpenGLRenderer) renderNodeShadow(node *SceneNode, worldMatrix Matrix4x4) {
	switch obj := node.Object.(type) {
	case *Mesh:
		r.renderMeshShadow(obj, worldMatrix)
	case *LODGroup:
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil && len(currentMesh.Vertices) > 0 && len(currentMesh.Indices) > 0 {
			r.renderMeshShadow(currentMesh, worldMatrix)
		}
	case *LODGroupWithTransitions:
		currentMesh := obj.GetCurrentMesh()
		if currentMesh != nil && len(currentMesh.Vertices) > 0 && len(currentMesh.Indices) > 0 {
			r.renderMeshShadow(currentMesh, worldMatrix)
		}
	// Skip lines, points, etc. for shadow pass
	}
}

// renderMeshShadow renders a mesh in shadow pass
func (r *OpenGLRenderer) renderMeshShadow(mesh *Mesh, worldMatrix Matrix4x4) {
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return
	}

	// Upload model matrix
	var m [16]float32
	for i := 0; i < 16; i++ {
		m[i] = float32(worldMatrix.M[i])
	}
	gl.UniformMatrix4fv(r.shadowUniformModel, 1, false, &m[0])

	// Render mesh using existing VAO (reuse regular mesh VAO for shadow pass)
	// We'll create a simple vertex buffer with just positions
	positions := make([]float32, 0, len(mesh.Indices)*3)
	for _, idx := range mesh.Indices {
		if idx < len(mesh.Vertices) {
			v := worldMatrix.TransformPoint(mesh.Vertices[idx])
			positions = append(positions, float32(v.X), float32(v.Y), float32(v.Z))
		}
	}

	if len(positions) == 0 {
		return
	}

	// Use a temporary VAO for shadow rendering (position-only)
	var shadowVAO, shadowVBO uint32
	gl.GenVertexArrays(1, &shadowVAO)
	gl.BindVertexArray(shadowVAO)

	gl.GenBuffers(1, &shadowVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, shadowVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(positions)*4, gl.Ptr(positions), gl.STREAM_DRAW)

	// Position attribute (location 0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Draw
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(positions)/3))

	// Cleanup temporary VAO/VBO
	gl.DeleteVertexArrays(1, &shadowVAO)
	gl.DeleteBuffers(1, &shadowVBO)
}
