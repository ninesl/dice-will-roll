package render

import (
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Vector3 represents a 3D vector
type Vector3 struct {
	X, Y, Z float32
}

// Vector2 represents a 2D vector
type Vector2 struct {
	X, Y float32
}

// RockInstance represents a single rock in 3D space
type RockInstance struct {
	Position Vector3 // 3D world position
	Velocity Vector3 // 3D velocity for movement
	Rotation Vector3 // Current rotation angles
	RotSpeed Vector3 // Rotation speed
	Scale    float32 // Size multiplier
	Seed     float32 // Unique seed for shape generation
	SpriteID int     // Which sprite variant to use
}

// Camera represents the 3D camera
type Camera struct {
	Position Vector3
	Forward  Vector3
	Up       Vector3
	Right    Vector3
	FOV      float32 // Field of view in radians
	Near     float32
	Far      float32
}

// RocksRenderer manages efficient rendering of millions of rocks
type RocksRenderer struct {
	shader      *ebiten.Shader
	rockSprites []*ebiten.Image
	rocks       []RockInstance
	camera      Camera

	// Rendering parameters
	spriteSize  int
	numVariants int
	maxVisible  int

	// Frustum culling
	frustumPlanes [6]Vector4 // 6 frustum planes for culling

	// Batching
	vertices []ebiten.Vertex
	indices  []uint16

	// Performance tracking
	visibleCount int
	totalRocks   int
}

// Vector4 represents a 4D vector (used for planes)
type Vector4 struct {
	X, Y, Z, W float32
}

// RocksConfig holds configuration for rock generation
type RocksConfig struct {
	TotalRocks    int
	SpriteSize    int
	NumVariants   int
	MaxVisible    int
	WorldSize     float32
	MinRockSize   float32
	MaxRockSize   float32
	MovementSpeed float32
}

// NewRocksRenderer creates a new rocks rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()
	r := &RocksRenderer{
		shader:      shaderMap[shaders.RocksShaderKey],
		spriteSize:  config.SpriteSize,
		numVariants: config.NumVariants,
		maxVisible:  config.MaxVisible,
		totalRocks:  config.TotalRocks,
		vertices:    make([]ebiten.Vertex, 0, config.MaxVisible*4),
		indices:     make([]uint16, 0, config.MaxVisible*6),
	}

	// Initialize camera
	r.camera = Camera{
		Position: Vector3{0, 0, 0},
		Forward:  Vector3{0, 0, 1},
		Up:       Vector3{0, 1, 0},
		Right:    Vector3{1, 0, 0},
		FOV:      math.Pi / 3, // 60 degrees
		Near:     0.1,
		Far:      1000.0,
	}

	// Generate rock sprites
	r.generateRockSprites()

	// Generate rock instances
	r.generateRocks(config)

	return r
}

// generateRockSprites creates sprite variants using the SDF shader
func (r *RocksRenderer) generateRockSprites() {
	r.rockSprites = make([]*ebiten.Image, r.numVariants)

	for i := 0; i < r.numVariants; i++ {
		sprite := ebiten.NewImage(r.spriteSize, r.spriteSize)

		// Set shader uniforms for this variant
		uniforms := map[string]interface{}{
			"RockSeed":        float32(i) * 123.456,
			"RockSize":        0.4 + rand.Float32()*0.2,  // 0.4 to 0.6
			"Roughness":       0.3 + rand.Float32()*0.4,  // 0.3 to 0.7
			"LightDirection":  []float32{0.5, 0.5, -1.0}, // 3D light direction
			"AmbientLight":    0.3,
			"DiffuseStrength": 0.7,
			"BaseColor":       []float32{0.6, 0.5, 0.4}, // Rock color
			"ColorVariation":  0.2,
			"Time":            0.0,
		}

		// Render sprite using shader
		opts := &ebiten.DrawRectShaderOptions{}
		opts.Uniforms = uniforms
		sprite.DrawRectShader(r.spriteSize, r.spriteSize, r.shader, opts)

		r.rockSprites[i] = sprite
	}
}

// generateRocks creates the initial rock instances
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	r.rocks = make([]RockInstance, config.TotalRocks)

	for i := 0; i < config.TotalRocks; i++ {
		// Random position in world space - bias towards positive Z (in front of camera)
		pos := Vector3{
			X: (rand.Float32() - 0.5) * config.WorldSize,
			Y: (rand.Float32() - 0.5) * config.WorldSize,
			Z: rand.Float32() * config.WorldSize, // 0 to WorldSize (in front of camera)
		}

		// Random velocity for DVD screensaver effect
		vel := Vector3{
			X: (rand.Float32() - 0.5) * config.MovementSpeed,
			Y: (rand.Float32() - 0.5) * config.MovementSpeed,
			Z: (rand.Float32() - 0.5) * config.MovementSpeed,
		}

		// Random rotation speed
		rotSpeed := Vector3{
			X: (rand.Float32() - 0.5) * 2.0,
			Y: (rand.Float32() - 0.5) * 2.0,
			Z: (rand.Float32() - 0.5) * 2.0,
		}

		r.rocks[i] = RockInstance{
			Position: pos,
			Velocity: vel,
			Rotation: Vector3{0, 0, 0},
			RotSpeed: rotSpeed,
			Scale:    config.MinRockSize + rand.Float32()*(config.MaxRockSize-config.MinRockSize),
			Seed:     rand.Float32() * 1000.0,
			SpriteID: rand.Intn(r.numVariants),
		}
	}
}

// Update updates rock positions and rotations
func (r *RocksRenderer) Update(deltaTime float32, worldBounds float32) {
	for i := range r.rocks {
		rock := &r.rocks[i]

		// Update position
		rock.Position.X += rock.Velocity.X * deltaTime
		rock.Position.Y += rock.Velocity.Y * deltaTime
		rock.Position.Z += rock.Velocity.Z * deltaTime

		// Update rotation
		rock.Rotation.X += rock.RotSpeed.X * deltaTime
		rock.Rotation.Y += rock.RotSpeed.Y * deltaTime
		rock.Rotation.Z += rock.RotSpeed.Z * deltaTime

		// DVD screensaver bouncing
		halfBounds := worldBounds * 0.5
		if rock.Position.X > halfBounds || rock.Position.X < -halfBounds {
			rock.Velocity.X = -rock.Velocity.X
			rock.Position.X = clamp(rock.Position.X, -halfBounds, halfBounds)
		}
		if rock.Position.Y > halfBounds || rock.Position.Y < -halfBounds {
			rock.Velocity.Y = -rock.Velocity.Y
			rock.Position.Y = clamp(rock.Position.Y, -halfBounds, halfBounds)
		}
		if rock.Position.Z > halfBounds || rock.Position.Z < -halfBounds {
			rock.Velocity.Z = -rock.Velocity.Z
			rock.Position.Z = clamp(rock.Position.Z, -halfBounds, halfBounds)
		}
	}
}

// UpdateCamera updates the camera position and orientation
func (r *RocksRenderer) UpdateCamera(position Vector3, forward Vector3, up Vector3) {
	r.camera.Position = position
	r.camera.Forward = normalizeVector3(forward)
	r.camera.Up = normalizeVector3(up)
	r.camera.Right = normalizeVector3(cross(r.camera.Forward, r.camera.Up))

	// Recalculate frustum planes
	r.calculateFrustumPlanes()
}

// calculateFrustumPlanes calculates the 6 frustum planes for culling
func (r *RocksRenderer) calculateFrustumPlanes() {
	// This is a simplified frustum calculation
	// In a real implementation, you'd use the full view-projection matrix

	// For now, we'll use a simple distance-based culling
	// Real frustum culling would be more complex
}

// frustumCull performs frustum culling and returns visible rocks
func (r *RocksRenderer) frustumCull() []int {
	visible := make([]int, 0, r.maxVisible)

	for i, rock := range r.rocks {
		// Simple distance-based culling for now
		dist := distance(rock.Position, r.camera.Position)
		if dist < r.camera.Far {
			visible = append(visible, i)
			if len(visible) >= r.maxVisible {
				break
			}
		}
	}

	// Sort by distance (back to front for proper alpha blending)
	sort.Slice(visible, func(i, j int) bool {
		distI := distance(r.rocks[visible[i]].Position, r.camera.Position)
		distJ := distance(r.rocks[visible[j]].Position, r.camera.Position)
		return distI > distJ
	})

	r.visibleCount = len(visible)
	return visible
}

// projectToScreen projects a 3D point to screen coordinates
func (r *RocksRenderer) projectToScreen(worldPos Vector3, screenWidth, screenHeight int) (Vector2, float32, bool) {
	// Transform to camera space
	relPos := Vector3{
		X: worldPos.X - r.camera.Position.X,
		Y: worldPos.Y - r.camera.Position.Y,
		Z: worldPos.Z - r.camera.Position.Z,
	}

	// Project to camera space
	camX := dot3(relPos, r.camera.Right)
	camY := dot3(relPos, r.camera.Up)
	camZ := dot3(relPos, r.camera.Forward)

	// Check if behind camera
	if camZ <= r.camera.Near {
		return Vector2{}, 0, false
	}

	// Perspective projection
	halfFOV := r.camera.FOV * 0.5
	scale := 1.0 / float32(math.Tan(float64(halfFOV)))

	projX := camX * scale / camZ
	projY := camY * scale / camZ

	// Convert to screen coordinates
	screenX := (projX + 1.0) * float32(screenWidth) * 0.5
	screenY := (1.0 - projY) * float32(screenHeight) * 0.5

	return Vector2{X: screenX, Y: screenY}, camZ, true
}

// Draw renders all visible rocks
func (r *RocksRenderer) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Get visible rocks
	visibleRocks := r.frustumCull()

	// Debug: Force some rocks to be visible if none are found
	if len(visibleRocks) == 0 {
		// Add first 100 rocks for debugging
		for i := 0; i < min(100, len(r.rocks)); i++ {
			visibleRocks = append(visibleRocks, i)
		}
	}

	// Group by sprite type for batching
	batches := make(map[int][]int)
	for _, rockIdx := range visibleRocks {
		spriteID := r.rocks[rockIdx].SpriteID
		batches[spriteID] = append(batches[spriteID], rockIdx)
	}

	// Render each batch
	for spriteID, rockIndices := range batches {
		r.renderBatch(screen, spriteID, rockIndices, screenWidth, screenHeight)
	}
}

// renderBatch renders a batch of rocks with the same sprite
func (r *RocksRenderer) renderBatch(screen *ebiten.Image, spriteID int, rockIndices []int, screenWidth, screenHeight int) {
	if len(rockIndices) == 0 {
		return
	}

	// Clear vertex and index buffers
	r.vertices = r.vertices[:0]
	r.indices = r.indices[:0]

	sprite := r.rockSprites[spriteID]
	spriteSize := float32(r.spriteSize)

	for _, rockIdx := range rockIndices {
		rock := &r.rocks[rockIdx]

		// Project to screen
		screenPos, depth, visible := r.projectToScreen(rock.Position, screenWidth, screenHeight)
		if !visible {
			continue
		}

		// Calculate screen size based on distance and rock scale
		screenSize := (spriteSize * rock.Scale * 100.0) / depth
		if screenSize < 1.0 {
			continue // Too small to render
		}
		// Create quad vertices
		halfSize := screenSize * 0.5

		// Vertex indices for this quad
		baseIdx := uint16(len(r.vertices))

		// Add vertices (clockwise winding)
		r.vertices = append(r.vertices,
			ebiten.Vertex{
				DstX: screenPos.X - halfSize, DstY: screenPos.Y - halfSize,
				SrcX: 0, SrcY: 0,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			ebiten.Vertex{
				DstX: screenPos.X + halfSize, DstY: screenPos.Y - halfSize,
				SrcX: spriteSize, SrcY: 0,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			ebiten.Vertex{
				DstX: screenPos.X + halfSize, DstY: screenPos.Y + halfSize,
				SrcX: spriteSize, SrcY: spriteSize,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			ebiten.Vertex{
				DstX: screenPos.X - halfSize, DstY: screenPos.Y + halfSize,
				SrcX: 0, SrcY: spriteSize,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
		)

		// Add indices for two triangles
		r.indices = append(r.indices,
			baseIdx, baseIdx+1, baseIdx+2,
			baseIdx, baseIdx+2, baseIdx+3,
		)

	}

	// Render the batch
	if len(r.vertices) > 0 {
		opts := &ebiten.DrawTrianglesOptions{}
		opts.Filter = ebiten.FilterLinear
		screen.DrawTriangles(r.vertices, r.indices, sprite, opts)
	}
}

// GetStats returns rendering statistics
func (r *RocksRenderer) GetStats() (visible, total int) {
	return r.visibleCount, r.totalRocks
}

// Utility functions
func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func distance(a, b Vector3) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func normalizeVector3(v Vector3) Vector3 {
	length := float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
	if length == 0 {
		return Vector3{0, 0, 1}
	}
	return Vector3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

func cross(a, b Vector3) Vector3 {
	return Vector3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func dot3(a, b Vector3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}
