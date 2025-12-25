package rocks

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
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

	// Rock physics constants
	MIN_SEPARATION           = 5 // Distance to push rocks away from dice after collision
	ROCK_JITTER              = 2
	ROCK_DAMPING_FRAME_CYCLE = 20 // How often rocks slow down (every N frames)
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

	// Medium rock size multipliers
	MediumLargeSize  = 0.50
	MediumMediumSize = 0.45
	MediumSmallSize  = 0.40

	// Big rock size multipliers
	BigLargeSize  = 0.80
	BigMediumSize = 0.75
	BigSmallSize  = 0.70

	// Huge rock size multipliers
	HugeLargeSize  = 1.20
	HugeMediumSize = 1.10
	HugeSmallSize  = 1.00
)

// RockSizeData holds pre-computed size values for a rock type
// This eliminates repeated multiplications in hot collision paths
type RockSizeData struct {
	Size          float32 // Full size (e.g., 64px)
	HalfSize      float32 // Size / 2 (for center calculations)
	EffectiveSize float32 // Size * 0.75 (for collision)
	HalfEffective float32 // EffectiveSize / 2
	Inset         float32 // (Size - EffectiveSize) / 2
}

// Global lookup table, indexed by RockScoreType (1-12)
// Initialized once at startup in initRockSizeLookup()
var rockSizeLookup [MaxRockType]RockSizeData

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

// SimpleRock represents a rock using pre-extracted sprite frames
type SimpleRock struct {
	Position    render.Vec2 // 2D screen position
	SpriteIndex uint8       // Current rotation frame index (0-71)
	SlopeX      int8        // Current X speed component (-4 to +4)
	SlopeY      int8        // Current Y speed component (-4 to +4)

	// Transition system for smooth sprite rotation during direction changes
	SpriteSlopeX int8          // Visual speed X used during transition (gradually moves toward SpeedX)
	SpriteSlopeY int8          // Visual speed Y used during transition (gradually moves toward SpeedY)
	Score        RockScoreType //  how many 'rocks' this rock counts for during scoring. also determines size, etc

	transitionSteps uint8 // Bit-packed: lower 4 bits = X remaining steps, upper 4 bits = Y remaining steps
}

// const BaseVelocity = 1.0

const BaseVelocity = 1.0

type RockBuffer struct {
	Rocks           []SimpleRock
	TransitionColor render.Vec3
	Color           render.Vec3
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
	sprites [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*render.Sprite

	// Three-tier buffer system for rock color management
	BaseColorBuffers  []RockBuffer                       // Source rocks (Grey, Brown, etc.)
	HeldColorBuffers  map[render.DieIdentity]*RockBuffer // Rocks owned by held dice
	TransitionBuffers []*RockBuffer                      // Rocks transitioning back to base colors

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
	imagePool *render.ImagePool

	config RocksConfig
}

// RocksConfig holds configuration for rock system
type RocksConfig struct {
	TotalRocks            int
	BaseColors            []render.Vec3 // the colors that rock render applies
	RockTileSize          float32       // Base tile size for rock rendering and collision calculations
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
	r.HeldColorBuffers = make(map[render.DieIdentity]*RockBuffer)
	r.TransitionBuffers = make([]*RockBuffer, 0, 10)

	// Initialize rock size lookup table (pre-compute all size calculations)
	r.initRockSizeLookup()

	// Generate and pre-extract all sprite frames (single grayscale spritesheet)
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	// Initialize image pool for temporary rendering (lazy allocation)
	r.imagePool = render.NewImagePool(int(config.WorldBoundsX), int(config.WorldBoundsY))

	return r
}

// generateSprites creates a single grayscale spritesheet array (shared by all rock types)
// Colors will be applied at draw-time via the color filter shader
func (r *RocksRenderer) generateSprites() {
	genSprites := [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*render.Sprite{}

	// Use white values for base sprite (will be colored via shader later)
	// Different white tones create visual variety in the base geometry
	innerDark := render.WhiteDark    // Darker white for inner/crater areas
	innerLight := render.WhiteMid    // Medium white for inner/crater areas
	outerDark := render.WhiteLight   // Light white for outer surface
	outerLight := render.WhiteBright // Pure white for outer surface highlights

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
					"Mouse":           render.Vec2{X: 0.0, Y: 0.0}.KageVec2(),
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
			sprite := render.Sprite{
				Image:       spriteSheet,
				SpriteSheet: render.NewSpriteSheet(SHEET_COLS, SHEET_ROWS, spriteSize),
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
		pos := render.Vec2{
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
	for i := range len(render.RainbowColors) {
		buffer, ok := r.HeldColorBuffers[render.DieIdentity(i)]
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

// initRockSizeLookup pre-computes all rock size values
// Called once in NewRocksRenderer to eliminate runtime calculations
func (r *RocksRenderer) initRockSizeLookup() {
	for scoreType := RockScoreType(1); scoreType < MaxRockType; scoreType++ {
		size := r.RockTileSize * scoreType.SizeMultiplier()
		effectiveSize := size * 0.75

		rockSizeLookup[scoreType] = RockSizeData{
			Size:          size,
			HalfSize:      size / 2,
			EffectiveSize: effectiveSize,
			HalfEffective: effectiveSize / 2,
			Inset:         (size - effectiveSize) / 2,
		}
	}
}

func (r *RocksRenderer) Explode(num int, identity render.DieIdentity) {
	// get num rocks
}
