package render

import (
	"math"
	"math/rand"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// Constants for sprite system
const (
	// NUM_ROCK_TYPES = 2 // 2 rock types: different colors

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
	NUM_INTERLEAVE_LAYERS = 2
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
	SmallScore        = 1
	MediumScore       = 3
	BigScore          = 5
	HugeScore         = 10
	rockScoreVariants = 3

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

const BaseVelocity = 1.0

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

// // IsNearDie checks if rock is within radius of a die's center using Manhattan distance
// // This is the BROAD PHASE collision check - cheaper than precise AABB collision
// func (r *SimpleRock) IsNearDie(rockSize float32, die *DieRenderable, radius float32) bool {
// 	dieCenterX := die.Vec2.X + HalfDieTileSize
// 	dieCenterY := die.Vec2.Y + HalfDieTileSize

// 	return r.IsNearPoint(rockSize, dieCenterX, dieCenterY, radius)
// }

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
// shouldSlowDown: if true, rock gradually slows down over time (for base/transition buffers)
func (r *SimpleRock) Update(frameCounter int, shouldSlowDown bool) {
	r.Position.Y += BaseVelocity * float32(r.SlopeY)
	r.Position.X += BaseVelocity * float32(r.SlopeX)

	r.UpdateTransition(frameCounter)

	// Apply slowdown effect if enabled (base and transition buffers only)
	if shouldSlowDown && frameCounter%15 == 0 { // Slow down every 15 frames (~0.25 seconds at 60fps)
		// Gradually reduce slope toward zero
		if r.SlopeX > 0 {
			r.SlopeX--
		} else if r.SlopeX < 0 {
			r.SlopeX++
		}

		if r.SlopeY > 0 {
			r.SlopeY--
		} else if r.SlopeY < 0 {
			r.SlopeY++
		}
	}
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

// TODO: need to have static rocks update rockTransition vs the sprite index
// UpdateTransition handles the smooth sprite rotation during direction changes
// Now includes full rotation on each bounce
func (r *SimpleRock) UpdateTransition(frameCounter int) {
	// Update SpriteIndex based on rock SIZE (smaller rocks rotate faster)
	if frameCounter%r.Score.GetScore() == 0 {
		// Increment or decrement sprite index based on horizontal direction
		// Moving right (positive SlopeX): increment (rotate clockwise)
		// Moving left (negative SlopeX): decrement (rotate counter-clockwise)
		if r.SlopeX != 0 && r.SlopeY != 0 {
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
		//  else {
		// Stationary rocks: cycle through transition slopes for visual variety
		// r.SpriteSlopeX++
		// if r.SpriteSlopeX >= DIRECTIONS_TO_SNAP {
		// r.SpriteSlopeX = 0
		// r.SpriteSlopeY++
		// if r.SpriteSlopeY >= DIRECTIONS_TO_SNAP {
		// r.SpriteSlopeY = 0
		// }
		// }
		// }
	}

	if r.SlopeX == 0 && r.SlopeY == 0 {
		return
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

	// If rock is completely stopped, don't update sprite slopes
	// Keep SpriteSlopeX/Y frozen at current values
	if r.SlopeX == 0 && r.SlopeY == 0 {
		return
	}

	// Always update sprite slopes by +1 (transitions always increment)
	// X component
	if r.getTransitionStepsX() > 0 {
		r.SpriteSlopeX++
		if r.SpriteSlopeX >= DIRECTIONS_TO_SNAP {
			r.SpriteSlopeX = 0
		}
		r.decrementTransitionStepX()
	}

	// Y component
	if r.getTransitionStepsY() > 0 {
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

type RockBuffer struct {
	Rocks           []SimpleRock
	TransitionColor Vec3
	Color           Vec3
	Transition      int
	FrameCounter    int
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

	// Three-tier buffer system for rock color management
	BaseColorBuffers  []RockBuffer                // Source rocks (Grey, Brown, etc.)
	HeldColorBuffers  map[DieIdentity]*RockBuffer // Rocks owned by held dice
	TransitionBuffers []*RockBuffer               // Rocks transitioning back to base colors

	totalRocks int

	ActiveRockFlag bool // true/false to update BaseColorBuffers[even] or BaseColorBuffers[odd]

	RockTileSize float32 // Base tile size for rock rendering and collision calculations

	// Collision check radii for this rock renderer
	CursorCheckRadius float32 // Distance from cursor to check for rock collisions
	DieCheckRadius    float32 // Distance from die center to check for rock collisions

	// Internal collision buffers - reused each frame to avoid allocations
	diceCollisionBuffer   []*SimpleRock
	cursorCollisionBuffer []*SimpleRock

	// Image pool for temporary rendering buffers (lazily allocated and reused every frame)
	imagePool *ImagePool

	config RocksConfig
}

// RocksConfig holds configuration for rock system
type RocksConfig struct {
	TotalRocks            int
	BaseColors            []Vec3  // the colors that rock render applies
	RockTileSize          float32 // Base tile size for rock rendering and collision calculations
	WorldBoundsX          float32
	WorldBoundsY          float32
	ColorTransitionFrames int // frames for color transitions (default: 30)
}

// func (r *RocksRenderer) SetActiveBuffers() {
// 	//clear buffer
// 	r.ActiveBuffers = r.ActiveBuffers[:0]

// 	for _, buffer := range r.RockBuffers {
// 		r.ActiveBuffers = append(r.ActiveBuffers, &buffer)
// 	}
// 	for _, buffer := range r.HeldRockBuffers {
// 		r.ActiveBuffers = append(r.ActiveBuffers, &buffer)
// 	}
// }

// NewRocksRenderer creates a new ultra-fast sprite rendering system
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()

	r := &RocksRenderer{
		shader:       shaderMap[shaders.RocksShaderKey],
		colorShader:  shaderMap[shaders.ColorFilterShaderKey],
		RockTileSize: config.RockTileSize,
		totalRocks:   config.TotalRocks,
		// Initialize collision check radii - TIGHT buffers to reduce expensive collision calculations
		// Accepts that some edge-case collisions at buffer boundaries may be missed

		CursorCheckRadius: config.RockTileSize * 0.6, // Reduced from 1.0x to 0.6x
		DieCheckRadius:    config.RockTileSize * 0.8, // Reduced from 1.0x to 0.8x
		// Pre-allocate collision buffers with typical capacity to avoid allocations
		// Capacity based on typical collision counts: ~50 dice collisions, ~20 cursor collisions
		diceCollisionBuffer:   make([]*SimpleRock, 0, 128),
		cursorCollisionBuffer: make([]*SimpleRock, 0, 128),

		config: config,
		// ActiveBuffers: make([]*RockBuffer, 0, 16),
		// Define colors for each rock type (applied via shader at draw time)
	}

	// Initialize empty rock buffers slice (will grow dynamically)
	r.BaseColorBuffers = make([]RockBuffer, 0, len(config.BaseColors))
	r.HeldColorBuffers = make(map[DieIdentity]*RockBuffer)
	r.TransitionBuffers = make([]*RockBuffer, 0, 10)

	// Generate and pre-extract all sprite frames (single grayscale spritesheet)
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	// Initialize image pool for temporary rendering (lazy allocation)
	r.imagePool = NewImagePool(int(config.WorldBoundsX), int(config.WorldBoundsY))

	return r
}

// generateSprites creates a single grayscale spritesheet array (shared by all rock types)
// Colors will be applied at draw-time via the color filter shader
func (r *RocksRenderer) generateSprites() {
	genSprites := [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*Sprite{}

	// Use white values for base sprite (will be colored via shader later)
	// Different white tones create visual variety in the base geometry
	innerDark := WhiteDark    // Darker white for inner/crater areas
	innerLight := WhiteMid    // Medium white for inner/crater areas
	outerDark := WhiteLight   // Light white for outer surface
	outerLight := WhiteBright // Pure white for outer surface highlights

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
	var allRocks = make([][]SimpleRock, 0, len(config.BaseColors))
	for range len(config.BaseColors) {
		allRocks = append(allRocks, []SimpleRock{})
	}

	var curRockTypeIndex int

	// Generate rocks until we reach target score
	for currentScore < targetScore {
		remaining := targetScore - currentScore

		// Pick a random RockScoreType variant that doesn't exceed remaining

		//TODO: get a good distribution of rock sizes instead of % chance
		// could be based on rock config?
		var scoreType RockScoreType
		switch {
		case remaining >= HugeScore && rand.Float32() < 0.15: // 15% chance for Huge
			// Pick random Huge variant (10, 11, or 12)
			scoreType = HugeLarge + RockScoreType(rand.Intn(rockScoreVariants))
		case remaining >= BigScore && rand.Float32() < 0.25: // 25% chance for Big
			// Pick random Big variant (7, 8, or 9)
			scoreType = BigLarge + RockScoreType(rand.Intn(rockScoreVariants))
		case remaining >= MediumScore && rand.Float32() < 0.35: // 35% chance for Medium
			// Pick random Medium variant (4, 5, or 6)
			scoreType = MediumLarge + RockScoreType(rand.Intn(rockScoreVariants))
		default: // Otherwise Small
			// Pick random Small variant (1, 2, or 3)
			scoreType = SmallLarge + RockScoreType(rand.Intn(rockScoreVariants))
		}

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

		rock := SimpleRock{
			Position:    pos,
			SpriteIndex: spriteIndex,
			// SlopeX:       slopeX,
			// SlopeY:       slopeY,
			SlopeX:       0,
			SlopeY:       0,
			SpriteSlopeX: 0,
			SpriteSlopeY: 0,
			Score:        scoreType,
		}

		allRocks[curRockTypeIndex] = append(allRocks[curRockTypeIndex], rock)
		currentScore += scoreType.GetScore()

		curRockTypeIndex++
		if curRockTypeIndex >= len(config.BaseColors) {
			curRockTypeIndex = 0
		}
	}

	// Assign to renderer using append for dynamic growth
	for i := range len(allRocks) {
		r.BaseColorBuffers = append(r.BaseColorBuffers, RockBuffer{
			Color:           config.BaseColors[i],
			TransitionColor: config.BaseColors[i],
			Transition:      0,
			Rocks:           allRocks[i],
		})
	}

	// Update total count
	// r.totalRocks = 0
	// for i := 0; i < NUM_ROCK_TYPES; i++ {
	// 	r.totalRocks += len(r.Rocks[i])
	// }
}

// drawBufferToImage draws all rocks from a buffer to a temporary image
func (r *RocksRenderer) drawBufferToImage(
	buffer *RockBuffer,
	tempImage *ebiten.Image,
	opts *ebiten.DrawImageOptions,
) {
	// Note: tempImage is already cleared by imagePool.GetNext()

	for _, rock := range buffer.Rocks {
		sprite := r.sprites[rock.SpriteSlopeX][rock.SpriteSlopeY]
		frameRect := sprite.SpriteSheet.Rect(int(rock.SpriteIndex))
		frameImage := sprite.Image.SubImage(frameRect).(*ebiten.Image)

		opts.GeoM.Reset()
		scale := rock.Score.SizeMultiplier()
		opts.GeoM.Scale(float64(scale), float64(scale))
		opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
		tempImage.DrawImage(frameImage, opts)
	}
}

// drawWithColorShader applies color tint shader to a temp image and draws to screen
// Handles color transitions via shader uniforms (GPU-side mixing)
func (r *RocksRenderer) drawWithColorShader(
	buffer *RockBuffer,
	tempImage *ebiten.Image,
	screen *ebiten.Image,
) {
	// Calculate transition amount (0.0 to 1.0)
	var transitionAmount float32
	if buffer.Transition > 0 {
		// Transition counter counts down from max to 0
		// At start: Transition = max, transitionAmount = 1.0 (full TransitionColor)
		// At end: Transition = 0, transitionAmount = 0.0 (full Color)
		transitionAmount = float32(buffer.Transition) / float32(r.config.ColorTransitionFrames)
	} else {
		transitionAmount = 0.0 // No transition, use ColorFrom only
	}

	colorOpts := &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{tempImage, nil, nil, nil},
		Uniforms: map[string]interface{}{
			"ColorFrom":        buffer.Color.KageVec3(),           // Target color (where we end up)
			"ColorTo":          buffer.TransitionColor.KageVec3(), // Source color (where we start)
			"TransitionAmount": transitionAmount,                  // 1.0 at start, 0.0 at end
		},
	}
	screen.DrawRectShader(int(GAME_BOUNDS_X), int(GAME_BOUNDS_Y), r.colorShader, colorOpts)
}

// DrawRocks renders all rocks with ultra-fast direct array access
// Uses grayscale sprites with color filter shader applied per rock type
// Implements index-based interleaving to prevent one color from always appearing on top
func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	// Reset image pool for this frame
	r.imagePool.Reset()

	// Reuse DrawImageOptions to avoid allocations (important for 10k+ rocks)
	opts := &ebiten.DrawImageOptions{}

	// Render base color buffers with interleaving
	for layer := 0; layer < NUM_INTERLEAVE_LAYERS; layer++ {
		for rockType := range len(r.BaseColorBuffers) {
			buffer := &r.BaseColorBuffers[rockType]

			// Get next temp image from pool (already cleared)
			tempImg := r.imagePool.GetNext()

			// Draw rocks with interleaving
			for i := layer; i < len(buffer.Rocks); i += NUM_INTERLEAVE_LAYERS {
				rock := buffer.Rocks[i]
				sprite := r.sprites[rock.SpriteSlopeX][rock.SpriteSlopeY]
				frameImage := sprite.Image.SubImage(
					sprite.SpriteSheet.Rect(int(rock.SpriteIndex)),
				).(*ebiten.Image)

				opts.GeoM.Reset()
				scale := rock.Score.SizeMultiplier()
				opts.GeoM.Scale(float64(scale), float64(scale))
				opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
				tempImg.DrawImage(frameImage, opts)
			}

			// Apply color shader (base buffers should have Transition = 0, so just use base color)
			r.drawWithColorShader(buffer, tempImg, screen)
		}
	}

	// Render held color buffers
	for i := range len(RainbowColors) {
		buffer, ok := r.HeldColorBuffers[DieIdentity(i)]
		if !ok {
			continue
		}

		tempImg := r.imagePool.GetNext()
		r.drawBufferToImage(buffer, tempImg, opts)
		r.drawWithColorShader(buffer, tempImg, screen)
	}

	// Render transition buffers
	for _, buffer := range r.TransitionBuffers {
		tempImg := r.imagePool.GetNext()
		r.drawBufferToImage(buffer, tempImg, opts)
		r.drawWithColorShader(buffer, tempImg, screen)
	}
}

func (r *RocksRenderer) GetStats() (visible, total int) {
	return r.totalRocks, r.totalRocks
}

// Pre-computed die collision constants (computed once at init time)
var (
	EffectiveDieTileSize float32
	DieTileInset         float32
	HalfEffectiveDie     float32 // EffectiveDieTileSize / 2, for computing bounds from center
)

// updateBufferRocks updates all rocks in a buffer: physics, wall bouncing, and collision detection
func (r *RocksRenderer) updateBufferRocks(
	buffer *RockBuffer,
	cursorX, cursorY float32,
	diceCenters []Vec3,
	shouldSlowDown bool,
) {
	buffer.FrameCounter++

	for i := range buffer.Rocks {
		rock := &buffer.Rocks[i]
		rock.Update(buffer.FrameCounter, shouldSlowDown)

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

		// BROAD PHASE: Collect rocks near any die center
		for j := range diceCenters {
			if rock.IsNearPoint(rockSize, diceCenters[j].X, diceCenters[j].Y, r.DieCheckRadius) {
				r.diceCollisionBuffer = append(r.diceCollisionBuffer, rock)
				break // Only add once even if near multiple dice
			}
		}
	}
}

// updateAllBufferTransitions decrements transition counters for all buffer types
func (r *RocksRenderer) updateAllBufferTransitions() {
	// Update HeldColorBuffers transitions (stop at 0, don't go negative)
	for _, buffer := range r.HeldColorBuffers {
		if buffer.Transition > 0 {
			buffer.Transition--
		}
	}

	// Update TransitionBuffers transitions and move completed ones to base buffers
	for i := len(r.TransitionBuffers) - 1; i >= 0; i-- {
		buffer := r.TransitionBuffers[i]

		// Decrement transition counter (stop at 0)
		if buffer.Transition > 0 {
			buffer.Transition--
		}

		// If transition complete, move rocks to base buffer
		if buffer.Transition <= 0 {
			// Find matching base buffer by color
			for j := range r.BaseColorBuffers {
				baseBuffer := &r.BaseColorBuffers[j]

				if colorMatch(baseBuffer.Color, buffer.Color) {
					baseBuffer.Rocks = append(baseBuffer.Rocks, buffer.Rocks...)
					break
				}
			}

			// Remove this transition buffer
			r.TransitionBuffers = append(r.TransitionBuffers[:i], r.TransitionBuffers[i+1:]...)
		}
	}
}

// performs all rock updates, wall bouncing, and collision detection/response
// diceCenters: X=centerX, Y=centerY, Z=speed (for velocity transfer to rocks)
func (r *RocksRenderer) UpdateRocksAndCollide(cursorX, cursorY float32, diceCenters []Vec3) {
	// Advance frame counter and rock type
	// r.ActiveRockBuffer++
	// if r.ActiveRockBuffer >= len(r.RockBuffers) {
	// 	r.ActiveRockBuffer = 0
	// }

	r.ActiveRockFlag = !r.ActiveRockFlag

	// rockBuffer := r.RockBuffers[r.ActiveRockBuffer]

	// Reset collision buffers to length 0 (keeps capacity - no allocation!)
	r.diceCollisionBuffer = r.diceCollisionBuffer[:0]
	r.cursorCollisionBuffer = r.cursorCollisionBuffer[:0]

	// PASS 1: BROAD PHASE - Update all rocks and collect collision candidates

	// Update base color buffers (with slowdown)
	for k := range r.BaseColorBuffers {
		r.updateBufferRocks(&r.BaseColorBuffers[k], cursorX, cursorY, diceCenters, true)
	}

	// Update held color buffers (no slowdown - maintain speed while held)
	for _, buffer := range r.HeldColorBuffers {
		r.updateBufferRocks(buffer, cursorX, cursorY, diceCenters, false)
	}

	// Update transition buffers (with slowdown)
	for _, buffer := range r.TransitionBuffers {
		r.updateBufferRocks(buffer, cursorX, cursorY, diceCenters, true)
	}

	// PASS 2: NARROW PHASE - Precise collision checks and responses
	r.handleCursorCollisions(cursorX, cursorY)
	r.handleDieCollisions(diceCenters)

	// Update all buffer transitions and move completed transition buffers to base buffers
	r.updateAllBufferTransitions()
}

// colorMatch checks if two colors are approximately equal
func colorMatch(a, b Vec3) bool {
	const epsilon = 0.01
	return abs(a.X-b.X) < epsilon &&
		abs(a.Y-b.Y) < epsilon &&
		abs(a.Z-b.Z) < epsilon
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
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

// handleDieCollisions processes die-rock collision responses using die centers
// diceCenters: X=centerX, Y=centerY, Z=speed (for velocity transfer to rocks)
// Each rock collides with at most one die per frame (first collision wins)
func (r *RocksRenderer) handleDieCollisions(diceCenters []Vec3) {
	if len(r.diceCollisionBuffer) == 0 {
		return
	}

	// OUTER LOOP: Each rock (allows break to work correctly)
	for _, rock := range r.diceCollisionBuffer {
		rockSize := rock.GetSize(r.RockTileSize)

		// Track the die with maximum total overlap (deepest penetration)
		// This prevents rocks from getting stuck between multiple dice
		var maxOverlap float32 = -1
		var bestDieIndex int = -1
		var bestDieCenter Vec3

		// PASS 1: Find die with deepest penetration
		for dieIdx, dieCenter := range diceCenters {
			// Calculate die AABB bounds
			dieLeft := dieCenter.X - HalfEffectiveDie
			dieRight := dieCenter.X + HalfEffectiveDie
			dieTop := dieCenter.Y - HalfEffectiveDie
			dieBottom := dieCenter.Y + HalfEffectiveDie

			// Calculate rock AABB bounds
			rockLeft := rock.Position.X
			rockRight := rock.Position.X + rockSize
			rockTop := rock.Position.Y
			rockBottom := rock.Position.Y + rockSize

			// NARROW PHASE: Check for actual AABB overlap
			if rockRight <= dieLeft || rockLeft >= dieRight ||
				rockBottom <= dieTop || rockTop >= dieBottom {
				continue // No collision with this die, try next die
			}

			// Calculate overlap amounts
			xOverlap := rockRight
			if dieRight < xOverlap {
				xOverlap = dieRight
			}
			if rockLeft > dieLeft {
				xOverlap -= rockLeft
			} else {
				xOverlap -= dieLeft
			}

			yOverlap := rockBottom
			if dieBottom < yOverlap {
				yOverlap = dieBottom
			}
			if rockTop > dieTop {
				yOverlap -= rockTop
			} else {
				yOverlap -= dieTop
			}

			// Calculate total overlap (deepest penetration wins)
			totalOverlap := xOverlap + yOverlap

			// Track die with maximum overlap
			if totalOverlap > maxOverlap {
				maxOverlap = totalOverlap
				bestDieIndex = dieIdx
				bestDieCenter = dieCenter
			}
		}

		// If no collision found, skip this rock
		if bestDieIndex == -1 {
			continue
		}

		// PASS 2: Process collision with the die that has deepest penetration
		dieCenter := bestDieCenter

		// Calculate die bounds for position correction
		dieLeft := dieCenter.X - HalfEffectiveDie
		dieRight := dieCenter.X + HalfEffectiveDie
		dieTop := dieCenter.Y - HalfEffectiveDie
		dieBottom := dieCenter.Y + HalfEffectiveDie

		// Calculate rock center
		rockCenterX := rock.Position.X + rockSize/2
		rockCenterY := rock.Position.Y + rockSize/2

		// Calculate collision vector (from die center to rock center)
		// This is the direction the rock should bounce
		dx := rockCenterX - dieCenter.X
		dy := rockCenterY - dieCenter.Y

		// Calculate bounce angle using atan2 (result in radians)
		bounceAngleRad := math.Atan2(float64(dy), float64(dx))
		bounceAngleDeg := bounceAngleRad * 180.0 / math.Pi

		// Normalize to 0-360 range
		if bounceAngleDeg < 0 {
			bounceAngleDeg += 360
		}

		// Calculate size-based speed boost (smaller rocks = faster)
		var sizeBoost int8
		switch rock.Score.GetScore() {
		case SmallScore: // 1 - smallest rocks
			sizeBoost = 4
		case MediumScore: // 3 - medium rocks
			sizeBoost = 3
		case BigScore: // 5 - big rocks
			sizeBoost = 2
		case HugeScore: // 10 - huge rocks
			sizeBoost = 1
		default:
			sizeBoost = 1
		}

		// Use BounceTowardsAngle to set rock direction based on collision vector
		rock.BounceTowardsAngle(int(bounceAngleDeg))

		// Add perturbation to prevent infinite bouncing loops
		// If one slope is 0, add ±1 to break symmetry
		if rock.SlopeX == 0 && rock.SlopeY != 0 {
			// Vertical bounce - add small horizontal component
			if bounceAngleDeg < 180 {
				rock.SlopeX = 1
			} else {
				rock.SlopeX = -1
			}
		} else if rock.SlopeY == 0 && rock.SlopeX != 0 {
			// Horizontal bounce - add small vertical component
			if bounceAngleDeg < 90 || bounceAngleDeg >= 270 {
				rock.SlopeY = 1
			} else {
				rock.SlopeY = -1
			}
		}

		// Apply size boost to the calculated slopes
		if rock.SlopeX > 0 {
			rock.SlopeX += sizeBoost
		} else if rock.SlopeX < 0 {
			rock.SlopeX -= sizeBoost
		}

		if rock.SlopeY > 0 {
			rock.SlopeY += sizeBoost
		} else if rock.SlopeY < 0 {
			rock.SlopeY -= sizeBoost
		}

		// Clamp slopes to valid range [MIN_SLOPE, MAX_SLOPE]
		if rock.SlopeX > MAX_SLOPE {
			rock.SlopeX = MAX_SLOPE
		} else if rock.SlopeX < MIN_SLOPE {
			rock.SlopeX = MIN_SLOPE
		}

		if rock.SlopeY > MAX_SLOPE {
			rock.SlopeY = MAX_SLOPE
		} else if rock.SlopeY < MIN_SLOPE {
			rock.SlopeY = MIN_SLOPE
		}

		// Position correction: push rock outside die bounds
		// Determine which side to push based on angle
		if bounceAngleDeg >= 315 || bounceAngleDeg < 45 {
			// Push right
			rock.Position.X = dieRight + 1
		} else if bounceAngleDeg >= 45 && bounceAngleDeg < 135 {
			// Push down
			rock.Position.Y = dieBottom + 1
		} else if bounceAngleDeg >= 135 && bounceAngleDeg < 225 {
			// Push left
			rock.Position.X = dieLeft - rockSize - 1
		} else {
			// Push up
			rock.Position.Y = dieTop - rockSize - 1
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

// countAvailableRocks returns total rocks in base color buffers
func (r *RocksRenderer) countAvailableRocks() int {
	total := 0
	for i := 0; i < len(r.config.BaseColors); i++ {
		if i < len(r.BaseColorBuffers) {
			total += len(r.BaseColorBuffers[i].Rocks)
		}
	}
	return total
}

// takeRocksFromTransitionBuffers takes rocks from transition buffers when base buffers depleted
// Prioritizes buffers with lowest Transition value (closest to completion)
func (r *RocksRenderer) takeRocksFromTransitionBuffers(needed int) []SimpleRock {
	if len(r.TransitionBuffers) == 0 || needed <= 0 {
		return []SimpleRock{}
	}

	// Sort transition buffers by Transition value (ascending - lowest first)
	slices.SortFunc(r.TransitionBuffers, func(a, b *RockBuffer) int {
		return a.Transition - b.Transition
	})

	// Take rocks from sorted buffers
	collected := make([]SimpleRock, 0, needed)
	for _, buf := range r.TransitionBuffers {
		if needed <= 0 {
			break
		}

		takeCount := needed
		if takeCount > len(buf.Rocks) {
			takeCount = len(buf.Rocks)
		}

		collected = append(collected, buf.Rocks[:takeCount]...)
		buf.Rocks = buf.Rocks[takeCount:]
		needed -= takeCount
	}

	// Clean up empty transition buffers
	r.cleanEmptyTransitionBuffers()

	return collected
}

// cleanEmptyTransitionBuffers removes transition buffers with no rocks
func (r *RocksRenderer) cleanEmptyTransitionBuffers() {
	filtered := make([]*RockBuffer, 0, len(r.TransitionBuffers))
	for _, buf := range r.TransitionBuffers {
		if len(buf.Rocks) > 0 {
			filtered = append(filtered, buf)
		}
	}
	r.TransitionBuffers = filtered
}

// countTransitionRocks returns total rocks in transition buffers
func (r *RocksRenderer) countTransitionRocks() int {
	total := 0
	for _, buf := range r.TransitionBuffers {
		total += len(buf.Rocks)
	}
	return total
}

func (r *RocksRenderer) SelectRocksColor(color Vec3, dieIdentity DieIdentity, numDice int) {
	// Calculate how many dice are NOT currently holding rocks
	numDiceNotHolding := numDice - len(r.HeldColorBuffers)
	if numDiceNotHolding <= 0 {
		return // All dice already holding rocks, nothing to do
	}

	// Calculate rocks needed - divide by dice that don't have rocks yet
	availableInBase := r.countAvailableRocks()
	rocksToTake := availableInBase / numDiceNotHolding

	if rocksToTake <= 0 {
		// Try taking from transition buffers if base is empty
		rocksToTake = r.countTransitionRocks() / numDiceNotHolding
		if rocksToTake <= 0 {
			return // No rocks available at all
		}
	}

	// Collect rocks evenly from base color buffers
	numBaseBuffers := len(r.config.BaseColors)
	if numBaseBuffers == 0 {
		return // Safety check
	}

	rocksPerBuffer := rocksToTake / numBaseBuffers
	remainder := rocksToTake % numBaseBuffers

	collectedRocks := make([]SimpleRock, 0, rocksToTake)

	// Track how many rocks taken from each base buffer for random transition color
	rockCounts := make([]int, numBaseBuffers)

	for i := 0; i < numBaseBuffers && i < len(r.BaseColorBuffers); i++ {
		buffer := &r.BaseColorBuffers[i]

		// Calculate how many to take from this buffer
		numToTake := rocksPerBuffer
		if i < remainder {
			numToTake++ // Distribute remainder rocks
		}

		// Take what's available
		actualTake := numToTake
		if actualTake > len(buffer.Rocks) {
			actualTake = len(buffer.Rocks)
		}

		if actualTake > 0 {
			collectedRocks = append(collectedRocks, buffer.Rocks[:actualTake]...)
			buffer.Rocks = buffer.Rocks[actualTake:]
			rockCounts[i] = actualTake
			numToTake -= actualTake
		}

		// If base buffer was insufficient, try transition buffers
		if numToTake > 0 {
			fromTransition := r.takeRocksFromTransitionBuffers(numToTake)
			collectedRocks = append(collectedRocks, fromTransition...)
		}
	}

	if len(collectedRocks) == 0 {
		return // No rocks collected
	}

	// Give each collected rock movement/bounce based on index
	for i := range collectedRocks {
		rock := &collectedRocks[i]

		// Convert SpriteSlopeX/Y (0..7) back to SlopeX/Y (-4..+4)
		rock.SlopeX = rock.SpriteSlopeX + MIN_SLOPE
		rock.SlopeY = rock.SpriteSlopeY + MIN_SLOPE

		// Use index to determine bounce direction (much faster than random)
		if i%2 == 0 {
			rock.BounceY()
		} else {
			rock.BounceX()
		}
	}

	// Random transition color from base colors, weighted by how many rocks taken
	transitionColor := r.config.BaseColors[0] // Default to first base color
	if len(r.config.BaseColors) > 1 {
		// Build weighted list of colors
		totalTaken := 0
		for _, count := range rockCounts {
			totalTaken += count
		}

		if totalTaken > 0 {
			// Pick random rock index
			randomIdx := rand.Intn(totalTaken)

			// Find which buffer this rock came from
			cumulative := 0
			for i, count := range rockCounts {
				cumulative += count
				if randomIdx < cumulative {
					transitionColor = r.config.BaseColors[i]
					break
				}
			}
		}
	}

	// Create held buffer
	heldBuffer := &RockBuffer{
		Rocks:           collectedRocks,
		Color:           color,           // Die color (target)
		TransitionColor: transitionColor, // Random base color
		Transition:      r.config.ColorTransitionFrames,
		FrameCounter:    0,
	}

	r.HeldColorBuffers[dieIdentity] = heldBuffer
}

func (r *RocksRenderer) Explode(num int, identity DieIdentity) {
	// get num rocks
}

func (r *RocksRenderer) DeselectAll() {
	for identity := range r.HeldColorBuffers {
		r.DeselectRocks(identity)
	}
}

func (r *RocksRenderer) DeselectRocks(dieIdentity DieIdentity) {
	// Get held buffer
	heldBuffer, exists := r.HeldColorBuffers[dieIdentity]
	if !exists || len(heldBuffer.Rocks) == 0 {
		return
	}

	numRocks := len(heldBuffer.Rocks)
	numBaseBuffers := len(r.config.BaseColors)

	if numBaseBuffers == 0 {
		// Safety: no base buffers, can't return rocks
		delete(r.HeldColorBuffers, dieIdentity)
		return
	}

	// Calculate distribution across all base buffers
	rocksPerBuffer := numRocks / numBaseBuffers
	remainder := numRocks % numBaseBuffers

	offset := 0

	// Create one transition buffer per base color buffer
	for i := 0; i < numBaseBuffers; i++ {
		// Calculate how many rocks go to this buffer
		numToReturn := rocksPerBuffer
		if i < remainder {
			numToReturn++ // Distribute remainder
		}

		if numToReturn == 0 {
			continue // Skip if no rocks for this buffer
		}

		endIdx := offset + numToReturn
		if endIdx > numRocks {
			endIdx = numRocks // Safety clamp
		}

		// Create transition buffer for this portion
		transitionBuffer := &RockBuffer{
			Rocks:           append([]SimpleRock{}, heldBuffer.Rocks[offset:endIdx]...),
			Color:           r.BaseColorBuffers[i].Color, // Target: this base color
			TransitionColor: heldBuffer.Color,            // Source: die color
			Transition:      r.config.ColorTransitionFrames,
			FrameCounter:    0,
		}

		r.TransitionBuffers = append(r.TransitionBuffers, transitionBuffer)
		offset = endIdx
	}

	// Remove held buffer
	delete(r.HeldColorBuffers, dieIdentity)
}
