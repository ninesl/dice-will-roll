package render

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite caching system (reasonable numbers)
const (
	ROCK_TYPES        = 10                             // 8 different rock shapes
	ROTATION_VARIANTS = 50                             // 12 rotation variants per rock type (30° increments)
	TOTAL_SPRITES     = ROCK_TYPES * ROTATION_VARIANTS // 96 total sprites
	// TOTAL_SPRITES = 360
)

// SimpleRock represents a rock using cached sprites
type SimpleRock struct {
	Position    Vec2 // 2D screen position
	Velocity    Vec2 // 2D velocity (static, never changes)
	SpriteIndex int  // Index into sprite cache (static, never changes)
}

// RocksRenderer manages sprite-cached 3D SDF rock rendering
type RocksRenderer struct {
	shader     *ebiten.Shader
	sprites    [TOTAL_SPRITES]*ebiten.Image // Cached sprite variants
	rocks      []SimpleRock
	spriteSize int
	totalRocks int
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
		shader:     shaderMap[shaders.RocksShaderKey],
		spriteSize: config.SpriteSize,
		totalRocks: config.TotalRocks,
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

			// Calculate rotation angle (30° increments for 12 variants = 360°)
			rotationAngle := float32(rotation) * (2.0 * 3.14159265 / float32(ROTATION_VARIANTS))

			// Create varied rotations around different axes for more variety
			rotX := rotationAngle
			rotY := rotationAngle * 0.7 // Different speed for Y axis

			u := map[string]interface{}{
				"Time":       0.0,                 // No time-based animation for sprite generation
				"Mouse":      []float32{0.0, 0.0}, // No mouse interaction for cached sprites
				"Resolution": []float32{float32(r.spriteSize), float32(r.spriteSize)},
				"RotationX":  rotX, // Explicit X-axis rotation
				"RotationY":  rotY, // Explicit Y-axis rotation
			}

			opts := &ebiten.DrawRectShaderOptions{Uniforms: u}
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
		// Random position
		pos := Vec2{
			X: float64(rand.Float32() * config.WorldBoundsX),
			Y: float64(rand.Float32() * config.WorldBoundsY),
		}

		// Random static velocity
		vel := Vec2{
			X: float64((rand.Float32() - 0.5) * config.MovementSpeed),
			Y: float64((rand.Float32() - 0.5) * config.MovementSpeed),
		}

		// Pick one random sprite from the entire cache (0-499)
		spriteIndex := rand.Intn(TOTAL_SPRITES)

		r.rocks[i] = SimpleRock{
			Position:    pos,
			Velocity:    vel,
			SpriteIndex: spriteIndex,
		}
	}
}

// Update updates rock positions using their static velocity
func (r *RocksRenderer) Update() {
	for i := range r.rocks {
		rock := &r.rocks[i]

		rock.SpriteIndex = rock.SpriteIndex + 1
		if rock.SpriteIndex >= TOTAL_SPRITES {
			rock.SpriteIndex = 0
		}

		// Apply velocity to position
		rock.Position.X += rock.Velocity.X
		rock.Position.Y += rock.Velocity.Y

		// Wrap around screen boundaries
		if rock.Position.X < 0 {
			rock.Position.X += GAME_BOUNDS_X
		} else if rock.Position.X > GAME_BOUNDS_X {
			rock.Position.X -= GAME_BOUNDS_X
		}

		if rock.Position.Y < 0 {
			rock.Position.Y += GAME_BOUNDS_Y
		} else if rock.Position.Y > GAME_BOUNDS_Y {
			rock.Position.Y -= GAME_BOUNDS_Y
		}
	}
}

// Draw renders all rocks using simple sprite rendering
func (r *RocksRenderer) Draw(screen *ebiten.Image) {
	halfSize := float64(r.spriteSize) / 2

	for i := range r.rocks {
		rock := &r.rocks[i]

		// Get the sprite for this rock
		sprite := r.sprites[rock.SpriteIndex]

		// Draw at fixed size centered on position
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(
			rock.Position.X-halfSize,
			rock.Position.Y-halfSize,
		)
		opts.Filter = ebiten.FilterLinear

		screen.DrawImage(sprite, opts)
	}
}

// GetStats returns rendering statistics
func (r *RocksRenderer) GetStats() (visible, total int) {
	return r.totalRocks, r.totalRocks
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
