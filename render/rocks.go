package render

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite sheet system
const (
	SHEET_COLUMNS  = 30                         // Columns in sprite sheet (width)
	SHEET_ROWS     = 12                         // Rows in sprite sheet (height)
	TOTAL_SPRITES  = SHEET_COLUMNS * SHEET_ROWS // Total frames in sprite sheet (360)
	NUM_ROCK_TYPES = 2                          // 3 rock types: X-axis, Y-axis, Z-axis rotation
)

// type DriftDirction uint8

// const (

// )

// SimpleRock represents a rock using sprite sheet animation
type SimpleRock struct {
	Position    Vec2           // 2D screen position
	SpriteIndex uint16         // Index into sprite sheet (0-359, max 65535). uint16 reduces struct size from 32→24 bytes (25% savings)
	SpeedX      RockSpeedIndex // Index into SpeedMap defining X-axis velocity magnitude. Combined with SpeedY, creates movement "slope" or angle. High SpeedX with low SpeedY = shallow angle (mostly horizontal). Equal SpeedX and SpeedY = 45° diagonal. Remains constant throughout lifetime.
	SpeedY      RockSpeedIndex // Index into SpeedMap defining Y-axis velocity magnitude. Combined with SpeedX, creates movement "slope" or angle. High SpeedY with low SpeedX = steep angle (mostly vertical). SpeedMap values: [2.0, 3.0, 4.0, 5.0, 6.0]. Remains constant throughout lifetime.
	SignX       DirectionSign  // X-axis direction: false=left(-1.0), true=right(+1.0). Flips on X-axis wall collision.
	SignY       DirectionSign  // Y-axis direction: false=up(-1.0), true=down(+1.0). Flips on Y-axis wall collision.
}

// BounceUpLeft sets rock direction to upper-left quadrant (↖)
// SignX=false (left), SignY=false (up)
func (r *SimpleRock) BounceUpLeft() {
	r.SignX = false
	r.SignY = false
}

// BounceUpRight sets rock direction to upper-right quadrant (↗)
// SignX=true (right), SignY=false (up)
func (r *SimpleRock) BounceUpRight() {
	r.SignX = true
	r.SignY = false
}

// BounceDownLeft sets rock direction to lower-left quadrant (↙)
// SignX=false (left), SignY=true (down)
func (r *SimpleRock) BounceDownLeft() {
	r.SignX = false
	r.SignY = true
}

// BounceDownRight sets rock direction to lower-right quadrant (↘)
// SignX=true (right), SignY=true (down)
func (r *SimpleRock) BounceDownRight() {
	r.SignX = true
	r.SignY = true
}

// BounceX flips horizontal direction (bounce off vertical wall)
func (r *SimpleRock) BounceX() {
	r.SignX = !r.SignX
}

// BounceY flips vertical direction (bounce off horizontal wall)
func (r *SimpleRock) BounceY() {
	r.SignY = !r.SignY
}

func (r *SimpleRock) Bounce() {
	// XOR multiple state values for pseudo-random but deterministic result
	seed := int(r.SpriteIndex) ^ int(r.Position.X) ^ int(r.Position.Y) ^ int(r.SpeedX) ^ int(r.SpeedY)

	if seed%2 == 0 {
		r.SignX = !r.SignX // Flip horizontal (left↔right)
	} else {
		r.SignY = !r.SignY // Flip vertical (up↔down)
	}
}

func (r *SimpleRock) BounceBasedOnAngle() {
	// XOR speed indices to create deterministic per-angle behavior
	angleHash := uint8(r.SpeedX) ^ uint8(r.SpeedY)

	if r.SpeedX >= r.SpeedY {
		// Horizontal-dominant or equal: flip horizontal
		r.SignX = !r.SignX
		// 50% chance to also flip vertical (corner bounces)
		if angleHash%2 == 1 {
			r.SignY = !r.SignY
		}
	} else {
		// Vertical-dominant: flip vertical
		r.SignY = !r.SignY
		// 50% chance to also flip horizontal (corner bounces)
		if angleHash%2 == 1 {
			r.SignX = !r.SignX
		}
	}
}

// BounceTowardsAngle sets rock direction and speed based on target angle (0-360 degrees)
// Angle system: 0°=up, 90°=right, 180°=down, 270°=left (standard unit circle)
// Determines quadrant, sets direction signs, and assigns SpeedX/SpeedY for the slope
//
// Quadrants:
//
//	0-90°:    UpRight (↗)   - SignX=true,  SignY=false
//	90-180°:  DownRight (↘) - SignX=true,  SignY=true
//	180-270°: DownLeft (↙)  - SignX=false, SignY=true
//	270-360°: UpLeft (↖)    - SignX=false, SignY=false
func (r *SimpleRock) BounceTowardsAngle(angle int) {
	// Normalize angle to 0-360 range
	angle = angle % 360
	if angle < 0 {
		angle += 360
	}

	// Determine quadrant and set direction signs
	switch {
	case angle >= 0 && angle < 90:
		// Quadrant 1: UpRight (0-89°)
		r.SignX = true  // Moving right
		r.SignY = false // Moving up
	case angle >= 90 && angle < 180:
		// Quadrant 2: DownRight (90-179°)
		r.SignX = true // Moving right
		r.SignY = true // Moving down
	case angle >= 180 && angle < 270:
		// Quadrant 3: DownLeft (180-269°)
		r.SignX = false // Moving left
		r.SignY = true  // Moving down
	case angle >= 270 && angle < 360:
		// Quadrant 4: UpLeft (270-359°)
		r.SignX = false // Moving left
		r.SignY = false // Moving up
	}

	// Calculate angle within quadrant (0-90°)
	quadrantAngle := angle % 90

	// Bucket angles into speed zones (0-18°, 18-36°, 36-54°, 54-72°, 72-90°)
	speedZone := quadrantAngle / 18

	if quadrantAngle < 45 {
		// 0-44°: First axis dominant (vertical for Q1/Q4, horizontal for Q2/Q3)
		// For Q1 (0-89°): Y dominant at start, X increases
		// For Q2 (90-179°): X dominant at start, Y increases
		if angle < 90 || angle >= 270 {
			// Q1 or Q4: Y-axis dominant transitioning to equal
			r.SpeedY = RockSpeedIndex(4 - speedZone/2) // 4, 4, 3, 3, 2
			r.SpeedX = RockSpeedIndex(speedZone)       // 0, 1, 2, 3, 4
		} else {
			// Q2 or Q3: X-axis dominant transitioning to equal
			r.SpeedX = RockSpeedIndex(4 - speedZone/2) // 4, 4, 3, 3, 2
			r.SpeedY = RockSpeedIndex(speedZone)       // 0, 1, 2, 3, 4
		}
	} else {
		// 45-90°: Second axis dominant
		if angle < 90 || angle >= 270 {
			// Q1 or Q4: X-axis becomes dominant
			r.SpeedX = RockSpeedIndex(4 - (4-speedZone)/2) // 2, 3, 3, 4, 4
			r.SpeedY = RockSpeedIndex(4 - speedZone)       // 4, 3, 2, 1, 0
		} else {
			// Q2 or Q3: Y-axis becomes dominant
			r.SpeedY = RockSpeedIndex(4 - (4-speedZone)/2) // 2, 3, 3, 4, 4
			r.SpeedX = RockSpeedIndex(4 - speedZone)       // 4, 3, 2, 1, 0
		}
	}
}

// RocksRenderer manages sprite-sheet-based 3D SDF rock rendering
type RocksRenderer struct {
	shader         *ebiten.Shader
	spriteSheets   [NUM_ROCK_TYPES]*ebiten.Image // 3 sprite sheets for X, Y, Z axis rotations
	sheetConfig    SpriteSheet                   // Sprite sheet configuration (same for all)
	Rocks          [NUM_ROCK_TYPES][]*SimpleRock // Rocks organized by type (no flag needed!)
	SpriteSize     int
	FSpriteSize    float64
	totalRocks     int
	ActiveRockType int
}

// RocksConfig holds configuration for sprite-sheet-based rock system
type RocksConfig struct {
	TotalRocks    int
	SpriteSize    int     // TILE_SIZE / 2
	WorldBoundsX  float32 // GAME_BOUNDS_X
	WorldBoundsY  float32 // GAME_BOUNDS_Y
	MinRockSize   float32
	MaxRockSize   float32
	MovementSpeed float32
}

// NewRocksRenderer creates a new sprite-cached rocks rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()
	r := &RocksRenderer{
		shader:      shaderMap[shaders.RocksShaderKey],
		SpriteSize:  config.SpriteSize,
		FSpriteSize: float64(config.SpriteSize),
		totalRocks:  config.TotalRocks,
	}

	// Generate sprite sheet
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	return r
}

// generateSprites creates 3 sprite sheets for X, Y, Z axis rotations
func (r *RocksRenderer) generateSprites() {
	// Calculate sprite sheet dimensions
	// Layout: SHEET_COLUMNS columns × SHEET_ROWS rows
	sheetWidth := r.SpriteSize * SHEET_COLUMNS
	sheetHeight := r.SpriteSize * SHEET_ROWS

	// Initialize sprite sheet configuration (same for all 3 sheets)
	r.sheetConfig = NewSpriteSheet(SHEET_COLUMNS, SHEET_ROWS, r.SpriteSize)

	// Generate 3 sprite sheets: one for each rock type with different colors
	for rockIndex := 0; rockIndex < NUM_ROCK_TYPES; rockIndex++ {
		// Create sprite sheet for this rock type
		r.spriteSheets[rockIndex] = ebiten.NewImage(sheetWidth, sheetHeight)

		// Define rock colors based on rock type
		var innerDark, innerLight, outerDark, outerLight Vec3
		switch rockIndex {
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
		case 2: // Blue-gray
			innerDark = KageColor(30, 40, 50)
			innerLight = KageColor(60, 75, 90)
			outerDark = KageColor(50, 65, 80)
			outerLight = KageColor(80, 95, 110)
		}

		// Generate each rotation frame and draw into the sprite sheet
		for row := 0; row < SHEET_ROWS; row++ {
			for col := 0; col < SHEET_COLUMNS; col++ {
				// Calculate sprite index (0-359)
				spriteIndex := row*SHEET_COLUMNS + col

				// Calculate position in sprite sheet
				x := col * r.SpriteSize
				y := row * r.SpriteSize

				// Create temporary sprite for this variant
				tempSprite := ebiten.NewImage(r.SpriteSize, r.SpriteSize)

				// Calculate rotation angle for seamless looping
				// We want 360 sprites covering 0 to 359 degrees (not including 360, which equals 0)
				// This ensures sprite[0] and sprite[359+1] are the same (perfect loop)
				rotationAngle := float32(spriteIndex) * (2.0 * 3.14159265 / float32(TOTAL_SPRITES))

				// Each rock type rotates around a different axis
				var rotX, rotY float32 //, rotZ float32
				switch rockIndex {
				case 0: // X-axis rotation
					rotX = rotationAngle
					rotY = 0
					// rotZ = 0
				case 1: // Y-axis rotation
					rotX = 0
					rotY = rotationAngle
					// rotZ = 0
				case 2: // Z-axis rotation
					rotX = rotationAngle
					rotY = rotationAngle
					// rotZ = rotationAngle
				}

				u := map[string]interface{}{
					"Time":            0.0, // No time-based animation for sprite generation
					"Resolution":      []float32{float32(r.SpriteSize), float32(r.SpriteSize)},
					"Mouse":           Vec2{X: 0.0, Y: 0.0}.KageVec2(),
					"RotationX":       rotX,
					"RotationY":       rotY,
					"InnerColorDark":  innerDark.KageVec3(),
					"InnerColorLight": innerLight.KageVec3(),
					"OuterColorDark":  outerDark.KageVec3(),
					"OuterColorLight": outerLight.KageVec3(),
				}

				// Render rock variant with shader into temp sprite
				opts := &ebiten.DrawRectShaderOptions{Uniforms: u}
				tempSprite.DrawRectShader(r.SpriteSize, r.SpriteSize, r.shader, opts)

				// Draw temp sprite into sprite sheet at calculated position
				drawOpts := &ebiten.DrawImageOptions{}
				drawOpts.GeoM.Translate(float64(x), float64(y))
				r.spriteSheets[rockIndex].DrawImage(tempSprite, drawOpts)
			}
		}
	}
}

// generateRocks creates the initial rock instances, distributed across rock types
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	// Distribute rocks evenly across the 3 types
	rocksPerType := config.TotalRocks / NUM_ROCK_TYPES
	remainder := config.TotalRocks % NUM_ROCK_TYPES

	for rockIndex := 0; rockIndex < NUM_ROCK_TYPES; rockIndex++ {
		// Add remainder to first slices
		numRocks := rocksPerType
		if rockIndex < remainder {
			numRocks++
		}

		// Allocate slice for this rock type
		r.Rocks[rockIndex] = make([]*SimpleRock, numRocks)

		for i := 0; i < numRocks; i++ {
			// Random position
			pos := Vec2{
				X: float64(rand.Float32() * config.WorldBoundsX),
				Y: float64(rand.Float32() * config.WorldBoundsY),
			}

			// Pick random sprite index (0-359)
			spriteIndex := uint16(rand.Intn(TOTAL_SPRITES))

			r.Rocks[rockIndex][i] = &SimpleRock{
				Position:    pos,
				SpriteIndex: spriteIndex,
				SpeedX:      RockSpeedIndex(rand.Intn(len(SpeedMap))),
				SpeedY:      RockSpeedIndex(rand.Intn(len(SpeedMap))),
				SignX:       DirectionSign(rand.Intn(2) == 1), // Random bool: false=left, true=right
				SignY:       DirectionSign(rand.Intn(2) == 1), // Random bool: false=up, true=down
			}
		}
	}
}

// Update updates rock positions with constant drift and animation
// func (r *RocksRenderer) Update() {
// const driftSpeed = 0.1 // Static drift speed for all rocks

// Update all rock types

// for rockIndex := range NUM_ROCK_TYPES {
// 	for i := range r.rocks[rockIndex] {
// 		rock := &r.rocks[rockIndex][i]

// 		rock.SpriteIndex = rock.SpriteIndex + 1
// 		if rock.SpriteIndex >= TOTAL_SPRITES {
// 			rock.SpriteIndex = 0
// 		}

// 		// // Apply constant drift to position
// 		// rock.Position.X += driftSpeed
// 		// rock.Position.Y += driftSpeed * 0.5

// 		// Wrap around screen boundaries
// 		if rock.Position.X < 0 {
// 			rock.Position.X += GAME_BOUNDS_X
// 		} else if rock.Position.X > GAME_BOUNDS_X {
// 			rock.Position.X -= GAME_BOUNDS_X
// 		}

// 		if rock.Position.Y < 0 {
// 			rock.Position.Y += GAME_BOUNDS_Y
// 		} else if rock.Position.Y > GAME_BOUNDS_Y {
// 			rock.Position.Y -= GAME_BOUNDS_Y
// 		}
// 	}
// }
// }

// Draw renders all rocks using sprite sheet subrects
func (r *RocksRenderer) DEBUGDrawSheets(screen *ebiten.Image) {
	// DEBUG: Draw sprite sheets on the right side of screen to visualize them
	sheetWidth := float64(SHEET_COLUMNS * r.SpriteSize)
	xPos := GAME_BOUNDS_X - sheetWidth

	for rockIndex := 0; rockIndex < NUM_ROCK_TYPES; rockIndex++ {
		spriteSheet := r.spriteSheets[rockIndex]
		yPos := float64(rockIndex) * float64(r.SpriteSize*SHEET_ROWS)

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(xPos, yPos)
		screen.DrawImage(spriteSheet, opts)
	}
}

func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	// Draw each rock type using its corresponding sprite sheet
	for rockIndex := range NUM_ROCK_TYPES {
		spriteSheet := r.spriteSheets[rockIndex]

		// draws it in order
		for i := range r.Rocks[rockIndex] {
			rock := r.Rocks[rockIndex][i]

			// Get the subrect from sprite sheet using sprite index (cast uint16→int)
			subRect := r.sheetConfig.Rect(int(rock.SpriteIndex))

			// Extract the subimage from the sprite sheet
			sprite := spriteSheet.SubImage(subRect).(*ebiten.Image)

			// Draw at position
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(rock.Position.X, rock.Position.Y)
			opts.Filter = ebiten.FilterLinear

			screen.DrawImage(sprite, opts)
		}
	}
}

// var rocksSizes = [NUM_ROCK_TYPES]float64{
// 	float64(TILE_SIZE / 2),
// 	float64(TILE_SIZE / 2),
// 	float64(TILE_SIZE / 2),
// }

// func (r *RocksRenderer) DetermineSpriteSizes() [NUM_ROCK_TYPES]float64 {
// 	types := make()
// }

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
