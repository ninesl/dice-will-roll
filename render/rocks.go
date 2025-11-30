package render

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite system
const (
	NUM_ROCK_TYPES = 2 // 2 rock types: different shapes

	// TODO: need to benchmark this on varying hardware. higher number less sprites in sheet/memory used
	DEGREES_PER_FRAME = 20                      // Degrees of rotation per transition frame
	ROTATION_FRAMES   = 360 / DEGREES_PER_FRAME // 360/DEGREES_PER_FRAME = X frames static spin

	DIRECTIONS_TO_SNAP = MAX_SLOPE * 2 // # of possible angles for SpriteSlopeX and SpriteSlopeY. wraps

	MAX_SLOPE int8 = 4
	MIN_SLOPE int8 = -MAX_SLOPE
	// SPEED_RANGE      = MAX_SLOPE*2 + 1 // 9 (from -4 to +4 inclusive)
)

// Spritesheet layout variables - dynamically calculated from ROTATION_FRAMES
var (
	SHEET_COLS = calculateSheetCols(ROTATION_FRAMES)
	SHEET_ROWS = (ROTATION_FRAMES + SHEET_COLS - 1) / SHEET_COLS
)

// calculateSheetCols determines optimal column count for balanced spritesheet layout
func calculateSheetCols(frames int) int {
	// Aim for roughly square layout, slightly wider than tall
	sqrt := int(math.Sqrt(float64(frames)))

	// Round up to nearest even number for cleaner layout
	if sqrt%2 != 0 {
		sqrt++
	}

	// Ensure we don't make it too tall or too wide
	if sqrt < 3 {
		sqrt = 3
	}

	return sqrt
}

// Constants for animation rate calculation
const baseN = 22.0
const speedFactor = 3.5

// Sprite rotation rates indexed by RockScoreType value
// LOWER value = MORE frequent updates = FASTER rotation
// Small=1, Medium=3, Big=5, Huge=10
var rockRotationRate = [Huge + 1]uint8{
	0,  // 0: unused (default)
	2,  // 1: Small - updates every 2 frames (VERY FAST)
	0,  // 2: unused
	5,  // 3: Medium - updates every 5 frames (fast)
	0,  // 4: unused
	8,  // 5: Big - updates every 10 frames (normal)
	0,  // 6: unused
	0,  // 7: unused
	0,  // 8: unused
	0,  // 9: unused
	10, // 10: Huge - updates every 20 frames (SLOW)
}

type RockScoreType uint8

const (
	Small  RockScoreType = 1
	Medium RockScoreType = 3
	Big    RockScoreType = 5
	Huge   RockScoreType = 10
)

// SizeMultiplier returns the size multiplier for this RockScoreType
func (rst RockScoreType) SizeMultiplier() float32 {
	switch rst {
	case Small:
		return .4
	case Medium:
		return .75
	case Big:
		return .9
	case Huge:
		return 1.2
	default:
		return 1.0
	}
}

// SimpleRock represents a rock using pre-extracted sprite frames
type SimpleRock struct {
	Position      Vec2   // 2D screen position
	SpriteIndex   uint16 // Current rotation frame index (0-71)
	animationRate uint8  // Cached update frequency, FrameCounter%animationRate is when rock updates
	SlopeX        int8   // Current X speed component (-4 to +4)
	SlopeY        int8   // Current Y speed component (-4 to +4)

	// Transition system for smooth sprite rotation during direction changes
	SpriteSlopeX int8          // Visual speed X used during transition (gradually moves toward SpeedX)
	SpriteSlopeY int8          // Visual speed Y used during transition (gradually moves toward SpeedY)
	Score        RockScoreType //  how many 'rocks' this rock counts for during scoring. also determines size, etc

	transitionSteps uint8 // Bit-packed: lower 4 bits = X remaining steps, upper 4 bits = Y remaining steps

}

// const BaseVelocity = 1.0

const BaseVelocity = 2.0

func (r *SimpleRock) RockWithinDie(die DieRenderable, rockSize float32) bool {
	// AABB (Axis-Aligned Bounding Box) collision detection with 0.75 multiplier for tighter collision
	// Checks if rock's bounding box overlaps with die's bounding box
	effectiveTileSize := TileSize * 0.75
	effectiveRockSize := rockSize * 0.75

	// Center the effective collision boxes
	dieInset := (TileSize - effectiveTileSize) / 2
	rockInset := (rockSize - effectiveRockSize) / 2

	return (r.Position.X+rockInset+effectiveRockSize > die.Vec2.X+dieInset && r.Position.X+rockInset < die.Vec2.X+dieInset+effectiveTileSize) &&
		(r.Position.Y+rockInset+effectiveRockSize > die.Vec2.Y+dieInset && r.Position.Y+rockInset < die.Vec2.Y+dieInset+effectiveTileSize)
}

// TODO: determine if copy is faster than reference, and baseSpriteSize copy by ref or get it from g *Game
func (r *SimpleRock) XYWithinRock(X, Y float32, spriteSize float32) bool {
	return (Y >= r.Position.Y && Y <= r.Position.Y+spriteSize) && (X >= r.Position.X && X <= r.Position.X+spriteSize)
}

// getTransitionStepsX returns the remaining transition steps for X axis
func (r *SimpleRock) getTransitionStepsX() uint8 {
	return r.transitionSteps & 0x0F
}

// getTransitionStepsY returns the remaining transition steps for Y axis
func (r *SimpleRock) getTransitionStepsY() uint8 {
	return (r.transitionSteps >> 4) & 0x0F
}

// setTransitionSteps sets both X and Y transition steps (each must be 0-15)
func (r *SimpleRock) setTransitionSteps(stepsX, stepsY uint8) {
	r.transitionSteps = (stepsY << 4) | (stepsX & 0x0F)
}

// decrementTransitionStepX decrements X transition counter if > 0
func (r *SimpleRock) decrementTransitionStepX() {
	if r.getTransitionStepsX() > 0 {
		r.transitionSteps--
	}
}

// decrementTransitionStepY decrements Y transition counter if > 0
func (r *SimpleRock) decrementTransitionStepY() {
	if r.getTransitionStepsY() > 0 {
		r.transitionSteps -= 0x10 // Subtract from upper 4 bits
	}
}

// TODO:FIXME: maybe this should be another lookup table? to reduce arithmetic ops? need to benchmark
// GetSize returns the pixel size of this rock based on its RockScoreType
func (r *SimpleRock) GetSize(baseSpriteSize int) float32 {
	return float32(baseSpriteSize) * r.Score.SizeMultiplier()
}

// calculateAnimationRate computes the update frequency based on current slopes
func (r *SimpleRock) calculateAnimationRate() {
	speed := math.Sqrt(float64(r.SlopeX*r.SlopeX + r.SlopeY*r.SlopeY))
	n := int(math.Max(2, baseN-(speed*speedFactor)))
	r.animationRate = uint8(n) // n is guaranteed to be <= 22, fits in uint8
}

// calculateShortestPath returns the shortest distance and direction (+1 or -1) to reach target
func calculateShortestPath(current, target int8) (distance uint8, direction int8) {
	diff := target - current

	if diff > DIRECTIONS_TO_SNAP/2 {
		// Going backwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP - diff)
		direction = -1
	} else if diff < -DIRECTIONS_TO_SNAP/2 {
		// Going forwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP + diff)
		direction = 1
	} else if diff > 0 {
		// Normal forward
		distance = uint8(diff)
		direction = 1
	} else if diff < 0 {
		// Normal backward
		distance = uint8(-diff)
		direction = -1
	} else {
		// Already at target
		distance = 0
		direction = 0
	}

	return distance, direction
}

// Updates the rock based on the current target transitions.
//
// will update it's state based on other params every Tick/time this is called
func (r *SimpleRock) Update(frameCounter int) {
	r.Position.Y += BaseVelocity * float32(r.SlopeY)
	r.Position.X += BaseVelocity * float32(r.SlopeX)

	r.UpdateTransition(frameCounter)
}

// if newY or newX is IDENTICAL to the the current value in the struct
//
// does the go compiler ignore this if hte func is called? I want to reuse this in BounceX and BounceY but it seems to be
// a wasted allocation if we were to assign newX newY each time?
func (r *SimpleRock) Bounce(newX int8, newY int8) {
	r.SlopeX = newX
	r.SlopeY = newY

	// Calculate transition steps for X
	targetX := newX + MAX_SLOPE
	if targetX == DIRECTIONS_TO_SNAP {
		targetX = 0
	}
	distX, _ := calculateShortestPath(r.SpriteSlopeX, targetX)

	// Calculate transition steps for Y
	targetY := newY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP {
		targetY = 0
	}
	distY, _ := calculateShortestPath(r.SpriteSlopeY, targetY)

	// Add full rotation (DIRECTIONS_TO_SNAP) to shortest path
	r.setTransitionSteps(distX+uint8(DIRECTIONS_TO_SNAP), distY+uint8(DIRECTIONS_TO_SNAP))

	r.calculateAnimationRate() // Recalculate cached animation rate
}

// BounceX flips horizontal direction (bounce off vertical wall)
// Rock spins around Y-axis (sideways tumble) based on Y speed
func (r *SimpleRock) BounceX() {
	r.SlopeX = -r.SlopeX

	// X needs to transition to new target (shortest path, no extra rotation)
	targetX := r.SlopeX + MAX_SLOPE
	if targetX == DIRECTIONS_TO_SNAP {
		targetX = 0
	}
	distX, _ := calculateShortestPath(r.SpriteSlopeX, targetX)

	// When bouncing off vertical wall, tumble around Y-axis based on Y speed
	// Y slope doesn't change, so just tumble
	absY := r.SlopeY
	if absY < 0 {
		absY = -absY
	}
	tumbleY := uint8(absY) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(distX, tumbleY)

	r.calculateAnimationRate() // Recalculate cached animation rate
}

// BounceY flips vertical direction (bounce off horizontal wall)
// Rock spins around X-axis (forward/backward tumble) based on X speed
func (r *SimpleRock) BounceY() {
	r.SlopeY = -r.SlopeY

	// Y needs to transition to new target (shortest path, no extra rotation)
	targetY := r.SlopeY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP {
		targetY = 0
	}
	distY, _ := calculateShortestPath(r.SpriteSlopeY, targetY)

	// When bouncing off horizontal wall, tumble around X-axis based on X speed
	// X slope doesn't change, so just tumble
	absX := r.SlopeX
	if absX < 0 {
		absX = -absX
	}
	tumbleX := uint8(absX) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(tumbleX, distY)

	r.calculateAnimationRate() // Recalculate cached animation rate
}

// UpdateTransition handles the smooth sprite rotation during direction changes
// Now includes full rotation on each bounce
func (r *SimpleRock) UpdateTransition(frameCounter int) {
	// Update SpriteIndex based on rock SIZE (smaller rocks rotate faster)
	if frameCounter%int(rockRotationRate[r.Score]) == 0 {
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
	}

	// Update SpriteSlopeX/Y based on SPEED (for smooth transitions during bounces)
	if frameCounter%int(r.animationRate) != 0 {
		return
	}

	// Update X component with transition counter
	if r.getTransitionStepsX() > 0 {
		// Always increment for smooth continuous rotation
		r.SpriteSlopeX++
		if r.SpriteSlopeX >= DIRECTIONS_TO_SNAP {
			r.SpriteSlopeX = 0
		}
		r.decrementTransitionStepX()
	}

	// Update Y component with transition counter
	if r.getTransitionStepsY() > 0 {
		// Always increment for smooth continuous rotation
		r.SpriteSlopeY++
		if r.SpriteSlopeY >= DIRECTIONS_TO_SNAP {
			r.SpriteSlopeY = 0
		}
		r.decrementTransitionStepY()
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

	// 3D array of sprite sheets
	// [rockType][speedX_index][speedY_index] -> Sprite struct containing spritesheet
	// Each spritesheet has ROTATION_FRAMES (72) frames arranged in 18 columns x 4 rows
	// Size: [2][8][8] = 64 spritesheets total
	sprites [NUM_ROCK_TYPES][DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]Sprite

	Rocks          [NUM_ROCK_TYPES][]*SimpleRock // Rocks organized by type
	SpriteSize     int
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
		shader:     shaderMap[shaders.RocksShaderKey],
		SpriteSize: config.SpriteSize,
		totalRocks: config.TotalRocks,
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

				// Create spritesheet based on ROTATION_FRAMES
				sheetWidth := r.SpriteSize * SHEET_COLS
				sheetHeight := r.SpriteSize * SHEET_ROWS
				spriteSheet := ebiten.NewImage(sheetWidth, sheetHeight)

				// Render all rotation frames into the spritesheet
				for frameIdx := 0; frameIdx < ROTATION_FRAMES; frameIdx++ {
					// Calculate Z rotation angle (0 to 360 degrees in DEGREES_PER_FRAME increments)
					rotationRadAngle := float32(frameIdx*DEGREES_PER_FRAME) * (math.Pi / 180.0)

					// Create temporary image for this frame
					frameImg := ebiten.NewImage(r.SpriteSize, r.SpriteSize)

					u := map[string]interface{}{
						"Time":            0.0,
						"Resolution":      []float32{float32(r.SpriteSize), float32(r.SpriteSize)},
						"Mouse":           Vec2{X: 0.0, Y: 0.0}.KageVec2(),
						"RotationX":       angleRadX,
						"RotationY":       angleRadY,
						"RotationZ":       rotationRadAngle,
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

// generateRocks creates rock instances with random RockScoreTypes that accumulate to target score
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	targetScore := config.TotalRocks // e.g., 500
	currentScore := 0

	// Track rocks by type
	var allRocks [][]*SimpleRock
	for i := 0; i < NUM_ROCK_TYPES; i++ {
		allRocks = append(allRocks, []*SimpleRock{})
	}

	// Generate rocks until we reach target score
	for currentScore < targetScore {
		remaining := targetScore - currentScore

		// Pick a random RockScoreType that doesn't exceed remaining
		var scoreType RockScoreType
		switch {
		case remaining >= 10 && rand.Float32() < 0.15: // 15% chance for Huge
			scoreType = Huge
		case remaining >= 5 && rand.Float32() < 0.25: // 25% chance for Big
			scoreType = Big
		case remaining >= 3 && rand.Float32() < 0.35: // 35% chance for Medium
			scoreType = Medium
		default: // Otherwise Small
			scoreType = Small
		}

		// Pick random rock type (0 or 1)
		rockType := rand.Intn(NUM_ROCK_TYPES)

		// Random position
		pos := Vec2{
			X: rand.Float32() * config.WorldBoundsX,
			Y: rand.Float32() * config.WorldBoundsY,
		}

		// Pick random rotation frame
		spriteIndex := uint16(rand.Intn(ROTATION_FRAMES))

		// Generate slope values
		slopeX := int8(rand.Intn(int(DIRECTIONS_TO_SNAP)+1)) - MAX_SLOPE
		slopeY := int8(rand.Intn(int(DIRECTIONS_TO_SNAP)+1)) - MAX_SLOPE

		// Convert slopes to sprite indices
		spriteSlopeX := slopeX + MAX_SLOPE
		if spriteSlopeX == DIRECTIONS_TO_SNAP {
			spriteSlopeX = 0
		}
		spriteSlopeY := slopeY + MAX_SLOPE
		if spriteSlopeY == DIRECTIONS_TO_SNAP {
			spriteSlopeY = 0
		}

		rock := &SimpleRock{
			Position:     pos,
			SpriteIndex:  spriteIndex,
			SlopeX:       slopeX,
			SlopeY:       slopeY,
			SpriteSlopeX: spriteSlopeX,
			SpriteSlopeY: spriteSlopeY,
			Score:        scoreType,
		}
		rock.calculateAnimationRate()

		allRocks[rockType] = append(allRocks[rockType], rock)
		currentScore += int(scoreType)
	}

	// Assign to renderer
	for i := 0; i < NUM_ROCK_TYPES; i++ {
		r.Rocks[i] = allRocks[i]
	}

	// Update total count
	r.totalRocks = 0
	for i := 0; i < NUM_ROCK_TYPES; i++ {
		r.totalRocks += len(r.Rocks[i])
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

			// Reset and set transform with SCALING based on RockScoreType
			opts.GeoM.Reset()
			scale := rock.Score.SizeMultiplier()
			opts.GeoM.Scale(float64(scale), float64(scale))
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
