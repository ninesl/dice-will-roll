package render

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite system
const (
	NUM_ROCK_TYPES = 2 // 2 rock types: different colors

	// TODO: need to benchmark this on varying hardware. higher number less sprites in sheet/memory used
	DEGREES_PER_FRAME = 20                      // Degrees of rotation per transition frame
	ROTATION_FRAMES   = 360 / DEGREES_PER_FRAME // 360/DEGREES_PER_FRAME = X frames static spin

	DIRECTIONS_TO_SNAP = MAX_SLOPE * 2 // # of possible angles for SpriteSlopeX and SpriteSlopeY. wraps

	MAX_SLOPE int8 = 4
	MIN_SLOPE int8 = -MAX_SLOPE
	// SPEED_RANGE      = MAX_SLOPE*2 + 1 // 9 (from -4 to +4 inclusive)

	// Interleaving constant: controls how finely rocks of different colors are mixed
	// Higher value = finer mixing, more draw calls but better visual interleaving
	// Lower value = coarser mixing, fewer draw calls but more color clumping
	NUM_INTERLEAVE_LAYERS = 3
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

const (
	// Score value constants - the actual score value of the rock
	//TODO:FIXME: hsould this not be in render/?
	SmallScore  = 1
	MediumScore = 3
	BigScore    = 5
	HugeScore   = 10

	// for animation rate calculation
	baseN       = 22
	speedFactor = 3
)

// Size multiplier constants for rock variants
const (
	// Small rock size multipliers
	SmallLargeSize  = 0.25
	SmallMediumSize = 0.22
	SmallTinySize   = 0.20
	SmallMinSize    = SmallTinySize // For collision detection

	// Medium rock size multipliers
	MediumLargeSize  = 0.50
	MediumMediumSize = 0.45
	MediumSmallSize  = 0.40
	MediumMinSize    = MediumSmallSize // For collision detection

	// Big rock size multipliers
	BigLargeSize  = 0.80
	BigMediumSize = 0.75
	BigSmallSize  = 0.70
	BigMinSize    = BigSmallSize // For collision detection

	// Huge rock size multipliers
	HugeLargeSize  = 1.20
	HugeMediumSize = 1.10
	HugeSmallSize  = 1.00
	HugeMinSize    = HugeSmallSize // For collision detection
)

// The underlying Score that the rock counts for. Also track rock size/size multiplier and animation rate
type RockScoreType uint8

const (
	// Small rocks (enum 1-3) - all worth SmallScore points
	SmallLarge  RockScoreType = 1 // 0.25× size
	SmallMedium RockScoreType = 2 // 0.22× size
	SmallTiny   RockScoreType = 3 // 0.20× size

	// Medium rocks (enum 4-6) - all worth MediumScore points
	MediumLarge  RockScoreType = 4 // 0.50× size
	MediumMedium RockScoreType = 5 // 0.45× size
	MediumSmall  RockScoreType = 6 // 0.40× size

	// Big rocks (enum 7-9) - all worth BigScore points
	BigLarge  RockScoreType = 7 // 0.80× size
	BigMedium RockScoreType = 8 // 0.75× size
	BigSmall  RockScoreType = 9 // 0.70× size

	// Huge rocks (enum 10-12) - all worth HugeScore points
	HugeLarge  RockScoreType = 10 // 1.20× size
	HugeMedium RockScoreType = 11 // 1.10× size
	HugeSmall  RockScoreType = 12 // 1.00× size

	MaxRockType RockScoreType = 13
)

// GetScore returns the point value for scoring (groups variants together)
func (rst RockScoreType) GetScore() int {
	switch rst {
	case SmallLarge, SmallMedium, SmallTiny:
		return SmallScore // 1
	case MediumLarge, MediumMedium, MediumSmall:
		return MediumScore // 3
	case BigLarge, BigMedium, BigSmall:
		return BigScore // 5
	case HugeLarge, HugeMedium, HugeSmall:
		return HugeScore // 10
	default:
		return SmallScore
	}
}

// SizeMultiplier returns the size multiplier for this RockScoreType variant
func (rst RockScoreType) SizeMultiplier() float32 {
	switch rst {
	// Small variants
	case SmallLarge:
		return SmallLargeSize
	case SmallMedium:
		return SmallMediumSize
	case SmallTiny:
		return SmallTinySize

	// Medium variants
	case MediumLarge:
		return MediumLargeSize
	case MediumMedium:
		return MediumMediumSize
	case MediumSmall:
		return MediumSmallSize

	// Big variants
	case BigLarge:
		return BigLargeSize
	case BigMedium:
		return BigMediumSize
	case BigSmall:
		return BigSmallSize

	// Huge variants
	case HugeLarge:
		return HugeLargeSize
	case HugeMedium:
		return HugeMediumSize
	case HugeSmall:
		return HugeSmallSize

	default:
		return 1.0
	}
}

// GetMinSizeForCategory returns the smallest size variant for collision detection
// This allows using a single hitbox size per category for simplified collision
func (rst RockScoreType) GetMinSizeForCategory() float32 {
	switch rst {
	case SmallLarge, SmallMedium, SmallTiny:
		return SmallMinSize
	case MediumLarge, MediumMedium, MediumSmall:
		return MediumMinSize
	case BigLarge, BigMedium, BigSmall:
		return BigMinSize
	case HugeLarge, HugeMedium, HugeSmall:
		return HugeMinSize
	default:
		return 1.0
	}
}

// SimpleRock represents a rock using pre-extracted sprite frames
type SimpleRock struct {
	Position    Vec2  // 2D screen position
	SpriteIndex uint8 // Current rotation frame index (0-71)
	SlopeX      int8  // Current X speed component (-4 to +4)
	SlopeY      int8  // Current Y speed component (-4 to +4)

	// Transition system for smooth sprite rotation during direction changes
	SpriteSlopeX int8          // Visual speed X used during transition (gradually moves toward SpeedX)
	SpriteSlopeY int8          // Visual speed Y used during transition (gradually moves toward SpeedY)
	Score        RockScoreType //  how many 'rocks' this rock counts for during scoring. also determines size, etc

	transitionSteps uint8 // Bit-packed: lower 4 bits = X remaining steps, upper 4 bits = Y remaining steps

}

// const BaseVelocity = 1.0

const BaseVelocity = 2.0

func (r *SimpleRock) RockWithinDie(die *DieRenderable, rockSize float32) bool {
	// AABB (Axis-Aligned Bounding Box) collision detection with 0.75 multiplier for tighter collision
	// Checks if rock's bounding box overlaps with die's bounding box

	effectiveDieTileSize := DieTileSize * 0.75
	effectiveRockSize := rockSize * 0.75

	// Center the effective collision boxes
	dieInset := (DieTileSize - effectiveDieTileSize) / 2
	rockInset := (rockSize - effectiveRockSize) / 2

	return (r.Position.X+rockInset+effectiveRockSize > die.Vec2.X+dieInset && r.Position.X+rockInset < die.Vec2.X+dieInset+effectiveDieTileSize) &&
		(r.Position.Y+rockInset+effectiveRockSize > die.Vec2.Y+dieInset && r.Position.Y+rockInset < die.Vec2.Y+dieInset+effectiveDieTileSize)
}

// TODO: determine if copy is faster than reference, and baseSpriteSize copy by ref or get it from g *Game
func (r *SimpleRock) XYWithinRock(X, Y float32, spriteSize float32) bool {
	return (Y >= r.Position.Y && Y <= r.Position.Y+spriteSize) && (X >= r.Position.X && X <= r.Position.X+spriteSize)
}

// IsNearPoint checks if rock is within radius of a point using Manhattan distance (fast, no sqrt)
// This is the BROAD PHASE collision check - cheaper than precise AABB collision
func (r *SimpleRock) IsNearPoint(rockSize, pointX, pointY, radius float32) bool {
	rockCenterX := r.Position.X + rockSize/2
	rockCenterY := r.Position.Y + rockSize/2

	dx := rockCenterX - pointX
	dy := rockCenterY - pointY

	// Manhattan distance approximation (fast, no sqrt)
	manhattanDist := float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
	return manhattanDist < radius
}

// IsNearDie checks if rock is within radius of a die's center using Manhattan distance
// This is the BROAD PHASE collision check - cheaper than precise AABB collision
func (r *SimpleRock) IsNearDie(rockSize float32, die *DieRenderable, radius float32) bool {
	dieCenterX := die.Vec2.X + HalfDieTileSize
	dieCenterY := die.Vec2.Y + HalfDieTileSize

	return r.IsNearPoint(rockSize, dieCenterX, dieCenterY, radius)
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
func (r *SimpleRock) GetSize(baseSpriteSize float32) float32 {
	return baseSpriteSize * r.Score.SizeMultiplier()
}

func calculateShortestPath(current, target int8) (distance uint8) {
	diff := target - current

	if diff > DIRECTIONS_TO_SNAP/2 {
		// Going backwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP - diff)
	} else if diff < -DIRECTIONS_TO_SNAP/2 {
		// Going forwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP + diff)
	} else if diff > 0 {
		// Normal forward
		distance = uint8(diff)
	} else if diff < 0 {
		// Normal backward
		distance = uint8(-diff)
	} else {
		// Already at target
		distance = 0
	}

	return distance
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
	distX := calculateShortestPath(r.SpriteSlopeX, targetX)

	// Calculate transition steps for Y
	targetY := newY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP {
		targetY = 0
	}
	distY := calculateShortestPath(r.SpriteSlopeY, targetY)

	// Add full rotation (DIRECTIONS_TO_SNAP) to shortest path
	r.setTransitionSteps(distX+uint8(DIRECTIONS_TO_SNAP), distY+uint8(DIRECTIONS_TO_SNAP))
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
	distX := calculateShortestPath(r.SpriteSlopeX, targetX)

	// When bouncing off vertical wall, tumble around Y-axis based on Y speed
	// Y slope doesn't change, so just tumble
	absY := r.SlopeY
	if absY < 0 {
		absY = -absY
	}
	tumbleY := uint8(absY) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(distX, tumbleY)
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
	distY := calculateShortestPath(r.SpriteSlopeY, targetY)

	// When bouncing off horizontal wall, tumble around X-axis based on X speed
	// X slope doesn't change, so just tumble
	absX := r.SlopeX
	if absX < 0 {
		absX = -absX
	}
	tumbleX := uint8(absX) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(tumbleX, distY)
}

// UpdateTransition handles the smooth sprite rotation during direction changes
// Now includes full rotation on each bounce
func (r *SimpleRock) UpdateTransition(frameCounter int) {
	// Update SpriteIndex based on rock SIZE (smaller rocks rotate faster)
	if frameCounter%r.Score.GetScore() == 0 {
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
	// Derive animation rate on-the-fly from current slopes (faster than storing/loading from memory)
	// Fast approximation: max(|x|, |y|) + min(|x|, |y|)/2
	absX := r.SlopeX
	if absX < 0 {
		absX = -absX
	}
	absY := r.SlopeY
	if absY < 0 {
		absY = -absY
	}

	var speed int8
	if absX > absY {
		speed = absX + absY/2
	} else {
		speed = absY + absX/2
	}

	n := int(baseN) - int(speed)*3 // Simplified from speed*speedFactor (3.5 → 3 for int math)
	if n < 2 {
		n = 2
	}

	if frameCounter%n != 0 {
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
// Uses trigonometry to accurately map angles to SlopeX/SlopeY components
func (r *SimpleRock) BounceTowardsAngle(angle int) {
	// Convert angle to radians for trigonometric functions
	angleRad := float64(angle) * math.Pi / 180.0

	// Calculate velocity components using trigonometry
	speedX := math.Cos(angleRad)
	speedY := math.Sin(angleRad)

	// Scale to int8 range (-4 to +4) while maintaining ratio
	const maxVal = float64(MAX_SLOPE) // 4

	// Scale both components so the larger one reaches maxVal
	maxComponent := math.Max(math.Abs(speedX), math.Abs(speedY))
	scale := maxVal / maxComponent
	targetSlopeX := int8(math.Round(speedX * scale))
	targetSlopeY := int8(math.Round(speedY * scale))

	r.Bounce(targetSlopeX, targetSlopeY)
}

// RocksRenderer manages pre-extracted sprite rendering with ultra-fast array indexing
type RocksRenderer struct {
	shader      *ebiten.Shader
	colorShader *ebiten.Shader // Color filter shader for applying colors to grayscale sprites

	// 2D array of grayscale sprite sheets (shared by all rock types)
	// [speedX_index][speedY_index] -> Sprite struct containing spritesheet
	// Each spritesheet has ROTATION_FRAMES (72) frames arranged in 18 columns x 4 rows
	// Size: [8][8] = 64 spritesheets total (was 128 with NUM_ROCK_TYPES before!)
	sprites [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*Sprite

	// Color tints for each rock type (applied via shader at draw time)
	rockTypeColors [NUM_ROCK_TYPES]Vec3

	Rocks          [NUM_ROCK_TYPES][]*SimpleRock // Rocks organized by type
	RockTileSize   float32                       // Base tile size for rock rendering and collision calculations
	totalRocks     int
	ActiveRockType int
	FrameCounter   [NUM_ROCK_TYPES]int // Global frame counter for transition timing

	// Collision check radii for this rock renderer
	CursorCheckRadius float32 // Distance from cursor to check for rock collisions
	DieCheckRadius    float32 // Distance from die center to check for rock collisions

	// Internal collision buffers - reused each frame to avoid allocations
	diceCollisionBuffer   []*SimpleRock
	cursorCollisionBuffer []*SimpleRock

	// Pre-allocated temp images for rendering (reused every frame to avoid allocations)
	// [NUM_ROCK_TYPES] temp images, one per rock color type
	tempImages [NUM_ROCK_TYPES]*ebiten.Image
}

// RocksConfig holds configuration for rock system
type RocksConfig struct {
	TotalRocks   int
	RockTileSize float32 // Base tile size for rock rendering and collision calculations
	WorldBoundsX float32
	WorldBoundsY float32
}

// NewRocksRenderer creates a new ultra-fast sprite rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()
	r := &RocksRenderer{
		shader:       shaderMap[shaders.RocksShaderKey],
		colorShader:  shaderMap[shaders.ColorFilterShaderKey],
		RockTileSize: config.RockTileSize,
		totalRocks:   config.TotalRocks,
		// Initialize collision check radii based on RockTileSize
		CursorCheckRadius: config.RockTileSize,
		DieCheckRadius:    config.RockTileSize,
		// Pre-allocate collision buffers with typical capacity to avoid allocations
		// Capacity based on typical collision counts: ~50 dice collisions, ~20 cursor collisions
		diceCollisionBuffer:   make([]*SimpleRock, 0, 128),
		cursorCollisionBuffer: make([]*SimpleRock, 0, 128),
		// Define colors for each rock type (applied via shader at draw time)
		rockTypeColors: [NUM_ROCK_TYPES]Vec3{
			Grey,
			Brown,
		},
	}

	// Generate and pre-extract all sprite frames (single grayscale spritesheet)
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	// Pre-allocate temp images for rendering (reused every frame to avoid allocations)
	for i := 0; i < NUM_ROCK_TYPES; i++ {
		r.tempImages[i] = ebiten.NewImage(int(config.WorldBoundsX), int(config.WorldBoundsY))
	}

	return r
}

// generateSprites creates a single grayscale spritesheet array (shared by all rock types)
// Colors will be applied at draw-time via the color filter shader
func (r *RocksRenderer) generateSprites() {
	genSprites := [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*Sprite{}

	// Use grayscale values for base sprite (will be colored via shader later)
	// Different grayscale tones create visual variety in the base geometry
	innerDark := KageColor(60, 60, 60)     // Dark gray for inner/crater areas
	innerLight := KageColor(200, 200, 200) // Light gray for inner/crater areas
	outerDark := KageColor(100, 100, 100)  // Medium gray for outer surface
	outerLight := KageColor(220, 220, 220) // Light gray for outer surface

	for XSnapIdx := range DIRECTIONS_TO_SNAP {
		angleDegX := int(XSnapIdx) * (360 / int(DIRECTIONS_TO_SNAP))
		angleRadX := float32(angleDegX) * (math.Pi / 180.0)

		for YSnapIdx := range DIRECTIONS_TO_SNAP {
			angleDegY := int(YSnapIdx) * (360 / int(DIRECTIONS_TO_SNAP))
			angleRadY := float32(angleDegY) * (math.Pi / 180.0)

			// Create spritesheet based on ROTATION_FRAMES
			spriteSize := int(r.RockTileSize)
			sheetWidth := spriteSize * SHEET_COLS
			sheetHeight := spriteSize * SHEET_ROWS
			spriteSheet := ebiten.NewImage(sheetWidth, sheetHeight)

			// Render all rotation frames into the spritesheet
			for frameIdx := 0; frameIdx < ROTATION_FRAMES; frameIdx++ {
				// Calculate Z rotation angle (0 to 360 degrees in DEGREES_PER_FRAME increments)
				rotationRadAngle := float32(frameIdx*DEGREES_PER_FRAME) * (math.Pi / 180.0)

				// Create temporary image for this frame
				frameImg := ebiten.NewImage(spriteSize, spriteSize)

				u := map[string]interface{}{
					"Time":            0.0,
					"Resolution":      []float32{r.RockTileSize, r.RockTileSize},
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
				frameImg.DrawRectShader(spriteSize, spriteSize, r.shader, opts)

				// Calculate position in spritesheet (row-major order)
				col := frameIdx % SHEET_COLS
				row := frameIdx / SHEET_COLS

				// Draw this frame into the spritesheet at the correct position
				drawOpts := &ebiten.DrawImageOptions{}
				drawOpts.GeoM.Translate(float64(col*spriteSize), float64(row*spriteSize))
				spriteSheet.DrawImage(frameImg, drawOpts)
			}

			// Create Sprite struct with spritesheet metadata
			sprite := Sprite{
				Image:       spriteSheet,
				SpriteSheet: NewSpriteSheet(SHEET_COLS, SHEET_ROWS, spriteSize),
				ActiveFrame: 0,
			}

			if XSnapIdx >= DIRECTIONS_TO_SNAP {
				continue
			} else if YSnapIdx >= DIRECTIONS_TO_SNAP {
				continue
			}

			genSprites[XSnapIdx][YSnapIdx] = &sprite
		}
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

		// Pick a random RockScoreType variant that doesn't exceed remaining

		//TODO: get a good distribution of rock sizes instead of % chance
		var scoreType RockScoreType
		switch {
		case remaining >= HugeScore && rand.Float32() < 0.15: // 15% chance for Huge
			// Pick random Huge variant (10, 11, or 12)
			scoreType = HugeLarge + RockScoreType(rand.Intn(3))
		case remaining >= BigScore && rand.Float32() < 0.25: // 25% chance for Big
			// Pick random Big variant (7, 8, or 9)
			scoreType = BigLarge + RockScoreType(rand.Intn(3))
		case remaining >= MediumScore && rand.Float32() < 0.35: // 35% chance for Medium
			// Pick random Medium variant (4, 5, or 6)
			scoreType = MediumLarge + RockScoreType(rand.Intn(3))
		default: // Otherwise Small
			// Pick random Small variant (1, 2, or 3)
			scoreType = SmallLarge + RockScoreType(rand.Intn(3))
		}

		// Pick random rock type (0 or 1)
		rockType := rand.Intn(NUM_ROCK_TYPES)

		// Random position
		pos := Vec2{
			X: rand.Float32() * config.WorldBoundsX,
			Y: rand.Float32() * config.WorldBoundsY,
		}

		// Pick random rotation frame
		spriteIndex := uint8(rand.Intn(ROTATION_FRAMES))

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

		allRocks[rockType] = append(allRocks[rockType], rock)
		currentScore += scoreType.GetScore()
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
// Uses grayscale sprites with color filter shader applied per rock type
// Implements index-based interleaving to prevent one color from always appearing on top
func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	// Reuse DrawImageOptions to avoid allocations (important for 10k+ rocks)
	opts := &ebiten.DrawImageOptions{}
	// opts.Filter = ebiten.FilterLinear

	// Interleave rocks by drawing them in layers based on their index
	// This ensures colors are mixed visually rather than one color always on top
	for layer := 0; layer < NUM_INTERLEAVE_LAYERS; layer++ {
		for rockType := range NUM_ROCK_TYPES {
			// Clear the pre-allocated temp image (reuse instead of allocating new one each frame!)
			r.tempImages[rockType].Clear()

			// Draw only rocks whose index matches this layer using stride-based iteration
			// This avoids expensive modulo operations and conditional checks
			// Layer 0: indices 0, 2, 4, 6... | Layer 1: indices 1, 3, 5, 7...
			for i := layer; i < len(r.Rocks[rockType]); i += NUM_INTERLEAVE_LAYERS {
				rock := r.Rocks[rockType][i]

				// Get the grayscale sprite for this slope combination (same for all rock types)
				sprite := r.sprites[rock.SpriteSlopeX][rock.SpriteSlopeY]

				// Get the specific rotation frame from the spritesheet
				frameRect := sprite.SpriteSheet.Rect(int(rock.SpriteIndex))
				frameImage := sprite.Image.SubImage(frameRect).(*ebiten.Image)

				// Reset and set transform with SCALING based on RockScoreType
				opts.GeoM.Reset()
				scale := rock.Score.SizeMultiplier()
				opts.GeoM.Scale(float64(scale), float64(scale))
				opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
				r.tempImages[rockType].DrawImage(frameImage, opts)
			}

			// Apply color shader to all rocks of this type in this layer
			colorOpts := &ebiten.DrawRectShaderOptions{
				Images: [4]*ebiten.Image{r.tempImages[rockType], nil, nil, nil},
				Uniforms: map[string]interface{}{
					"TintColor": r.rockTypeColors[rockType].KageVec3(),
				},
			}
			screen.DrawRectShader(int(GAME_BOUNDS_X), int(GAME_BOUNDS_Y), r.colorShader, colorOpts)
		}
	}
}

func (r *RocksRenderer) GetStats() (visible, total int) {
	return r.totalRocks, r.totalRocks
}

func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// dieCollisionData holds pre-computed collision data for a die
type dieCollisionData struct {
	centerX, centerY         float32
	velocitySlopeX           int8
	velocitySlopeY           int8
	left, right, top, bottom float32
}

// UpdateAndHandleCollisions performs all rock updates, wall bouncing, and collision detection/response
func (r *RocksRenderer) UpdateAndHandleCollisions(cursorX, cursorY float32, dice []*DieRenderable) {
	// Advance frame counter and rock type
	r.ActiveRockType++
	if r.ActiveRockType >= NUM_ROCK_TYPES {
		r.ActiveRockType = 0
	}
	r.FrameCounter[r.ActiveRockType]++

	// Reset collision buffers to length 0 (keeps capacity - no allocation!)
	r.diceCollisionBuffer = r.diceCollisionBuffer[:0]
	r.cursorCollisionBuffer = r.cursorCollisionBuffer[:0]

	// PASS 1: BROAD PHASE - Update all rocks and collect collision candidates
	for _, rock := range r.Rocks[r.ActiveRockType] {
		rock.Update(r.FrameCounter[r.ActiveRockType])

		rockSize := rock.GetSize(r.RockTileSize)

		// Wall bouncing
		if rock.Position.X+rockSize >= GAME_BOUNDS_X {
			rock.Position.X = GAME_BOUNDS_X - rockSize
			rock.BounceX()
		} else if rock.Position.X <= 0 {
			rock.Position.X = 0
			rock.BounceX()
		}

		if rock.Position.Y+rockSize >= GAME_BOUNDS_Y {
			rock.Position.Y = GAME_BOUNDS_Y - rockSize
			rock.BounceY()
		} else if rock.Position.Y <= 0 {
			rock.Position.Y = 0
			rock.BounceY()
		}

		// BROAD PHASE: Collect rocks near cursor
		if rock.IsNearPoint(rockSize, cursorX, cursorY, r.CursorCheckRadius) {
			r.cursorCollisionBuffer = append(r.cursorCollisionBuffer, rock)
		}

		// BROAD PHASE: Collect rocks near any die
		for _, die := range dice {
			if rock.IsNearDie(rockSize, die, r.DieCheckRadius) {
				r.diceCollisionBuffer = append(r.diceCollisionBuffer, rock)
				break // Only add once even if near multiple dice
			}
		}
	}

	// PASS 2: NARROW PHASE - Precise collision checks and responses
	r.handleCursorCollisions(cursorX, cursorY)
	r.handleDieCollisions(dice)
}

// handleCursorCollisions processes cursor-rock collision responses
func (r *RocksRenderer) handleCursorCollisions(cursorX, cursorY float32) {
	for _, rock := range r.cursorCollisionBuffer {
		rockSize := rock.GetSize(r.RockTileSize)

		if rock.XYWithinRock(cursorX, cursorY, rockSize) {
			// Determine which side of rock the cursor is on
			rockCenterX := rock.Position.X + rockSize/2
			rockCenterY := rock.Position.Y + rockSize/2
			dx := cursorX - rockCenterX
			dy := cursorY - rockCenterY

			// Push rock away from cursor on the primary collision axis
			if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
				// Horizontal collision
				if dx > 0 {
					// Cursor is on right side, push rock left
					rock.Position.X = cursorX - rockSize - 1
				} else {
					// Cursor is on left side, push rock right
					rock.Position.X = cursorX + 1
				}
				rock.BounceX()
			} else {
				// Vertical collision
				if dy > 0 {
					// Cursor is below, push rock up
					rock.Position.Y = cursorY - rockSize - 1
				} else {
					// Cursor is above, push rock down
					rock.Position.Y = cursorY + 1
				}
				rock.BounceY()
			}
		}
	}
}

// handleDieCollisions processes die-rock collision responses with pre-calculated die data
func (r *RocksRenderer) handleDieCollisions(dice []*DieRenderable) {
	if len(r.diceCollisionBuffer) == 0 {
		return
	}

	// Calculate once for all collisions - dice use their own DieTileSize
	effectiveTileSize := DieTileSize * 0.75
	dieInset := (DieTileSize - effectiveTileSize) / 2

	// Pre-compute per-die data to avoid redundant calculations
	diceData := make([]dieCollisionData, len(dice))
	for i, die := range dice {
		diceData[i] = dieCollisionData{
			centerX:        die.Vec2.X + HalfDieTileSize,
			centerY:        die.Vec2.Y + HalfDieTileSize,
			velocitySlopeX: int8(die.Velocity.X / BaseVelocity),
			velocitySlopeY: int8(die.Velocity.Y / BaseVelocity),
			left:           die.Vec2.X + dieInset,
			right:          die.Vec2.X + dieInset + effectiveTileSize,
			top:            die.Vec2.Y + dieInset,
			bottom:         die.Vec2.Y + dieInset + effectiveTileSize,
		}
	}

	// Process collisions with pre-calculated data
	for _, rock := range r.diceCollisionBuffer {
		rockSize := rock.GetSize(r.RockTileSize)
		effectiveRockSize := rockSize * 0.75
		rockInset := (rockSize - effectiveRockSize) / 2
		rockCenterX := rock.Position.X + rockSize/2
		rockCenterY := rock.Position.Y + rockSize/2

		for i, die := range dice {
			if rock.RockWithinDie(die, rockSize) {
				data := diceData[i]

				// Calculate which edge is closest
				dx := rockCenterX - data.centerX
				dy := rockCenterY - data.centerY

				// Determine primary collision axis based on which has greater separation
				if math.Abs(float64(dx)) > math.Abs(float64(dy)) {
					// Horizontal collision (left or right side of die)
					if dx > 0 {
						// Rock is on right side of die
						rock.Position.X = data.right + 1 - rockInset
					} else {
						// Rock is on left side of die
						rock.Position.X = data.left - effectiveRockSize - 1 - rockInset
					}

					// Add die velocity to rock's X slope, clamp to MAX_SLOPE range
					newSlopeX := -rock.SlopeX + data.velocitySlopeX
					if newSlopeX > MAX_SLOPE {
						newSlopeX = MAX_SLOPE
					} else if newSlopeX < MIN_SLOPE {
						newSlopeX = MIN_SLOPE
					}
					rock.Bounce(newSlopeX, rock.SlopeY)
				} else {
					// Vertical collision (top or bottom side of die)
					if dy > 0 {
						// Rock is below die
						rock.Position.Y = data.bottom + 1 - rockInset
					} else {
						// Rock is above die
						rock.Position.Y = data.top - effectiveRockSize - 1 - rockInset
					}

					// Add die velocity to rock's Y slope, clamp to MAX_SLOPE range
					newSlopeY := -rock.SlopeY + data.velocitySlopeY
					if newSlopeY > MAX_SLOPE {
						newSlopeY = MAX_SLOPE
					} else if newSlopeY < MIN_SLOPE {
						newSlopeY = MIN_SLOPE
					}
					rock.Bounce(rock.SlopeX, newSlopeY)
				}
			}
		}
	}
}

// CalculateRockTileSize dynamically calculates rock tile size based on the number of rocks
// Uses tiered scaling system:
//
//	< 100 rocks    → 2.0× base tile size (large rocks)
//	100-1000 rocks → 1.0× base tile size (normal)
//	1000-10000     → 0.75× base tile size (smaller)
//	> 10000 rocks  → 0.5× base tile size (tiny)
func CalculateRockTileSize(baseTileSize float32, rockAmount int) float32 {
	var scaleFactor float32

	if rockAmount <= 100 {
		scaleFactor = 2.0
	} else if rockAmount <= 1000 {
		scaleFactor = 1.5
	} else if rockAmount <= 10000 {
		scaleFactor = 1.0
	} else {
		scaleFactor = 1.0
	}

	return baseTileSize * scaleFactor
}
