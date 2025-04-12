package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	windowWidth  = 1280
	windowHeight = 720
	windowTitle  = "GoCraft Simple Demo - Faster Rendering" // Updated Title

	worldSizeX = 16
	worldSizeY = 8
	worldSizeZ = 16

	mouseSensitivity = 0.1
	moveSpeed        = 4.0 // Movement speed should feel consistent again
	playerHeight     = 1.8

	textureFile    = "terrain.png"
	atlasDimension = 16.0
)

var tileSize = float32(1.0 / atlasDimension)

const (
	BlockAir = iota
	BlockGrass
	BlockDirt
	BlockStone
)

// FaceType constants are no longer needed for drawing logic
// const ( FaceTop = iota ... )

var (
	window            *glfw.Window
	program           uint32
	blockTexture      uint32
	cameraPos         = mgl32.Vec3{}
	cameraFront       = mgl32.Vec3{0.0, 0.0, -1.0}
	cameraUp          = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraYaw         = -90.0
	cameraPitch       = 0.0
	firstMouse        = true
	lastX             = float64(windowWidth / 2.0)
	lastY             = float64(windowHeight / 2.0)
	deltaTime         = 0.0
	lastFrame         = 0.0
	worldData         [worldSizeX][worldSizeY][worldSizeZ]int
	cubeVAO           uint32
	cubeVBO           uint32
	modelLoc          int32
	viewLoc           int32
	projectionLoc     int32
	textureSamplerLoc int32
	tileStartXLoc     int32
	tileStartYLoc     int32
	tileSizeLoc       int32
	onGround          bool = false
)

var cubeVertices = []float32{ /* Vertex data remains exactly the same */
	// Back face (-Z)
	-0.5, -0.5, -0.5, 0.0, 0.0, 0.5, -0.5, -0.5, 1.0, 0.0, 0.5, 0.5, -0.5, 1.0, 1.0,
	0.5, 0.5, -0.5, 1.0, 1.0, -0.5, 0.5, -0.5, 0.0, 1.0, -0.5, -0.5, -0.5, 0.0, 0.0,
	// Front face (+Z)
	-0.5, -0.5, 0.5, 0.0, 0.0, 0.5, -0.5, 0.5, 1.0, 0.0, 0.5, 0.5, 0.5, 1.0, 1.0,
	0.5, 0.5, 0.5, 1.0, 1.0, -0.5, 0.5, 0.5, 0.0, 1.0, -0.5, -0.5, 0.5, 0.0, 0.0,
	// Left face (-X)
	-0.5, 0.5, 0.5, 1.0, 1.0, -0.5, 0.5, -0.5, 0.0, 1.0, -0.5, -0.5, -0.5, 0.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0, -0.5, -0.5, 0.5, 1.0, 0.0, -0.5, 0.5, 0.5, 1.0, 1.0,
	// Right face (+X)
	0.5, 0.5, 0.5, 0.0, 1.0, 0.5, 0.5, -0.5, 1.0, 1.0, 0.5, -0.5, -0.5, 1.0, 0.0,
	0.5, -0.5, -0.5, 1.0, 0.0, 0.5, -0.5, 0.5, 0.0, 0.0, 0.5, 0.5, 0.5, 0.0, 1.0,
	// Bottom face (-Y)
	-0.5, -0.5, -0.5, 0.0, 1.0, 0.5, -0.5, -0.5, 1.0, 1.0, 0.5, -0.5, 0.5, 1.0, 0.0,
	0.5, -0.5, 0.5, 1.0, 0.0, -0.5, -0.5, 0.5, 0.0, 0.0, -0.5, -0.5, -0.5, 0.0, 1.0,
	// Top face (+Y)
	-0.5, 0.5, -0.5, 0.0, 1.0, 0.5, 0.5, -0.5, 1.0, 1.0, 0.5, 0.5, 0.5, 1.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 0.0, -0.5, 0.5, 0.5, 0.0, 0.0, -0.5, 0.5, -0.5, 0.0, 1.0,
}

var vertexShaderSource = ` /* Shader source remains exactly the same */
    #version 410 core
    layout (location = 0) in vec3 aPos; layout (location = 1) in vec2 aTexCoord;
    uniform mat4 model; uniform mat4 view; uniform mat4 projection;
    out vec2 TexCoord;
    void main() { gl_Position = projection * view * model * vec4(aPos.x + 0.5, aPos.y + 0.5, aPos.z + 0.5, 1.0); TexCoord = aTexCoord; }
` + "\x00"

var fragmentShaderSource = ` /* Shader source remains exactly the same */
    #version 410 core
    out vec4 FragColor; in vec2 TexCoord;
    uniform sampler2D textureSampler; uniform float tileStartX; uniform float tileStartY; uniform float tileSize;
    void main() {
        vec2 atlasTexCoord; atlasTexCoord.x = tileStartX + TexCoord.x * tileSize; atlasTexCoord.y = tileStartY + (1.0 - TexCoord.y) * tileSize; // Flip Y
        FragColor = texture(textureSampler, atlasTexCoord); if(FragColor.a < 0.1) discard;
    }
` + "\x00"

func initGlfw() *glfw.Window { /* initGlfw remains exactly the same */
	log.Println("Initializing GLFW...")
	if err := glfw.Init(); err != nil {
		panic(fmt.Errorf("failed to initialize glfw: %w", err))
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	log.Println("Creating GLFW window...")
	win, err := glfw.CreateWindow(windowWidth, windowHeight, windowTitle, nil, nil)
	if err != nil {
		glfw.Terminate()
		panic(fmt.Errorf("failed to create glfw window: %w", err))
	}
	log.Println("Making context current...")
	win.MakeContextCurrent()
	win.SetFramebufferSizeCallback(framebufferSizeCallback)
	win.SetCursorPosCallback(mouseCallback)
	win.SetKeyCallback(keyCallback)
	win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	if glfw.RawMouseMotionSupported() {
		win.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}
	log.Println("GLFW Initialized.")
	return win
}

func initOpenGL() { /* initOpenGL remains exactly the same */
	log.Println("Initializing OpenGL...")
	if err := gl.Init(); err != nil {
		panic(fmt.Errorf("failed to initialize OpenGL: %w", err))
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	program = gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	var success int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &success)
	if success == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		logMsg := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(logMsg))
		panic(fmt.Errorf("failed to link program: %v", logMsg))
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	log.Println("Shaders compiled and linked.")
	modelLoc = gl.GetUniformLocation(program, gl.Str("model\x00"))
	viewLoc = gl.GetUniformLocation(program, gl.Str("view\x00"))
	projectionLoc = gl.GetUniformLocation(program, gl.Str("projection\x00"))
	textureSamplerLoc = gl.GetUniformLocation(program, gl.Str("textureSampler\x00"))
	tileStartXLoc = gl.GetUniformLocation(program, gl.Str("tileStartX\x00"))
	tileStartYLoc = gl.GetUniformLocation(program, gl.Str("tileStartY\x00"))
	tileSizeLoc = gl.GetUniformLocation(program, gl.Str("tileSize\x00"))
	gl.GenVertexArrays(1, &cubeVAO)
	gl.GenBuffers(1, &cubeVBO)
	gl.BindVertexArray(cubeVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, cubeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)
	stride := int32(5 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, nil)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	log.Println("VAO/VBO created.")
	var loadErr error
	blockTexture, loadErr = loadTexture(textureFile)
	if loadErr != nil {
		panic(fmt.Errorf("failed to load texture %s: %w", textureFile, loadErr))
	}
	log.Printf("Loaded texture atlas '%s' with ID %d\n", textureFile, blockTexture)
	gl.Enable(gl.DEPTH_TEST)
	gl.ClearColor(0.5, 0.8, 1.0, 1.0)
	log.Println("OpenGL Initialized.")
}

func compileShader(source string, shaderType uint32) (uint32, error) { /* compileShader remains exactly the same */
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)
	var success int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &success)
	if success == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		logMsg := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(logMsg))
		return 0, fmt.Errorf("failed to compile %v shader:\n%v", shaderType, logMsg)
	}
	return shader, nil
}

func setupWorld() { /* setupWorld remains exactly the same */
	fmt.Println("Generating simple world...")
	for x := 0; x < worldSizeX; x++ {
		for z := 0; z < worldSizeZ; z++ {
			height := worldSizeY / 2
			if height >= 0 && height < worldSizeY {
				worldData[x][height][z] = BlockGrass
			}
			for y := 0; y < height; y++ {
				if y < height-2 {
					worldData[x][y][z] = BlockStone
				} else {
					worldData[x][y][z] = BlockDirt
				}
			}
			for y := height + 1; y < worldSizeY; y++ {
				worldData[x][y][z] = BlockAir
			}
		}
	}
	fmt.Println("World generation complete.")
}

func framebufferSizeCallback(w *glfw.Window, width int, height int) { /* remains exactly the same */
	gl.Viewport(0, 0, int32(width), int32(height))
}

func mouseCallback(w *glfw.Window, xpos float64, ypos float64) { /* remains exactly the same */
	if firstMouse {
		lastX, lastY, firstMouse = xpos, ypos, false
	}
	xoffset, yoffset := xpos-lastX, lastY-ypos
	lastX, lastY = xpos, ypos
	xoffset, yoffset = xoffset*mouseSensitivity, yoffset*mouseSensitivity
	cameraYaw, cameraPitch = cameraYaw+xoffset, cameraPitch+yoffset
	if cameraPitch > 89.0 {
		cameraPitch = 89.0
	}
	if cameraPitch < -89.0 {
		cameraPitch = -89.0
	}
	yawRad32, pitchRad32 := mgl32.DegToRad(float32(cameraYaw)), mgl32.DegToRad(float32(cameraPitch))
	cosYaw, cosPitch := math.Cos(float64(yawRad32)), math.Cos(float64(pitchRad32))
	sinPitch, sinYaw := math.Sin(float64(pitchRad32)), math.Sin(float64(yawRad32))
	front := mgl32.Vec3{float32(cosYaw * cosPitch), float32(sinPitch), float32(sinYaw * cosPitch)}
	cameraFront = front.Normalize()
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) { /* remains exactly the same */
	if key == glfw.KeyEscape && action == glfw.Press {
		w.SetShouldClose(true)
	}
}

// --- Main Loop ---

func main() {
	runtime.LockOSThread()
	initialX := float32(worldSizeX / 2)
	initialZ := float32(worldSizeZ / 2)
	cameraPos = mgl32.Vec3{initialX, float32(worldSizeY), initialZ}
	window = initGlfw()
	defer glfw.Terminate()
	initOpenGL()
	if glErr := gl.GetError(); glErr != gl.NO_ERROR {
		log.Fatalf("OpenGL error after initOpenGL: 0x%x\n", glErr)
	}
	setupWorld()
	startX := int(math.Floor(float64(cameraPos.X())))
	startZ := int(math.Floor(float64(cameraPos.Z())))
	groundY := 0
	for y := worldSizeY - 1; y >= 0; y-- {
		if isSolid(startX, y, startZ) {
			groundY = y
			break
		}
	}
	cameraPos[1] = float32(groundY) + 0.5 + playerHeight
	log.Println("Entering main loop...")

	// faceOrder no longer needed for drawing
	// faceOrder := []int{FaceBack, FaceFront, FaceLeft, FaceRight, FaceBottom, FaceTop}

	for !window.ShouldClose() {
		currentFrame := glfw.GetTime()
		deltaTime = currentFrame - lastFrame
		if deltaTime > 0.1 {
			deltaTime = 0.1
		}
		lastFrame = currentFrame
		processInput(window)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(program)
		view := mgl32.LookAtV(cameraPos, cameraPos.Add(cameraFront), cameraUp)
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
		fbWidth, fbHeight := window.GetFramebufferSize()
		if fbHeight == 0 {
			fbHeight = 1
		}
		aspectRatio := float32(fbWidth) / float32(fbHeight)
		projection := mgl32.Perspective(mgl32.DegToRad(45.0), aspectRatio, 0.1, 100.0)
		gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, blockTexture)
		gl.Uniform1i(textureSamplerLoc, 0)
		gl.Uniform1f(tileSizeLoc, tileSize)
		gl.BindVertexArray(cubeVAO)

		// --- Draw World (Reverted to 1 Draw Call Per Block) ---
		for x := 0; x < worldSizeX; x++ {
			for y := 0; y < worldSizeY; y++ {
				for z := 0; z < worldSizeZ; z++ {
					blockType := worldData[x][y][z]
					if blockType != BlockAir {
						// Set Model matrix once per block
						model := mgl32.Translate3D(float32(x), float32(y), float32(z))
						gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

						// Get tile coordinates for this block type (all faces will use this)
						tileX, tileY := getBlockTileCoords(blockType) // Using simplified function
						currentTileStartX := float32(tileX) * tileSize
						currentTileStartY := float32(tileY) * tileSize

						// Set the tile uniforms for the fragment shader *once per block*
						gl.Uniform1f(tileStartXLoc, currentTileStartX)
						gl.Uniform1f(tileStartYLoc, currentTileStartY)

						// Draw all 36 vertices (the whole cube) at once
						gl.DrawArrays(gl.TRIANGLES, 0, 36)

						// Removed the inner loop for drawing faces separately
						// for i := 0; i < 6; i++ { ... }
					}
				}
			}
		}
		gl.BindVertexArray(0)
		window.SwapBuffers()
		glfw.PollEvents()
	}
	log.Println("Exiting main loop.")
}

// --- Helper Functions ---

// RENAMED and SIMPLIFIED: Returns single tile coord based only on block type
func getBlockTileCoords(blockType int) (int, int) {
	switch blockType {
	case BlockGrass:
		// All faces of Grass block will now use the Grass Top texture
		return 0, 0 // Grass Top Tile (0,0)
	case BlockDirt:
		// Dirt blocks will use the Stone texture (as requested previously)
		return 1, 0 // Stone Tile (1,0)
	case BlockStone:
		return 1, 0 // Stone Tile (1,0)
	default: // Fallback for Air or unknown blocks
		return 15, 15
	}
}

func loadTexture(filename string) (uint32, error) { /* loadTexture remains exactly the same */
	log.Printf("Loading texture: %s\n", filename)
	file, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()
	img, format, err := image.Decode(file)
	if err != nil {
		return 0, fmt.Errorf("failed to decode image %s: %w", filename, err)
	}
	if format != "png" {
		log.Printf("Warning: Texture %s is format %s, not png", filename, format)
	}
	log.Printf("Texture decoded: %s, Format: %s, Bounds: %v\n", filename, format, img.Bounds())
	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride for image %s (stride: %d, expected: %d)", filename, rgba.Stride, rgba.Rect.Size().X*4)
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	log.Printf("Texture converted to RGBA for %s\n", filename)
	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	log.Printf("Uploading texture data to GPU for %s (ID: %d)\n", filename, textureID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	if glErr := gl.GetError(); glErr != gl.NO_ERROR {
		return 0, fmt.Errorf("opengl error after TexImage2D for %s: 0x%x", filename, glErr)
	}
	gl.GenerateMipmap(gl.TEXTURE_2D)
	if glErr := gl.GetError(); glErr != gl.NO_ERROR {
		log.Printf("Warning: OpenGL error after GenerateMipmap for %s: 0x%x\n", filename, glErr)
	}
	gl.BindTexture(gl.TEXTURE_2D, 0)
	log.Printf("Texture loading complete for %s\n", filename)
	return textureID, nil
}

func isSolid(x, y, z int) bool { /* remains exactly the same */
	if x < 0 || x >= worldSizeX || y < 0 || y >= worldSizeY || z < 0 || z >= worldSizeZ {
		return false
	}
	return worldData[x][y][z] != BlockAir
}

func processInput(window *glfw.Window) { /* processInput remains exactly the same */
	dt := float32(deltaTime)
	currentSpeed := float32(moveSpeed) * dt
	right := cameraFront.Cross(cameraUp).Normalize()
	forward := cameraFront
	if window.GetKey(glfw.KeyW) == glfw.Press {
		cameraPos = cameraPos.Add(forward.Mul(currentSpeed))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		cameraPos = cameraPos.Sub(forward.Mul(currentSpeed))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		cameraPos = cameraPos.Sub(right.Mul(currentSpeed))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		cameraPos = cameraPos.Add(right.Mul(currentSpeed))
	}
	feetY, feetX, feetZ := cameraPos.Y()-playerHeight, cameraPos.X(), cameraPos.Z()
	blockBelowX, blockBelowY, blockBelowZ := int(math.Floor(float64(feetX))), int(math.Floor(float64(feetY-0.1))), int(math.Floor(float64(feetZ)))
	onGround = isSolid(blockBelowX, blockBelowY, blockBelowZ)
	if window.GetKey(glfw.KeySpace) == glfw.Press {
		cameraPos = cameraPos.Add(cameraUp.Mul(currentSpeed))
		onGround = false
	}
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press || window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		potentialFeetY := feetY - currentSpeed
		blockBelowNextX, blockBelowNextY, blockBelowNextZ := int(math.Floor(float64(feetX))), int(math.Floor(float64(potentialFeetY))), int(math.Floor(float64(feetZ)))
		if !isSolid(blockBelowNextX, blockBelowNextY, blockBelowNextZ) {
			cameraPos = cameraPos.Sub(cameraUp.Mul(currentSpeed))
			onGround = false
		} else {
			cameraPos[1] = float32(blockBelowNextY+1) + playerHeight + 0.01
			onGround = true
		}
	}
}
