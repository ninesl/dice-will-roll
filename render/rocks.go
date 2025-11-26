package render

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite system
const (
	NUM_ROCK_TYPES    = 1                       // 2 rock types: different shapes
	DEGREES_PER_FRAME = 5                       // Degrees of rotation per transition frame
	ROTATION_FRAMES   = 360 / DEGREES_PER_FRAME // 360/5 = 72 frames (every 5 degrees) static spin

	DIRECTIONS_TO_SNAP = MAX_SLOPE * 2 // # of possible angles for SpriteSlopeX and SpriteSlopeY

	MAX_SLOPE int8 = 4
	MIN_SLOPE int8 = -MAX_SLOPE
	// SPEED_RANGE      = MAX_SLOPE*2 + 1 // 9 (from -4 to +4 inclusive)
)

// SimpleRock represents a rock using pre-extracted sprite frames
type SimpleRock struct {
	Position    Vec2   // 2D screen position
	SpriteIndex uint16 // Current rotation frame index (0-71)
	SlopeX      int8   // Current X speed component (-4 to +4)
	SlopeY      int8   // Current Y speed component (-4 to +4)

	// Transition system for smooth sprite rotation during direction changes
	SpriteSlopeX int8 // Visual speed X used during transition (gradually moves toward SpeedX)
	SpriteSlopeY int8 // Visual speed Y used during transition (gradually moves toward SpeedY)
}

// Updates the rock based on the current target transitions.
//
// will update it's state based on other params every Tick/time this is called
func (r *SimpleRock) Update() {
	r.UpdateTransition()
	// r.SpriteSlopeX = r.SlopeX
	// r.SpriteSlopeY = r.SlopeY
}

// if newY or newX is IDENTICAL to the the current value in the struct
//
// does the go compiler ignore this if hte func is called? I want to reuse this in BounceX and BounceY but it seems to be
// a wasted allocation if we were to assign newX newY each time?
func (r *SimpleRock) Bounce(newX int8, newY int8) {
	r.SlopeX = newX
	r.SlopeY = newY
}

// BounceX flips horizontal direction (bounce off vertical wall)
func (r *SimpleRock) BounceX() {
	// Instantly change actual direction
	r.SlopeX = -r.SlopeX
}

// BounceY flips vertical direction (bounce off horizontal wall)
func (r *SimpleRock) BounceY() {
	// Instantly change actual direction
	r.SlopeY = -r.SlopeY
}

// UpdateTransition handles the smooth sprite rotation during direction changes
func (r *SimpleRock) UpdateTransition() {
	// Gradually move transition speeds toward target speeds (one step per frame)

	// Update X component
	if r.SpriteSlopeX <= r.SlopeX+MAX_SLOPE {
		r.SpriteSlopeX++
	} else if r.SpriteSlopeX >= r.SlopeX+MAX_SLOPE {
		r.SpriteSlopeX--
	}

	// Update Y component
	if r.SpriteSlopeY <= r.SlopeY+MAX_SLOPE {
		r.SpriteSlopeY++
	} else if r.SpriteSlopeY >= r.SlopeY+MAX_SLOPE {
		r.SpriteSlopeY--
	}
}

// BounceTowardsAngle sets rock direction and speed based on target angle (0-360 degrees)
// Angle system: 0°=right, 90°=down, 180°=left, 270°=up (standard screen coordinates)
// Uses trigonometry to accurately map angles to SpeedX/SpeedY components
// func (r *SimpleRock) BounceTowardsAngle(angle int) {
// 	// Normalize angle to 0-360 range
// 	// angle = angle % 360
// 	if angle >= 360 || angle < 0 { // technically this is an assert
// 		panic("angle out of range")
// 	}

// 	// Calculate current angle for transition frame calculation
// 	currentAngle := int(math.Atan2(float64(r.SlopeY), float64(r.SlopeY)) * 180 / math.Pi)

// 	// Convert angle to radians for trigonometric functions
// 	angleRad := float64(angle) * math.Pi / 180.0

// 	// Calculate velocity components using trigonometry
// 	// cos gives X-component, sin gives Y-component
// 	speedX := math.Cos(angleRad)
// 	speedY := math.Sin(angleRad)

// 	// Scale to int8 range (-4 to +4) while maintaining ratio
// 	// const maxVal = float64(MAX_SLOPE) // 4

// 	// Scale both components so the larger one reaches maxVal
// 	maxComponent := math.Max(math.Abs(speedX), math.Abs(speedY))
// 	var targetSlopeX, targetSlopeY int8
// 	if maxComponent > 0 {
// 		scale := maxVal / maxComponent
// 		targetSlopeX = int8(math.Round(speedX * scale))
// 		targetSlopeY = int8(math.Round(speedY * scale))
// 	} else {
// 		// Edge case: shouldn't happen, but default to stationary
// 		targetSlopeX = 0
// 		targetSlopeY = 0
// 	}

// 	// Ensure at least one speed is non-zero to prevent truly stationary rocks
// 	if targetSlopeX == 0 && targetSlopeY == 0 {
// 		targetSlopeX = 1
// 		targetSlopeY = 0
// 	}

// 	// Calculate angle difference for transition frames
// 	angleDiff := angle - currentAngle
// 	if angleDiff < 0 {
// 		angleDiff = -angleDiff
// 	}
// 	if angleDiff > 180 {
// 		angleDiff = 360 - angleDiff
// 	}

// 	// Set new speeds immediately
// 	r.SlopeX = targetSlopeX
// 	r.SlopeY = targetSlopeY

// }

// RocksRenderer manages pre-extracted sprite rendering with ultra-fast array indexing
type RocksRenderer struct {
	shader *ebiten.Shader

	// 4D array of pre-extracted individual sprite images
	// [rockType][speedX_index][speedY_index][rotationFrame] -> sprite image
	// Direct O(1) lookup, no SubImage() calls during rendering
	// Size: [2][9][9][72] = 11,664 sprite pointers (many shared via deduplication)
	sprites [NUM_ROCK_TYPES][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*ebiten.Image

	Rocks          [NUM_ROCK_TYPES][]*SimpleRock // Rocks organized by type
	SpriteSize     int
	FSpriteSize    float64
	totalRocks     int
	ActiveRockType int
}

// RocksConfig holds configuration for rock system
type RocksConfig struct {
	TotalRocks   int
	SpriteSize   int
	WorldBoundsX float32
	WorldBoundsY float32
}

// NewRocksRenderer creates a new ultra-fast sprite rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()
	r := &RocksRenderer{
		shader:      shaderMap[shaders.RocksShaderKey],
		SpriteSize:  config.SpriteSize,
		FSpriteSize: float64(config.SpriteSize),
		totalRocks:  config.TotalRocks,
	}

	// Generate and pre-extract all sprite frames
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	return r
}

// generateSprites creates individual sprite images for each unique angle
// Multiple array slots may point to the same sprite if they have the same angle (deduplication)
func (r *RocksRenderer) generateSprites() {
	// Track generated sprites by angle for deduplication
	// Key: angle in degrees, Value: [rockType][frameIdx] -> sprite

	// genSprites := make([][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]map[int]*ebiten.Image, 0, NUM_ROCK_TYPES)
	genSprites := [NUM_ROCK_TYPES][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*ebiten.Image{}

	for rockType := range NUM_ROCK_TYPES {
		var innerDark, innerLight, outerDark, outerLight Vec3
		switch rockType {
		case 0: // Dark gray
			innerDark = KageColor(40, 40, 42)
			innerLight = KageColor(80, 82, 85)
			outerDark = KageColor(70, 72, 75)
			outerLight = KageColor(100, 102, 105)
		case 1: // Brown
			innerDark = KageColor(60, 40, 30)
			innerLight = KageColor(100, 70, 50)
			outerDark = KageColor(80, 60, 45)
			outerLight = KageColor(120, 90, 70)
		}

		XSnap := [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*ebiten.Image{}
		genSprites[rockType] = XSnap
		for XSnapIdx := range DIRECTIONS_TO_SNAP {
			YSnap := [DIRECTIONS_TO_SNAP]*ebiten.Image{}
			genSprites[rockType][XSnapIdx] = YSnap

			angleX := (int(XSnapIdx) + 1) * (360 / int(DIRECTIONS_TO_SNAP))

			// angleX := math.Atan2(float64(y), float64(x))
			// angleDeg := angleRad * 180.0 / math.Pi
			// if angleDeg < 0 {
			// 	angleDeg += 360
			// }

			for YSnapIdx := range DIRECTIONS_TO_SNAP {

				angleY := (int(YSnapIdx) + 1) * (360 / int(DIRECTIONS_TO_SNAP))

				// imageMap := map[int]*ebiten.Image{}

				//TODO: make this a spritesheet and get the subrects
				// for frameDegrees := 0; frameDegrees <= 360; frameDegrees += DEGREES_PER_FRAME {

				sprite := ebiten.NewImage(r.SpriteSize, r.SpriteSize)

				// Calculate rotation angle (0 to 2π over ROTATION_FRAMES)
				// rotationAngle := float32(frameDegrees) * (2.0 * 3.14159265 / float32(ROTATION_FRAMES))

				u := map[string]interface{}{
					"Time":       0.0,
					"Resolution": []float32{float32(r.SpriteSize), float32(r.SpriteSize)},
					"Mouse":      Vec2{X: 0.0, Y: 0.0}.KageVec2(),
					"RotationX":  angleX,
					"RotationY":  angleY,
					// "RotationZ":       rotationAngle,
					"InnerColorDark":  innerDark.KageVec3(),
					"InnerColorLight": innerLight.KageVec3(),
					"OuterColorDark":  outerDark.KageVec3(),
					"OuterColorLight": outerLight.KageVec3(),
				}

				opts := &ebiten.DrawRectShaderOptions{Uniforms: u}
				sprite.DrawRectShader(r.SpriteSize, r.SpriteSize, r.shader, opts)

				if XSnapIdx >= DIRECTIONS_TO_SNAP {
					continue
				} else if YSnapIdx >= DIRECTIONS_TO_SNAP {
					continue
				}

				genSprites[rockType][XSnapIdx][YSnapIdx] = sprite
			}
		}
		//TODO: should make a single spritesheet and then assign each degree key in map
		// to the subrect of each to save memory allocations?

	}
}

// generateRocks creates the initial rock instances
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	rocksPerType := config.TotalRocks / NUM_ROCK_TYPES
	remainder := config.TotalRocks % NUM_ROCK_TYPES

	for rockType := 0; rockType < NUM_ROCK_TYPES; rockType++ {
		numRocks := rocksPerType
		if rockType < remainder {
			numRocks++
		}

		r.Rocks[rockType] = make([]*SimpleRock, numRocks)

		for i := 0; i < numRocks; i++ {
			// Random position
			pos := Vec2{
				X: float64(rand.Float32() * config.WorldBoundsX),
				Y: float64(rand.Float32() * config.WorldBoundsY),
			}

			// Pick random rotation frame
			spriteIndex := uint16(rand.Intn(ROTATION_FRAMES))

			// Pick random speed (avoid 0,0)
			var speedX, speedY int8
			for speedX == 0 && speedY == 0 {
				speedX = int8(rand.Intn(int(DIRECTIONS_TO_SNAP)))
				speedY = int8(rand.Intn(int(DIRECTIONS_TO_SNAP)))
			}

			r.Rocks[rockType][i] = &SimpleRock{
				Position:     pos,
				SpriteIndex:  spriteIndex,
				SlopeX:       speedX,
				SlopeY:       speedY,
				SpriteSlopeX: speedX, // Initialize transition to match current
				SpriteSlopeY: speedY, // Initialize transition to match current
			}
		}
	}
}

// DrawRocks renders all rocks with ultra-fast direct array access
func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	for rockType := range NUM_ROCK_TYPES {
		for _, rock := range r.Rocks[rockType] {
			// Always use transition speeds for sprite selection (they match target speeds when not transitioning)
			// speedX := rock.TransitionSpeedX
			// speedY := rock.TransitionSpeedY

			// fmt.Printf("[%d + %d] [%d + %d]\n", speedX, MAX_SLOPE, speedY, MAX_SLOPE)

			// Single direct array lookup - no function calls, no SubImage!
			// Inline index calculation: speed + 4 converts (-4..+4) to (0..8)
			sprite := r.sprites[rockType][rock.SpriteSlopeX][rock.SpriteSlopeY]

			// Draw directly
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(rock.Position.X, rock.Position.Y)
			opts.Filter = ebiten.FilterLinear
			screen.DrawImage(sprite, opts)
		}
	}
}

// GetStats returns rendering statistics
func (r *RocksRenderer) GetStats() (visible, total int) {
	return r.totalRocks, r.totalRocks
}

// Utility functions
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
