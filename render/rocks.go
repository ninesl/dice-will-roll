package render

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite system
const (
	NUM_ROCK_TYPES    = 2                       // 2 rock types: different shapes
	DEGREES_PER_FRAME = 45                      // Degrees of rotation per transition frame
	ROTATION_FRAMES   = 360 / DEGREES_PER_FRAME // 360/DEGREES_PER_FRAME = X frames static spin

	DIRECTIONS_TO_SNAP = MAX_SLOPE * 2 // # of possible angles for SpriteSlopeX and SpriteSlopeY. wraps

	MAX_SLOPE int8 = 4
	MIN_SLOPE int8 = -MAX_SLOPE
	// SPEED_RANGE      = MAX_SLOPE*2 + 1 // 9 (from -4 to +4 inclusive)
)

// Constants for animation rate calculation
const baseN = 22.0
const speedFactor = 3.5

// SimpleRock represents a rock using pre-extracted sprite frames
type SimpleRock struct {
	Position      Vec2   // 2D screen position
	SpriteIndex   uint16 // Current rotation frame index (0-71)
	animationRate uint8  // Cached update frequency, FrameCounter%animationRate is when rock updates
	SlopeX        int8   // Current X speed component (-4 to +4)
	SlopeY        int8   // Current Y speed component (-4 to +4)

	// Transition system for smooth sprite rotation during direction changes
	SpriteSlopeX int8 // Visual speed X used during transition (gradually moves toward SpeedX)
	SpriteSlopeY int8 // Visual speed Y used during transition (gradually moves toward SpeedY)
}

const BaseVelocity = 1.0

// calculateAnimationRate computes the update frequency based on current slopes
func (r *SimpleRock) calculateAnimationRate() {
	speed := math.Sqrt(float64(r.SlopeX*r.SlopeX + r.SlopeY*r.SlopeY))
	n := int(math.Max(2, baseN-(speed*speedFactor)))
	r.animationRate = uint8(n) // n is guaranteed to be <= 22, fits in uint8
}

// Updates the rock based on the current target transitions.
//
// will update it's state based on other params every Tick/time this is called
func (r *SimpleRock) Update(frameCounter int) {
	r.UpdateTransition(frameCounter)

	r.Position.Y += BaseVelocity * float32(r.SlopeY)
	r.Position.X += BaseVelocity * float32(r.SlopeX)
}

// if newY or newX is IDENTICAL to the the current value in the struct
//
// does the go compiler ignore this if hte func is called? I want to reuse this in BounceX and BounceY but it seems to be
// a wasted allocation if we were to assign newX newY each time?
func (r *SimpleRock) Bounce(newX int8, newY int8) {
	r.SlopeX = newX
	r.SlopeY = newY
	r.calculateAnimationRate() // Recalculate cached animation rate
}

// BounceX flips horizontal direction (bounce off vertical wall)
func (r *SimpleRock) BounceX() {
	r.SlopeX = -r.SlopeX
	r.calculateAnimationRate() // Recalculate cached animation rate
}

// BounceY flips vertical direction (bounce off horizontal wall)
func (r *SimpleRock) BounceY() {
	r.SlopeY = -r.SlopeY
	r.calculateAnimationRate() // Recalculate cached animation rate
}

// UpdateTransition handles the smooth sprite rotation during direction changes
// Each frame, SpeedX/Y gradually move toward TransitionSpeedX/Y
func (r *SimpleRock) UpdateTransition(frameCounter int) {
	if frameCounter%int(r.animationRate) != 0 {
		// Only update every N frame for visible transitions (performance & visual)
		return
	}

	// Increment or decrement sprite index based on horizontal direction
	// Moving right (positive SlopeX): increment (rotate clockwise)
	// Moving left (negative SlopeX): decrement (rotate counter-clockwise)
	if r.SlopeX >= 0 {
		if r.SpriteIndex == 0 {
			r.SpriteIndex = ROTATION_FRAMES - 1
		} else {
			r.SpriteIndex--
		}
	} else {
		r.SpriteIndex++
		if r.SpriteIndex >= ROTATION_FRAMES {
			r.SpriteIndex = 0
		}
	}

	// Gradually move transition speeds toward target speeds (one step per frame)
	// Takes shortest path around the circular direction system (0-7 wraps)

	// Update X component
	targetX := r.SlopeX + MAX_SLOPE
	if targetX == DIRECTIONS_TO_SNAP { // Wrap 8 -> 0
		targetX = 0
	}

	// Calculate shortest angular distance
	diffX := targetX - r.SpriteSlopeX
	if diffX > DIRECTIONS_TO_SNAP/2 { // If distance > 4, go the other way
		// Going backwards is shorter (wrap around)
		r.SpriteSlopeX--
		if r.SpriteSlopeX < 0 {
			r.SpriteSlopeX = DIRECTIONS_TO_SNAP - 1 // Wrap -1 -> 7
		}
	} else if diffX < -DIRECTIONS_TO_SNAP/2 { // If distance < -4, go the other way
		// Going forwards is shorter (wrap around)
		r.SpriteSlopeX++
		if r.SpriteSlopeX == DIRECTIONS_TO_SNAP {
			r.SpriteSlopeX = 0 // Wrap 8 -> 0
		}
	} else if diffX > 0 {
		// Normal increment (target is ahead)
		r.SpriteSlopeX++
		if r.SpriteSlopeX == DIRECTIONS_TO_SNAP {
			r.SpriteSlopeX = 0 // Wrap 8 -> 0
		}
	} else if diffX < 0 {
		// Normal decrement (target is behind)
		r.SpriteSlopeX--
		if r.SpriteSlopeX < 0 {
			r.SpriteSlopeX = DIRECTIONS_TO_SNAP - 1 // Wrap -1 -> 7
		}
	}
	// If diffX == 0, already at target, do nothing

	// Update Y component (same logic)
	targetY := r.SlopeY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP { // Wrap 8 -> 0
		targetY = 0
	}

	// Calculate shortest angular distance
	diffY := targetY - r.SpriteSlopeY
	if diffY > DIRECTIONS_TO_SNAP/2 { // If distance > 4, go the other way
		// Going backwards is shorter (wrap around)
		r.SpriteSlopeY--
		if r.SpriteSlopeY < 0 {
			r.SpriteSlopeY = DIRECTIONS_TO_SNAP - 1 // Wrap -1 -> 7
		}
	} else if diffY < -DIRECTIONS_TO_SNAP/2 { // If distance < -4, go the other way
		// Going forwards is shorter (wrap around)
		r.SpriteSlopeY++
		if r.SpriteSlopeY == DIRECTIONS_TO_SNAP {
			r.SpriteSlopeY = 0 // Wrap 8 -> 0
		}
	} else if diffY > 0 {
		// Normal increment (target is ahead)
		r.SpriteSlopeY++
		if r.SpriteSlopeY == DIRECTIONS_TO_SNAP {
			r.SpriteSlopeY = 0 // Wrap 8 -> 0
		}
	} else if diffY < 0 {
		// Normal decrement (target is behind)
		r.SpriteSlopeY--
		if r.SpriteSlopeY < 0 {
			r.SpriteSlopeY = DIRECTIONS_TO_SNAP - 1 // Wrap -1 -> 7
		}
	}
	// If diffY == 0, already at target, do nothing
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

	// 3D array of sprite sheets
	// [rockType][speedX_index][speedY_index] -> Sprite struct containing spritesheet
	// Each spritesheet has ROTATION_FRAMES (72) frames arranged in 18 columns x 4 rows
	// Size: [2][8][8] = 64 spritesheets total
	sprites [NUM_ROCK_TYPES][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]Sprite

	Rocks          [NUM_ROCK_TYPES][]*SimpleRock // Rocks organized by type
	SpriteSize     int
	FSpriteSize    float32
	totalRocks     int
	ActiveRockType int
	FrameCounter   [NUM_ROCK_TYPES]int // Global frame counter for transition timing
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
		FSpriteSize: float32(config.SpriteSize),
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
	genSprites := [NUM_ROCK_TYPES][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]Sprite{}

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

		XSnap := [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]Sprite{}
		genSprites[rockType] = XSnap
		for XSnapIdx := range DIRECTIONS_TO_SNAP {
			YSnap := [DIRECTIONS_TO_SNAP]Sprite{}
			genSprites[rockType][XSnapIdx] = YSnap

			angleDegX := int(XSnapIdx) * (360 / int(DIRECTIONS_TO_SNAP))
			angleRadX := float32(angleDegX) * (math.Pi / 180.0)

			// angleX := math.Atan2(float64(y), float64(x))
			// angleDeg := angleRad * 180.0 / math.Pi
			// if angleDeg < 0 {
			// 	angleDeg += 360
			// }

			for YSnapIdx := range DIRECTIONS_TO_SNAP {

				angleDegY := int(YSnapIdx) * (360 / int(DIRECTIONS_TO_SNAP))
				angleRadY := float32(angleDegY) * (math.Pi / 180.0)

				// Create spritesheet: 18 columns x 4 rows = 72 frames
				const SHEET_COLS = 18
				const SHEET_ROWS = 4
				sheetWidth := r.SpriteSize * SHEET_COLS
				sheetHeight := r.SpriteSize * SHEET_ROWS
				spriteSheet := ebiten.NewImage(sheetWidth, sheetHeight)

				// Render all rotation frames into the spritesheet
				for frameIdx := 0; frameIdx < ROTATION_FRAMES; frameIdx++ {
					// Calculate Z rotation angle (0 to 360 degrees in DEGREES_PER_FRAME increments)
					rotationAngle := float32(frameIdx*DEGREES_PER_FRAME) * (math.Pi / 180.0)

					// Create temporary image for this frame
					frameImg := ebiten.NewImage(r.SpriteSize, r.SpriteSize)

					u := map[string]interface{}{
						"Time":            0.0,
						"Resolution":      []float32{float32(r.SpriteSize), float32(r.SpriteSize)},
						"Mouse":           Vec2{X: 0.0, Y: 0.0}.KageVec2(),
						"RotationX":       angleRadX,
						"RotationY":       angleRadY,
						"RotationZ":       rotationAngle,
						"InnerColorDark":  innerDark.KageVec3(),
						"InnerColorLight": innerLight.KageVec3(),
						"OuterColorDark":  outerDark.KageVec3(),
						"OuterColorLight": outerLight.KageVec3(),
					}

					opts := &ebiten.DrawRectShaderOptions{Uniforms: u}
					frameImg.DrawRectShader(r.SpriteSize, r.SpriteSize, r.shader, opts)

					// Calculate position in spritesheet (row-major order)
					col := frameIdx % SHEET_COLS
					row := frameIdx / SHEET_COLS

					// Draw this frame into the spritesheet at the correct position
					drawOpts := &ebiten.DrawImageOptions{}
					drawOpts.GeoM.Translate(float64(col*r.SpriteSize), float64(row*r.SpriteSize))
					spriteSheet.DrawImage(frameImg, drawOpts)
				}

				// Create Sprite struct with spritesheet metadata
				sprite := Sprite{
					Image:       spriteSheet,
					SpriteSheet: NewSpriteSheet(SHEET_COLS, SHEET_ROWS, r.SpriteSize),
					ActiveFrame: 0,
				}

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

	// Assign generated sprites to the renderer
	r.sprites = genSprites
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
				X: rand.Float32() * config.WorldBoundsX,
				Y: rand.Float32() * config.WorldBoundsY,
			}

			// Pick random rotation frame
			spriteIndex := uint16(rand.Intn(ROTATION_FRAMES))

			// Generate slope values from -4 to +4 (9 possible values)
			// rand gives 0-8, subtract MAX_SLOPE to get -4 to +4
			slopeX := int8(rand.Intn(int(DIRECTIONS_TO_SNAP)+1)) - MAX_SLOPE
			slopeY := int8(rand.Intn(int(DIRECTIONS_TO_SNAP)+1)) - MAX_SLOPE

			// Convert slopes to sprite indices (0-7), wrapping +4 back to 0
			spriteSlopeX := slopeX + MAX_SLOPE // Convert -4..+4 to 0..8
			if spriteSlopeX == DIRECTIONS_TO_SNAP {
				spriteSlopeX = 0 // Wrap 8 -> 0 (so +4 uses same sprite as -4)
			}
			spriteSlopeY := slopeY + MAX_SLOPE // Convert -4..+4 to 0..8
			if spriteSlopeY == DIRECTIONS_TO_SNAP {
				spriteSlopeY = 0 // Wrap 8 -> 0 (so +4 uses same sprite as -4)
			}

			rock := &SimpleRock{
				Position:     pos,
				SpriteIndex:  spriteIndex,
				SlopeX:       slopeX,
				SlopeY:       slopeY,
				SpriteSlopeX: spriteSlopeX,
				SpriteSlopeY: spriteSlopeY,
			}
			rock.calculateAnimationRate() // Initialize cached value
			r.Rocks[rockType][i] = rock
		}
	}
}

// DrawRocks renders all rocks with ultra-fast direct array access
func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	// Reuse DrawImageOptions to avoid allocations (important for 10k+ rocks)
	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear

	for rockType := range NUM_ROCK_TYPES {
		for _, rock := range r.Rocks[rockType] {
			// Get the sprite for this slope combination
			sprite := r.sprites[rockType][rock.SpriteSlopeX][rock.SpriteSlopeY]

			// Get the specific rotation frame from the spritesheet
			frameRect := sprite.SpriteSheet.Rect(int(rock.SpriteIndex))
			frameImage := sprite.Image.SubImage(frameRect).(*ebiten.Image)

			// Reset and set transform
			opts.GeoM.Reset()
			opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
			screen.DrawImage(frameImage, opts)
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
