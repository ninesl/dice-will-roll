package rocks

import (
	"fmt"
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

type RockID uint16

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
	frameCount      uint8 // this could be used for the timer? independent animations
}

const BaseVelocity = 1.0

type RockBuffer struct {
	RockIDs         []RockID
	Transition      int
	TransitionColor render.Vec3
	Color           render.Vec3
	//FrameCounter    int
}

// RocksRenderer manages pre-extracted sprite rendering with ultra-fast array indexing
type RocksRenderer struct {
	shader          *ebiten.Shader
	colorShader     *ebiten.Shader // Color filter shader for applying colors to grayscale sprites
	explosionShader *ebiten.Shader

	// 2D array of grayscale sprite sheets (shared by all rock types)
	// [speedX_index][speedY_index] -> Sprite struct containing spritesheet
	// Each spritesheet has ROTATION_FRAMES (72) frames arranged in 18 columns x 4 rows
	// Size: [8][8] = 64 spritesheets total (was 128 with NUM_ROCK_TYPES before!)
	sprites [DIRECTIONS_TO_SNAP][DIRECTIONS_TO_SNAP]*render.Sprite

	totalRocks []int          // rock depth nums 0 is activeBaseBuffer
	Rocks      [][]SimpleRock //technically max is uint16 65535

	// Three-tier buffer system for rock color management
	BaseColorBuffers    []RockBuffer // Source rocks (Grey, Brown, etc.)
	ActiveBaseBufferIdx int          // which index of rock buffer/associated active base buffer

	HeldColorBuffers  map[render.DieIdentity]RockBuffer // Rocks owned by held dice
	TransitionBuffers []RockBuffer                      // Rocks transitioning back to base colors
	ExplosionBuffers  map[render.DieIdentity]RockBuffer // Rocks currently exploding
	selectionOrder    []render.DieIdentity              // Tracks order dice were selected (for draw order)

	pendingExplosionBatches  []pendingExplosionBatch
	pendingExplosionBatchIdx int
	deselectAfterExplosions  bool

	// collection during updates
	updatingBuffers []RockBuffer

	RockTileSize float32 // Base tile size for rock rendering and collision calculations

	// Internal collision buffers - reused each frame to avoid allocations
	diceCollisionBuffer           []RockID
	cursorCollisionBuffer         []RockID
	diceCollisionDieIndexesBuffer []int
	diceCollisionDataBuffer       []dieCollisionData

	// Image pool for temporary rendering buffers (lazily allocated and reused every frame)
	imagePool *render.ImagePool

	// Spatial grid for L1-cache-friendly collision detection (hybrid offset+count)
	// All rock indices packed contiguously, grouped by cell
	gridCellSize float32  // Size of each grid cell (DieTileSize)
	gridCols     int      // Number of grid columns
	gridRows     int      // Number of grid rows
	gridRocks    []RockID // Flat array of rock IDs in grid order
	gridOffsets  []uint16 // Start index in gridRocks for each cell
	gridCounts   []uint16 // Number of rocks in each cell

	config RocksConfig
}

type pendingExplosionBatch struct {
	dieIdentity render.DieIdentity
	rockIDs     []RockID
}

// RocksConfig holds configuration for rock system
type RocksConfig struct {
	TotalRocks            []int         // 0-x index of depth. 0 is first base layer that is active
	BaseColors            []render.Vec3 // the colors that rock render applies
	RockTileSize          float32       // Base tile size for rock rendering and collision calculations
	WorldBoundsX          float32
	WorldBoundsY          float32
	ColorTransitionFrames int // frames for color transitions (default: 30)
}

// helper func for active buffers, wrapper allows us to stack trace and reuse in the future
func (r *RocksRenderer) AssignActiveBuffer(specificIdx int) {
	if specificIdx < 0 || specificIdx >= len(r.BaseColorBuffers) { // forced error fallback
		panic(fmt.Errorf("%d is not legal for the base colors that are left", specificIdx))
	}
	r.ActiveBaseBufferIdx = specificIdx
}

// NewRocksRenderer creates a sprite rendering system for
func NewRocksRenderer(config RocksConfig) *RocksRenderer {
	shaderMap := shaders.LoadShaders()

	r := &RocksRenderer{
		shader:          shaderMap[shaders.RocksShaderKey],
		colorShader:     shaderMap[shaders.ColorFilterShaderKey],
		explosionShader: shaderMap[shaders.ColorShaderKey],
		RockTileSize:    config.RockTileSize,
		totalRocks:      config.TotalRocks,
		Rocks:           make([][]SimpleRock, 0, len(config.TotalRocks)),
		// Initialize collision check radii - TIGHT buffers to reduce expensive collision calculations
		// Accepts that some edge-case collisions at buffer boundaries may be missed

		// Pre-allocate collision buffers with typical capacity to avoid allocations
		// Capacity based on typical collision counts: ~50 dice collisions, ~20 cursor collisions
		diceCollisionBuffer:           make([]RockID, 0, 128),
		diceCollisionDieIndexesBuffer: make([]int, 0, 128),
		cursorCollisionBuffer:         make([]RockID, 0, 128),

		diceCollisionDataBuffer: make([]dieCollisionData, 0, 7), //TODO:constant
		config:                  config,
		// Define colors for each rock type (applied via shader at draw time)
	}

	// Initialize rock size lookup table (pre-compute all size calculations)
	r.initRockSizeLookup()

	// Generate and pre-extract all sprite frames (single grayscale spritesheet)
	r.generateSprites()

	// Generate rock instances
	r.generateRocks(config)

	// Initialize image pool for temporary rendering (lazy allocation)
	r.imagePool = render.NewImagePool(int(config.WorldBoundsX), int(config.WorldBoundsY))

	// Initialize spatial grid for L1-cache-friendly collision detection
	r.initSpatialGrid(config)

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

			genSprites[XSnapIdx][YSnapIdx] = &sprite
		}
	}

	// Assign generated sprites to the renderer
	r.sprites = genSprites
}

// generateRocks creates rock instances with random RockScoreTypes that accumulate to target score
func (r *RocksRenderer) generateRocks(config RocksConfig) {
	// r.ActiveBaseBufferIdx is 0, only have implementation for 1
	r.Rocks = append(r.Rocks, make([]SimpleRock, 0, config.TotalRocks[r.ActiveBaseBufferIdx]))

	// Initialize empty rock buffers slice (will grow dynamically)
	r.BaseColorBuffers = make([]RockBuffer, 0, len(config.BaseColors))
	r.HeldColorBuffers = make(map[render.DieIdentity]RockBuffer)
	r.ExplosionBuffers = make(map[render.DieIdentity]RockBuffer)
	r.TransitionBuffers = make([]RockBuffer, 0, 10)
	r.selectionOrder = make([]render.DieIdentity, 0, 7) // Track selection order for draw order
	r.updatingBuffers = make([]RockBuffer, 0)

	// targetScore := config.TotalRocks // e.g., 500
	//
	// // Track rocks by type
	// var allRocks = make([][]SimpleRock, 0, len(config.BaseColors))
	// for range len(config.BaseColors) {
	// 	allRocks = append(allRocks, []SimpleRock{})
	// }
	//
	// //var curRockTypeIndex int
	//
	// // make each layer using base Colors
	// // currently will only make enough for base layer 0
	// for _, layerScore := range config.TotalRocks {
	// 	currentScore := 0
	// 	// Generate rocks until we reach target score

	baseColor := render.Vec3{}
	if len(config.BaseColors) > r.ActiveBaseBufferIdx {
		baseColor = config.BaseColors[r.ActiveBaseBufferIdx]
	}

	r.BaseColorBuffers = append(r.BaseColorBuffers, RockBuffer{
		RockIDs:         make([]RockID, 0, config.TotalRocks[r.ActiveBaseBufferIdx]),
		Color:           baseColor,
		TransitionColor: baseColor,
		Transition:      0,
	})
	remaining := config.TotalRocks[r.ActiveBaseBufferIdx]
	var rockIDx RockID = 0

	for remaining > 0 {

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
			SlopeX:       slopeX,
			SlopeY:       slopeY,
			SpriteSlopeX: spriteSlopeX,
			SpriteSlopeY: spriteSlopeY,
			Score:        scoreType,
		}
		r.Rocks[r.ActiveBaseBufferIdx] = append(r.Rocks[r.ActiveBaseBufferIdx], rock)
		r.BaseColorBuffers[r.ActiveBaseBufferIdx].RockIDs = append(r.BaseColorBuffers[r.ActiveBaseBufferIdx].RockIDs, rockIDx)
		rockIDx++
		remaining -= scoreType.GetScore()
	}

	// 		allRocks[curRockTypeIndex] = append(allRocks[curRockTypeIndex], rock)
	// 		currentScore += scoreType.GetScore()
	// 		curRockTypeIndex++
	// 		if curRockTypeIndex >= len(config.BaseColors) {
	// 			curRockTypeIndex = 0
	// 		}
	// 	}
	// 	r.BaseColorBuffers = append(r.BaseColorBuffers, RockBuffer{
	// 		Color:           config.BaseColors[i],
	// 		TransitionColor: config.BaseColors[i],
	// 		Transition:      0,
	// 	})
	// 	r.BaseColorBuffers[i].RockIDs = make([]RockID, 0, len())
	// }
	//
	// for i := range len(allRocks) {
	// 	r.BaseColorBuffers = append(r.BaseColorBuffers, RockBuffer{
	// 		Color:           config.BaseColors[i],
	// 		TransitionColor: config.BaseColors[i],
	// 		Transition:      0,
	// 	})
	// 	r.BaseColorBuffers[i].RockIDs = make([]RockID, 0, len())
	// }
}

// DrawRocks renders all rocks with fast direct array access
// Uses grayscale sprites with color filter shader applied per rock type
// Implements index-based interleaving to prevent one color from always appearing on top
func (r *RocksRenderer) DrawRocks(screen *ebiten.Image) {
	// Reset image pool for this frame
	r.imagePool.Reset()

	// Reuse DrawImageOptions to avoid allocations (important for 10k+ rocks)
	opts := &ebiten.DrawImageOptions{}

	// Render base color buffers with interleaving
	// for layer := 0; layer < NUM_INTERLEAVE_LAYERS; layer++ {
	for rockType := range len(r.BaseColorBuffers) {
		buffer := r.BaseColorBuffers[rockType]

		// Get next temp image from pool (already cleared)
		tempImg := r.imagePool.GetNext()

		// Draw rocks with interleaving
		for _, rockID := range buffer.RockIDs {
			rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]
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
		r.drawBufferWithColorShader(buffer, tempImg, screen)
	}
	// }

	// Render transition buffers
	for _, buffer := range r.TransitionBuffers {
		tempImg := r.imagePool.GetNext()
		r.drawBufferToImage(buffer, tempImg, opts)
		r.drawBufferWithColorShader(buffer, tempImg, screen)
	}
	// Render held color buffers in SELECTION ORDER (most recent = on top)
	for _, dieIdentity := range r.selectionOrder {
		buffer := r.HeldColorBuffers[dieIdentity]
		if len(buffer.RockIDs) == 0 {
			continue
		}

		tempImg := r.imagePool.GetNext()
		r.drawBufferToImage(buffer, tempImg, opts)
		r.drawBufferWithColorShader(buffer, tempImg, screen)
	}

	// Draw explosions on top of everything
	r.DrawExplosions(screen)
}

// SplitRock converts one exploding rock into smaller child rocks.
// The children start centered on the parent, then bounce outward in a radial pattern.
// Rocks that are already SmallScore do not split further and return a nil slice
func (r *RocksRenderer) SplitRock(rockID RockID) []RockID {
	rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]
	//TODO: make the rocks made spin in a specific way based on input rock
	childScores := splitRockScores(rock.Score.GetScore())
	if len(childScores) == 0 {
		return nil
	}

	parentSize := rock.SizeData()
	center := render.Vec2{
		X: rock.Position.X + parentSize.HalfSize,
		Y: rock.Position.Y + parentSize.HalfSize,
	}

	children := make([]SimpleRock, 0, len(childScores))
	for i, score := range childScores {
		childType := randomRockTypeForScore(score)
		childSize := rockSizeLookup[childType]
		child := SimpleRock{
			Position: render.Vec2{
				X: center.X - childSize.HalfSize,
				Y: center.Y - childSize.HalfSize,
			},
			SpriteIndex:  rock.SpriteIndex,
			SpriteSlopeX: rock.SpriteSlopeX,
			SpriteSlopeY: rock.SpriteSlopeY,
			Score:        childType,
		}

		angle := i * 360 / len(childScores)
		child.BounceTowardsAngle(angle)
		children = append(children, child)
	}

	rockIDs := make([]RockID, len(children))
	for i := range len(children) {
		rockIDs[i] = RockID(len(r.Rocks[r.ActiveBaseBufferIdx]) + i)
	}

	r.Rocks[r.ActiveBaseBufferIdx] = append(r.Rocks[r.ActiveBaseBufferIdx], children...)
	return rockIDs
}

// splitRockScores defines the gameplay value breakdown for exploded rocks.
// Each explosion consumes one score; the remaining score becomes child rocks.
func splitRockScores(score int) []int {
	switch score {
	case HugeScore:
		return []int{BigScore, MediumScore, SmallScore}
	case BigScore:
		return []int{MediumScore, SmallScore}
	case MediumScore:
		return []int{SmallScore, SmallScore}
	default:
		return nil
	}
}

// randomRockTypeForScore chooses a visual size variant for a logical score value.
// The score controls gameplay value, while the returned RockScoreType adds size variety.
func randomRockTypeForScore(score int) RockScoreType {
	switch score {
	case SmallScore:
		return SmallLarge + RockScoreType(rand.Intn(rockScoreVariants))
	case MediumScore:
		return MediumLarge + RockScoreType(rand.Intn(rockScoreVariants))
	case BigScore:
		return BigLarge + RockScoreType(rand.Intn(rockScoreVariants))
	case HugeScore:
		return HugeLarge + RockScoreType(rand.Intn(rockScoreVariants))
	default:
		panic(fmt.Errorf("invalid rock score %d", score))
	}
}

// ExplodeRocks batches up to numRocks rocks into the initiating die's color.
// The visual explosion starts on a later frame via UpdatePendingExplosionBatches.
func (r *RocksRenderer) ExplodeRocks(dieIdentity render.DieIdentity, numRocks int) {
	if numRocks <= 0 {
		return
	}

	rockIDs := r.popRocksForExplosion(dieIdentity, numRocks)
	if len(rockIDs) == 0 {
		return
	}

	r.pendingExplosionBatches = append(r.pendingExplosionBatches, pendingExplosionBatch{
		dieIdentity: dieIdentity,
		rockIDs:     rockIDs,
	})
}

// UpdatePendingExplosionBatches starts at most one queued explosion batch.
// Call this before same-frame scoring code can queue new batches, so batching and
// visual explosion never happen in the same frame.
func (r *RocksRenderer) UpdatePendingExplosionBatches() {
	if r.pendingExplosionBatchIdx >= len(r.pendingExplosionBatches) {
		if len(r.pendingExplosionBatches) > 0 {
			r.pendingExplosionBatches = r.pendingExplosionBatches[:0]
			r.pendingExplosionBatchIdx = 0
			if r.deselectAfterExplosions {
				r.DeselectAll()
				r.deselectAfterExplosions = false
			}
		}
		return
	}

	batch := r.pendingExplosionBatches[r.pendingExplosionBatchIdx]
	r.pendingExplosionBatchIdx++

	r.addExplosionRocks(batch.dieIdentity, batch.rockIDs)

	newRockIDs := r.SplitRocks(batch.rockIDs)
	if len(newRockIDs) > 0 {
		ownerBuffer := r.ensureHeldBuffer(batch.dieIdentity)
		ownerBuffer.RockIDs = append(ownerBuffer.RockIDs, newRockIDs...)
		r.HeldColorBuffers[batch.dieIdentity] = ownerBuffer
	}

	if r.pendingExplosionBatchIdx >= len(r.pendingExplosionBatches) {
		r.pendingExplosionBatches = r.pendingExplosionBatches[:0]
		r.pendingExplosionBatchIdx = 0
		if r.deselectAfterExplosions {
			r.DeselectAll()
			r.deselectAfterExplosions = false
		}
	}
}

func (r *RocksRenderer) DeselectAllAfterPendingExplosions() {
	if len(r.pendingExplosionBatches) == 0 {
		r.DeselectAll()
		return
	}
	r.deselectAfterExplosions = true
}

func takeRockIDs(buffer *RockBuffer, needed int, collected []RockID) (int, []RockID) {
	if needed <= 0 || len(buffer.RockIDs) == 0 {
		return needed, collected
	}

	take := needed
	if take > len(buffer.RockIDs) {
		take = len(buffer.RockIDs)
	}

	collected = append(collected, buffer.RockIDs[:take]...)
	buffer.RockIDs = buffer.RockIDs[take:]
	return needed - take, collected
}

// popRocksForExplosion removes rocks that should explode from the front of their rock buffers.
//
// Selection order is intentional:
// 1. consume rocks already held by the initiating die,
// 2. consume active base rocks if the die runs out,
// 3. consume rocks from other held buffers as a final fallback.
//
// Fallback rocks still visually explode into the initiating color and split children
// re-enter that same held buffer, producing a confetti-like color conversion.
func (r *RocksRenderer) popRocksForExplosion(preferredIdentity render.DieIdentity, needed int) []RockID {
	collected := make([]RockID, 0, needed)

	if buffer := r.HeldColorBuffers[preferredIdentity]; len(buffer.RockIDs) > 0 {
		needed, collected = takeRockIDs(&buffer, needed, collected)
		r.HeldColorBuffers[preferredIdentity] = buffer
	}

	if needed > 0 {
		buffer := r.BaseColorBuffers[r.ActiveBaseBufferIdx]
		needed, collected = takeRockIDs(&buffer, needed, collected)
		r.BaseColorBuffers[r.ActiveBaseBufferIdx] = buffer
	}

	for i := range r.TransitionBuffers {
		if needed <= 0 {
			break
		}
		needed, collected = takeRockIDs(&r.TransitionBuffers[i], needed, collected)
	}

	for otherIdentity, buffer := range r.HeldColorBuffers {
		if needed <= 0 {
			break
		}
		if otherIdentity == preferredIdentity {
			continue
		}
		needed, collected = takeRockIDs(&buffer, needed, collected)
		r.HeldColorBuffers[otherIdentity] = buffer
	}

	return collected
}

func (r *RocksRenderer) ensureHeldBuffer(dieIdentity render.DieIdentity) RockBuffer {
	buffer := r.HeldColorBuffers[dieIdentity]
	if buffer.RockIDs == nil {
		buffer.Color = render.RainbowColors[dieIdentity]
		buffer.RockIDs = make([]RockID, 0)
		r.HeldColorBuffers[dieIdentity] = buffer
	}
	return buffer
}

func (r *RocksRenderer) clearHeldBuffer(dieIdentity render.DieIdentity) {
	buffer := r.ensureHeldBuffer(dieIdentity)
	buffer.RockIDs = buffer.RockIDs[:0]
	r.HeldColorBuffers[dieIdentity] = buffer
}

func (r *RocksRenderer) SplitRocks(rockIDs []RockID) []RockID {
	newRockIDs := make([]RockID, 0, len(rockIDs)*2)
	for _, rockID := range rockIDs {
		newRockIDs = append(newRockIDs, r.SplitRock(rockID)...)
	}
	return newRockIDs
}

// addExplosionRocks moves rocks into the transient explosion buffer for a die identity.
// Explosion buffers are separate from held buffers so the disappearing parent rock can
// animate while any split children immediately continue moving in the held color buffer.
func (r *RocksRenderer) addExplosionRocks(dieIdentity render.DieIdentity, rockIDs []RockID) {
	if len(rockIDs) == 0 {
		return
	}

	explosionBuffer, exists := r.ExplosionBuffers[dieIdentity]
	if !exists {
		explosionBuffer = RockBuffer{
			RockIDs: make([]RockID, 0, len(rockIDs)),
			Color:   render.RainbowColors[dieIdentity],
		}
		if heldBuffer := r.HeldColorBuffers[dieIdentity]; len(heldBuffer.RockIDs) > 0 {
			explosionBuffer.Color = heldBuffer.Color
		}
	}

	for _, rockID := range rockIDs {
		r.Rocks[r.ActiveBaseBufferIdx][rockID].SpriteIndex = 60 // TODO: constant or modify based on what explode anim shader is going to be applied
	}
	explosionBuffer.RockIDs = append(explosionBuffer.RockIDs, rockIDs...)
	r.ExplosionBuffers[dieIdentity] = explosionBuffer
}

// updateExplosions updates all exploding rocks, decrementing their frame counters
func (r *RocksRenderer) updateExplosions() {
	for dieIdentity, buffer := range r.ExplosionBuffers {
		// Process rocks in reverse order for safe removal
		for i := len(buffer.RockIDs) - 1; i >= 0; i-- {
			rockID := buffer.RockIDs[i]
			rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]

			// Decrement explosion frame (stored in SpriteIndex)
			if rock.SpriteIndex > 0 {
				rock.SpriteIndex--
			}

			// Remove finished explosions
			if rock.SpriteIndex == 0 {
				buffer.RockIDs[i] = buffer.RockIDs[len(buffer.RockIDs)-1]
				buffer.RockIDs = buffer.RockIDs[:len(buffer.RockIDs)-1]
			}
		}
		r.ExplosionBuffers[dieIdentity] = buffer
	}
}

// DrawExplosions renders all exploding rocks with the explosion shader
func (r *RocksRenderer) DrawExplosions(screen *ebiten.Image) {
	for _, buffer := range r.ExplosionBuffers {
		if len(buffer.RockIDs) == 0 {
			continue
		}

		// Draw each exploding rock
		for _, rockID := range buffer.RockIDs {
			// Get size from rock score for shader
			rock := r.Rocks[r.ActiveBaseBufferIdx][rockID]
			sizeData := rock.SizeData()

			// Get temp image from pool for this explosion
			tempImg := r.imagePool.GetNext()

			// TODO: Draw rectangle with explosion shader
			// shaderOpts := &ebiten.DrawRectShaderOptions{
			//     Uniforms: map[string]interface{}{
			//         "Color": buffer.Color.KageVec3(),
			//         "Time": float32(30-rock.SpriteIndex) / 30.0,
			//         "Resolution": []float32{float32(sizeData.Size), float32(sizeData.Size)},
			//     },
			// }
			// tempImg.DrawRectShader(int(sizeData.Size), int(sizeData.Size), r.explosionShader, shaderOpts)

			// Test: Draw explosion shader with color uniform to verify effect triggers
			// fmt.Printf("Explosion color: RGB(%.3f, %.3f, %.3f)\n", colorVec[0], colorVec[1], colorVec[2])
			shaderOpts := &ebiten.DrawRectShaderOptions{
				Uniforms: map[string]interface{}{
					"Color": buffer.Color.KageVec3(),
				},
			}
			tempImg.DrawRectShader(int(sizeData.Size), int(sizeData.Size), r.explosionShader, shaderOpts)

			// Draw temp image to screen at rock position
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
			screen.DrawImage(tempImg, opts)
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

	// TODO:
	// FIXME:
	// we want varying behaviors for rock renderer based on scale
	// and len(rocks) left and numRocks [the score]
	// grab all small rocks, grab X small rocks and Y big rocks
	// grab rockso nly X fast, lots of variety here?

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

// UpdateAnimation handles the smooth sprite rotation during direction changes
// Now includes full rotation on each bounce
// Modifies the Transition
func (r *SimpleRock) UpdateAnimation() {
	if r.SlopeX != 0 || r.SlopeY != 0 {
		r.frameCount++
	}

	// FIXME: we need to have framecount adjusted here somehow the % isn't ideal

	// Update SpriteIndex based on rock SIZE (smaller rocks rotate faster)
	if int(r.frameCount)%r.Score.GetScore() == 0 {
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

	if int(r.frameCount)%n != 0 {
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
