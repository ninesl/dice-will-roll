package render

import (
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite caching system (reasonable numbers)
const (
	ROCK_TYPES        = 8                              // 8 different rock shapes
	ROTATION_VARIANTS = 12                             // 12 rotation variants per rock type (30° increments)
	TOTAL_SPRITES     = ROCK_TYPES * ROTATION_VARIANTS // 96 total sprites
)

// SimpleRock represents a rock using cached sprites
type SimpleRock struct {
	X, Y          float32    // 2D screen position
	VelX, VelY    float32    // 2D velocity
	SpriteIndex   int        // Index into sprite cache
	RotationSpeed float32    // How fast to cycle through rotation variants
	RotationPhase float32    // Current rotation phase (0-1)
	Scale         float32    // Size multiplier
	Color         [3]float32 // RGB color tint
	Distance      float32    // Distance from camera (for sorting)
}

// RocksRenderer manages sprite-cached 3D SDF rock rendering
type RocksRenderer struct {
	shader       *ebiten.Shader
	sprites      [TOTAL_SPRITES]*ebiten.Image // Cached sprite variants
	rocks        []SimpleRock
	frameCounter int
	spriteSize   int

	// Performance optimization
	maxVisible   int
	visibleRocks []int // Indices of visible rocks

	// Batching for performance
	vertices []ebiten.Vertex
	indices  []uint16

	// Performance tracking
	totalRocks   int
	visibleCount int
}

// RocksConfig holds configuration for sprite-cached rock system
type RocksConfig struct {
	TotalRocks    int
	SpriteSize    int     // TILE_SIZE / 2
	WorldBoundsX  float32 // GAME_BOUNDS_X
	WorldBoundsY  float32 // GAME_BOUNDS_Y
	MaxVisible    int     // Maximum rocks to render per frame
	MinRockSize   float32
	MaxRockSize   float32
	MovementSpeed float32
}

// NewRocksRenderer creates a new sprite-cached rocks rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()
	r := &RocksRenderer{
		shader:       shaderMap[shaders.RocksShaderKey],
		spriteSize:   config.SpriteSize,
		totalRocks:   config.TotalRocks,
		maxVisible:   config.MaxVisible,
		frameCounter: 0,
		visibleRocks: make([]int, 0, config.MaxVisible),
		vertices:     make([]ebiten.Vertex, 0, config.MaxVisible*4),
		indices:      make([]uint16, 0, config.MaxVisible*6),
	}

	// Generate sprite cache
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	return r
}

// generateSprites creates cached sprite variants
func (r *RocksRenderer) generateSprites() {
	spriteIndex := 0

	for rockType := 0; rockType < ROCK_TYPES; rockType++ {
		for rotation := 0; rotation < ROTATION_VARIANTS; rotation++ {
			sprite := ebiten.NewImage(r.spriteSize, r.spriteSize)

			// Calculate rotation angle (30° increments)
			rotationAngle := float32(rotation) * (2.0 * math.Pi / ROTATION_VARIANTS)

			// Create varied rotations around different axes for more variety
			rotX := rotationAngle
			rotY := rotationAngle * 0.7 // Different speed for Y
			rotZ := rotationAngle * 0.3 // Different speed for Z

			uniforms := map[string]interface{}{
				"RockSeed":        float32(rockType) * 123.456,
				"RockSize":        0.4 + rand.Float32()*0.2, // 0.4 to 0.6
				"Roughness":       0.3 + rand.Float32()*0.4, // 0.3 to 0.7
				"RotationX":       rotX,
				"RotationY":       rotY,
				"RotationZ":       rotZ,
				"LightDirection":  []float32{0.5, 0.5, -1.0},
				"AmbientLight":    0.3,
				"DiffuseStrength": 0.7,
				"BaseColor":       []float32{0.6, 0.5, 0.4},
				"ColorVariation":  0.2,
			}

			opts := &ebiten.DrawRectShaderOptions{Uniforms: uniforms}
			sprite.DrawRectShader(r.spriteSize, r.spriteSize, r.shader, opts)
			r.sprites[spriteIndex] = sprite
			spriteIndex++
		}
	}
}

// generateRocks creates the initial rock instances
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	r.rocks = make([]SimpleRock, config.TotalRocks)

	for i := 0; i < config.TotalRocks; i++ {
		// Random 2D screen position
		x := rand.Float32() * config.WorldBoundsX
		y := rand.Float32() * config.WorldBoundsY

		// Random 2D velocity
		velX := (rand.Float32() - 0.5) * config.MovementSpeed
		velY := (rand.Float32() - 0.5) * config.MovementSpeed

		// Random sprite (rock type determines base sprite set)
		rockType := rand.Intn(ROCK_TYPES)
		baseIndex := rockType * ROTATION_VARIANTS

		// Random rotation speed and phase
		rotationSpeed := 0.001 + rand.Float32()*0.005 // Very slow rotation
		rotationPhase := rand.Float32()

		// Random scale
		scale := config.MinRockSize + rand.Float32()*(config.MaxRockSize-config.MinRockSize)

		// Random color variation
		baseColor := [3]float32{0.6, 0.5, 0.4}
		colorVar := rand.Float32()*0.3 - 0.15 // -0.15 to +0.15 variation
		color := [3]float32{
			clamp(baseColor[0]+colorVar, 0.2, 1.0),
			clamp(baseColor[1]+colorVar, 0.2, 1.0),
			clamp(baseColor[2]+colorVar, 0.2, 1.0),
		}

		r.rocks[i] = SimpleRock{
			X:             x,
			Y:             y,
			VelX:          velX,
			VelY:          velY,
			SpriteIndex:   baseIndex, // Will be updated in Update()
			RotationSpeed: rotationSpeed,
			RotationPhase: rotationPhase,
			Scale:         scale,
			Color:         color,
			Distance:      0.0,
		}
	}
}

// Update updates rock positions and sprite indices
func (r *RocksRenderer) Update() {
	for i := range r.rocks {
		rock := &r.rocks[i]

		// Update 2D position
		rock.X += rock.VelX
		rock.Y += rock.VelY

		// Screen edge bouncing
		if rock.X < 0 || rock.X > float32(GAME_BOUNDS_X) {
			rock.VelX = -rock.VelX
			if rock.X < 0 {
				rock.X = 0
			} else {
				rock.X = float32(GAME_BOUNDS_X)
			}
		}
		if rock.Y < 0 || rock.Y > float32(GAME_BOUNDS_Y) {
			rock.VelY = -rock.VelY
			if rock.Y < 0 {
				rock.Y = 0
			} else {
				rock.Y = float32(GAME_BOUNDS_Y)
			}
		}

		// Update rotation phase
		rock.RotationPhase += rock.RotationSpeed
		if rock.RotationPhase >= 1.0 {
			rock.RotationPhase -= 1.0
		}

		// Calculate sprite index based on rotation phase
		rockType := rock.SpriteIndex / ROTATION_VARIANTS
		rotationIndex := int(rock.RotationPhase*float32(ROTATION_VARIANTS)) % ROTATION_VARIANTS
		rock.SpriteIndex = rockType*ROTATION_VARIANTS + rotationIndex

		// Calculate distance from center for sorting
		centerX := float32(GAME_BOUNDS_X) * 0.5
		centerY := float32(GAME_BOUNDS_Y) * 0.5
		dx := rock.X - centerX
		dy := rock.Y - centerY
		rock.Distance = dx*dx + dy*dy
	}
	r.frameCounter++
}

// performCulling performs frustum culling and distance sorting
func (r *RocksRenderer) performCulling() {
	r.visibleRocks = r.visibleRocks[:0]

	// Screen bounds culling with margin
	margin := float32(r.spriteSize)
	for i, rock := range r.rocks {
		if rock.X >= -margin && rock.X <= float32(GAME_BOUNDS_X)+margin &&
			rock.Y >= -margin && rock.Y <= float32(GAME_BOUNDS_Y)+margin {
			r.visibleRocks = append(r.visibleRocks, i)

			if len(r.visibleRocks) >= r.maxVisible {
				break
			}
		}
	}

	// Sort by distance (back to front)
	sort.Slice(r.visibleRocks, func(i, j int) bool {
		return r.rocks[r.visibleRocks[i]].Distance > r.rocks[r.visibleRocks[j]].Distance
	})

	r.visibleCount = len(r.visibleRocks)
}

// Draw renders all visible rocks using batched sprite rendering
func (r *RocksRenderer) Draw(screen *ebiten.Image) {
	r.performCulling()

	// Group rocks by sprite for batching
	spriteBatches := make(map[int][]int)
	for _, rockIdx := range r.visibleRocks {
		spriteIdx := r.rocks[rockIdx].SpriteIndex
		spriteBatches[spriteIdx] = append(spriteBatches[spriteIdx], rockIdx)
	}

	// Render each batch
	for spriteIdx, rockIndices := range spriteBatches {
		r.renderBatch(screen, spriteIdx, rockIndices)
	}
}

// renderBatch renders a batch of rocks with the same sprite
func (r *RocksRenderer) renderBatch(screen *ebiten.Image, spriteIdx int, rockIndices []int) {
	if len(rockIndices) == 0 || spriteIdx >= len(r.sprites) {
		return
	}

	sprite := r.sprites[spriteIdx]
	spriteSize := float32(r.spriteSize)

	// Clear vertex and index buffers
	r.vertices = r.vertices[:0]
	r.indices = r.indices[:0]

	for _, rockIdx := range rockIndices {
		rock := &r.rocks[rockIdx]

		// Calculate screen size based on scale
		screenSize := spriteSize * rock.Scale
		halfSize := screenSize * 0.5

		// Vertex indices for this quad
		baseIdx := uint16(len(r.vertices))

		// Add vertices with color tinting
		r.vertices = append(r.vertices,
			ebiten.Vertex{
				DstX: rock.X - halfSize, DstY: rock.Y - halfSize,
				SrcX: 0, SrcY: 0,
				ColorR: rock.Color[0], ColorG: rock.Color[1], ColorB: rock.Color[2], ColorA: 1,
			},
			ebiten.Vertex{
				DstX: rock.X + halfSize, DstY: rock.Y - halfSize,
				SrcX: spriteSize, SrcY: 0,
				ColorR: rock.Color[0], ColorG: rock.Color[1], ColorB: rock.Color[2], ColorA: 1,
			},
			ebiten.Vertex{
				DstX: rock.X + halfSize, DstY: rock.Y + halfSize,
				SrcX: spriteSize, SrcY: spriteSize,
				ColorR: rock.Color[0], ColorG: rock.Color[1], ColorB: rock.Color[2], ColorA: 1,
			},
			ebiten.Vertex{
				DstX: rock.X - halfSize, DstY: rock.Y + halfSize,
				SrcX: 0, SrcY: spriteSize,
				ColorR: rock.Color[0], ColorG: rock.Color[1], ColorB: rock.Color[2], ColorA: 1,
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
