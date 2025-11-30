package render

// where to render die

import (
	"image"
	"math"
	"sort"
)

var (
	DampingFactor float32 = 0.7
	BounceFactor  float32 = .95
	MoveFactor    float32 = .2
)

// TODO: determine if a 'uniforms' map is better than hardcoded consts
type DieRenderable struct {
	Vec2      Vec2 // current position
	Velocity  Vec2 // traveling speed xy +-
	Fixed     Vec2 // specific coordinates
	Direction Vec2 // vec2 representation of direction the die is traveling. used in uniforms
	Color     Vec3 // direct Kage values for the color of the die
	// Modifier  float32 // used for various things
	ZRotation float32 // 0.0 - 1.0 uniform, final angle it lands on for a natural 'spin'
	Height    float32 // used for emphasizing a die during animations like scoring
	// Theta        float64 // turning to the right opts.GeoM.Rotate(theta)
	// SpinningLeft bool    // left or right when rotating

	// ColorSpot    int // base color for spritesheet
	// IndexOnSheet int // corresponds to the Xth tile on the spritesheet
	Colliding bool // flag for collisions, set to true if a collision occurs in a frame
}

// could be a persistent rect that gets it's position updated
//
// this represents the collidable area as the die shader does not take up the entire image
//
// TODO: benchmarking
// TODO: could use this same idea for rocks. would need a hardcoded constant for the inset vs recalcing each time
func (d *DieRenderable) Rect() image.Rectangle {
	// Inset each side by a small amount, e.g., 5% of DieTileSize
	// This makes the total width and height smaller by 10% of DieTileSize
	insetAmount := float32(DieTileSize * 0.15)

	minX := int(d.Vec2.X + insetAmount)
	minY := int(d.Vec2.Y + insetAmount)
	maxX := int(d.Vec2.X + DieTileSize - insetAmount)
	maxY := int(d.Vec2.Y + DieTileSize - insetAmount)

	// Ensure min is not greater than max, which can happen if DieTileSize is very small or insetAmount is too large
	if minX > maxX {
		minX = int(math.Round(float64(d.Vec2.X + HalfDieTileSize)))
		maxX = minX
	}
	if minY > maxY {
		minY = int(math.Round(float64(d.Vec2.Y + HalfDieTileSize)))
		maxY = minY
	}

	return image.Rect(minX, minY, maxX, maxY)
}

// makes sure die is moving in the correct direction.
//
// will set die velocity to 0 if under .01
func (d *DieRenderable) SetDirection() {
	dir := Vec2{}

	if math.Abs(float64(d.Velocity.X)) < .01 {
		dir.X = 0
		d.Velocity.X = 0
	} else if d.Velocity.X > 0 {
		dir.X = 1.0
	} else {
		dir.X = -1.0
	}

	if math.Abs(float64(d.Velocity.Y)) < .01 {
		dir.Y = 0
		d.Velocity.Y = 0
	} else if d.Velocity.Y > 0 {
		dir.Y = 1.0
	} else {
		dir.Y = -1.0
	}

	d.Direction = dir
}

// Moves dice to fixed pos based on num of moving dice from being selected
func HandleMovingHeldDice(dice []*DieRenderable) {
	num := len(dice)
	if num == 0 {
		return
	}

	// when a die that just became HELD it's x/y is determined on it's position from
	// where the cursor was, essentially 'slotting' it between the other dice
	sort.Slice(dice, func(i, j int) bool {
		return dice[i].Vec2.X < dice[j].Vec2.X
	})

	// positioning
	var x, y float32
	x = GAME_BOUNDS_X/2 - HalfDieTileSize
	y = SCOREZONE.MinHeight/2 + DieTileSize/5
	if num > 1 {
		x -= DieTileSize * (float32(num) - 1.0)
	}

	// find where the moving dice should be going towards
	for i := 0; i < num; i++ {
		die := dice[i]

		die.Fixed.X = x
		die.Fixed.Y = y

		x += DieTileSize * 2
	}

	for i := 0; i < num; i++ {
		die := dice[i]

		// should be a gradual slowdown in the direction
		die.Velocity.X = (die.Fixed.X - die.Vec2.X) * MoveFactor
		die.Velocity.Y = (die.Fixed.Y - die.Vec2.Y) * MoveFactor

		// puts it back to 0
		die.ZRotation *= float32(BounceFactor)

		die.Vec2.X += die.Velocity.X
		die.Vec2.Y += die.Velocity.Y
	}
}

// HandleResettingDice animates ROLLING dice that have a Fixed position set
// This is used when a die needs to animate back to a specific location (e.g., center of ROLLZONE)
func HandleResettingDice(dice []*DieRenderable) {
	for _, die := range dice {
		// Calculate velocity towards the fixed position
		die.Velocity.X = (die.Fixed.X - die.Vec2.X) * MoveFactor
		die.Velocity.Y = (die.Fixed.Y - die.Vec2.Y) * MoveFactor

		// Gradually reduce rotation back to 0
		die.ZRotation *= BounceFactor

		// Update position
		die.Vec2.X += die.Velocity.X
		die.Vec2.Y += die.Velocity.Y

		// Check if die has arrived at its destination
		if math.Abs(float64(die.Vec2.Y-die.Fixed.Y)) < 0.5 && math.Abs(float64(die.Vec2.X-die.Fixed.X)) < 0.5 {
			// Clear the Fixed position to stop resetting behavior
			die.Fixed.X = 0
			die.Fixed.Y = 0
			die.Velocity.X = 0
			die.Velocity.Y = 0
		}
	}
}

// AnimateDieToPosition sets up a die to animate towards a target center position
// centerX, centerY specify where the CENTER of the die should end up
func AnimateDieToPosition(die *DieRenderable, centerX, centerY float32) {
	// Convert center position to top-left corner (since Vec2 is top-left)
	die.Fixed.X = centerX - HalfDieTileSize
	die.Fixed.Y = centerY - HalfDieTileSize
}

// gross code
func HandleDiceCollisions(dice []*DieRenderable) {
	for i := 0; i < len(dice); i++ {
		die := dice[i]

		dieRect := die.Rect()
		for q := 0; q < len(dice); q++ {
			if i == q {
				continue
			}
			die2 := dice[q]

			die2Rect := die2.Rect()
			if dieRect.Overlaps(die2Rect) {
				BounceOffEachother(die, die2)
			}
		}
	}

	for _, die := range dice {
		die.SetDirection()
	}
}

// TODO: make this just better entirely lmao
func BounceOffEachother(die1 *DieRenderable, die2 *DieRenderable) {
	// Calculate distance and collision normal vector
	collNormalX := (die1.Vec2.X + HalfDieTileSize) - (die2.Vec2.X + HalfDieTileSize)
	collNormalY := (die1.Vec2.Y + HalfDieTileSize) - (die2.Vec2.Y + HalfDieTileSize)
	distSq := collNormalX*collNormalX + collNormalY*collNormalY

	// Check if they are actually overlapping
	if distSq < DieTileSize*DieTileSize {
		dist := float32(math.Sqrt(float64(distSq)))

		// Avoid division by zero if dice are perfectly on top of each other
		if dist == 0 {
			// Apply a small random separation
			die1.Vec2.X += 0.1
			die1.Vec2.Y += 0.1
			return
		}

		// Move dice apart so they no longer overlap
		overlap := (DieTileSize - dist) * 0.5
		correctionX := (collNormalX / dist) * overlap
		correctionY := (collNormalY / dist) * overlap
		die1.Vec2.X += correctionX
		die1.Vec2.Y += correctionY
		die2.Vec2.X -= correctionX
		die2.Vec2.Y -= correctionY

		// Find the unit normal and unit tangent vectors
		unitNormalX := collNormalX / dist
		unitNormalY := collNormalY / dist
		unitTangentX := -unitNormalY
		unitTangentY := unitNormalX

		// Project the velocity of each die onto the normal and tangent vectors
		v1t := unitTangentX*die1.Velocity.X + unitTangentY*die1.Velocity.Y
		v2t := unitTangentX*die2.Velocity.X + unitTangentY*die2.Velocity.Y

		// The tangent velocities remain the same. Swap the normal velocities.
		// This is the core of the elastic collision calculation.
		newV1n := (unitNormalX*die2.Velocity.X + unitNormalY*die2.Velocity.Y)
		newV2n := (unitNormalX*die1.Velocity.X + unitNormalY*die1.Velocity.Y)

		// Convert the scalar normal and tangent velocities back into vectors
		v1nVecX := newV1n * unitNormalX
		v1nVecY := newV1n * unitNormalY
		v1tVecX := v1t * unitTangentX
		v1tVecY := v1t * unitTangentY

		v2nVecX := newV2n * unitNormalX
		v2nVecY := newV2n * unitNormalY
		v2tVecX := v2t * unitTangentX
		v2tVecY := v2t * unitTangentY

		// Sum the normal and tangent vectors and apply bounce factor for energy loss
		die1.Velocity.X = (v1nVecX + v1tVecX) * BounceFactor
		die1.Velocity.Y = (v1nVecY + v1tVecY) * BounceFactor
		die2.Velocity.X = (v2nVecX + v2tVecX) * BounceFactor
		die2.Velocity.Y = (v2nVecY + v2tVecY) * BounceFactor
	}
}

// takes a single die and clamps it within the zone
//
// does not modify velocity, only Vec2 positioning
func ClampInZone(die *DieRenderable, zone ZoneRenderable) {
	// Handle X-axis collisions
	if die.Vec2.X+DieTileSize >= zone.MaxWidth {
		die.Vec2.X = zone.MaxWidth - DieTileSize - 1
	} else if die.Vec2.X < zone.MinWidth {
		die.Vec2.X = zone.MinWidth + 1
	}

	if die.Vec2.Y+DieTileSize >= zone.MaxHeight {
		die.Vec2.Y = zone.MaxHeight - DieTileSize - 1
	} else if die.Vec2.Y < zone.MinHeight {
		die.Vec2.Y = zone.MinWidth + 1
	}
}

func BounceAndClamp(dice []*DieRenderable) {
	for _, die := range dice {
		// Handle X-axis collisions
		if (die.Vec2.X+DieTileSize >= ROLLZONE.MaxWidth && die.Velocity.X > 0) || (die.Vec2.X < ROLLZONE.MinWidth && die.Velocity.X < 0) {
			// Correct position to be just inside the boundary
			if die.Velocity.X > 0 {
				die.Vec2.X = ROLLZONE.MaxWidth - DieTileSize - 1
			} else {
				die.Vec2.X = ROLLZONE.MinWidth + 1
			}

			// Wall normal for vertical walls is purely horizontal.
			// Reflect the velocity vector across the normal.
			die.Velocity.X *= -1 * BounceFactor
		}

		// Handle Y-axis collisions
		if (die.Vec2.Y+DieTileSize >= ROLLZONE.MaxHeight && die.Velocity.Y > 0) || (die.Vec2.Y < ROLLZONE.MinHeight && die.Velocity.Y < 0) {
			// Correct position to be just inside the boundary
			if die.Velocity.Y > 0 {
				die.Vec2.Y = ROLLZONE.MaxHeight - DieTileSize - 1
			} else {
				die.Vec2.Y = ROLLZONE.MinHeight + 1
			}

			// Wall normal for horizontal walls is purely vertical.
			// Reflect the velocity vector across the normal.
			die.Velocity.Y *= -1 * BounceFactor
		}
	}
}
