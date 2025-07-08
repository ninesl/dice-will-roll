package render

// where to render die

import (
	"image"
	"math"
	"sort"
)

var (
	DampingFactor float64 = 0.7
	BounceFactor  float64 = .95
	MoveFactor    float64 = .2
)

// TODO: determine if a 'uniforms' map is better than hardcoded consts
//
// DieRenderable is a container class for
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
func (d *DieRenderable) Rect() image.Rectangle {
	// Inset each side by a small amount, e.g., 5% of TileSize
	// This makes the total width and height smaller by 10% of TileSize
	insetAmount := TileSize * 0.15

	minX := int(d.Vec2.X + insetAmount)
	minY := int(d.Vec2.Y + insetAmount)
	maxX := int(d.Vec2.X + TileSize - insetAmount)
	maxY := int(d.Vec2.Y + TileSize - insetAmount)

	// Ensure min is not greater than max, which can happen if TileSize is very small or insetAmount is too large
	if minX > maxX {
		minX = int(math.Round(d.Vec2.X + HalfTileSize))
		maxX = minX
	}
	if minY > maxY {
		minY = int(math.Round(d.Vec2.Y + HalfTileSize))
		maxY = minY
	}

	return image.Rect(minX, minY, maxX, maxY)
}

// makes sure die is moving in the correct direction.
//
// will set die velocity to 0 if under .01
func (d *DieRenderable) SetDirection() {
	dir := Vec2{}

	if math.Abs(d.Velocity.X) < .01 {
		dir.X = 0
		d.Velocity.X = 0
	} else if d.Velocity.X > 0 {
		dir.X = 1.0
	} else {
		dir.X = -1.0
	}

	if math.Abs(d.Velocity.Y) < .01 {
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
	var x, y float64
	x = GAME_BOUNDS_X/2 - HalfTileSize
	y = SCOREZONE.MinHeight/2 + TileSize/5
	if num > 1 {
		x -= TileSize * (float64(num) - 1.0)
	}

	// find where the moving dice should be going towards
	for i := 0; i < num; i++ {
		die := dice[i]

		die.Fixed.X = x
		die.Fixed.Y = y

		x += TileSize * 2
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

func BounceOffEachother(die1 *DieRenderable, die2 *DieRenderable) {
	// Calculate distance and collision normal vector
	collNormalX := (die1.Vec2.X + HalfTileSize) - (die2.Vec2.X + HalfTileSize)
	collNormalY := (die1.Vec2.Y + HalfTileSize) - (die2.Vec2.Y + HalfTileSize)
	distSq := collNormalX*collNormalX + collNormalY*collNormalY

	// Check if they are actually overlapping
	if distSq < TileSize*TileSize {
		dist := math.Sqrt(distSq)

		// Avoid division by zero if dice are perfectly on top of each other
		if dist == 0 {
			// Apply a small random separation
			die1.Vec2.X += 0.1
			die1.Vec2.Y += 0.1
			return
		}

		// --- Positional Correction ---
		// Move dice apart so they no longer overlap
		overlap := (TileSize - dist) * 0.5
		correctionX := (collNormalX / dist) * overlap
		correctionY := (collNormalY / dist) * overlap
		die1.Vec2.X += correctionX
		die1.Vec2.Y += correctionY
		die2.Vec2.X -= correctionX
		die2.Vec2.Y -= correctionY

		// --- Velocity Calculation ---
		// 1. Find the unit normal and unit tangent vectors
		unitNormalX := collNormalX / dist
		unitNormalY := collNormalY / dist
		unitTangentX := -unitNormalY
		unitTangentY := unitNormalX

		// 2. Project the velocity of each die onto the normal and tangent vectors
		v1t := unitTangentX*die1.Velocity.X + unitTangentY*die1.Velocity.Y
		v2t := unitTangentX*die2.Velocity.X + unitTangentY*die2.Velocity.Y

		// 3. The tangent velocities remain the same. Swap the normal velocities.
		// This is the core of the elastic collision calculation.
		newV1n := (unitNormalX*die2.Velocity.X + unitNormalY*die2.Velocity.Y)
		newV2n := (unitNormalX*die1.Velocity.X + unitNormalY*die1.Velocity.Y)

		// 4. Convert the scalar normal and tangent velocities back into vectors
		v1nVecX := newV1n * unitNormalX
		v1nVecY := newV1n * unitNormalY
		v1tVecX := v1t * unitTangentX
		v1tVecY := v1t * unitTangentY

		v2nVecX := newV2n * unitNormalX
		v2nVecY := newV2n * unitNormalY
		v2tVecX := v2t * unitTangentX
		v2tVecY := v2t * unitTangentY

		// 5. Sum the normal and tangent vectors and apply bounce factor for energy loss
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
	if die.Vec2.X+TileSize >= zone.MaxWidth {
		die.Vec2.X = zone.MaxWidth - TileSize - 1
	} else if die.Vec2.X < zone.MinWidth {
		die.Vec2.X = zone.MinWidth + 1
	}

	if die.Vec2.Y+TileSize >= zone.MaxHeight {
		die.Vec2.Y = zone.MaxHeight - TileSize - 1
	} else if die.Vec2.Y < zone.MinHeight {
		die.Vec2.Y = zone.MinWidth + 1
	}
}

func BounceAndClamp(dice []*DieRenderable) {
	for _, die := range dice {
		// Handle X-axis collisions
		if (die.Vec2.X+TileSize >= ROLLZONE.MaxWidth && die.Velocity.X > 0) || (die.Vec2.X < ROLLZONE.MinWidth && die.Velocity.X < 0) {
			// Correct position to be just inside the boundary
			if die.Velocity.X > 0 {
				die.Vec2.X = ROLLZONE.MaxWidth - TileSize - 1
			} else {
				die.Vec2.X = ROLLZONE.MinWidth + 1
			}

			// Wall normal for vertical walls is purely horizontal.
			// Reflect the velocity vector across the normal.
			die.Velocity.X *= -1 * BounceFactor
		}

		// Handle Y-axis collisions
		if (die.Vec2.Y+TileSize >= ROLLZONE.MaxHeight && die.Velocity.Y > 0) || (die.Vec2.Y < ROLLZONE.MinHeight && die.Velocity.Y < 0) {
			// Correct position to be just inside the boundary
			if die.Velocity.Y > 0 {
				die.Vec2.Y = ROLLZONE.MaxHeight - TileSize - 1
			} else {
				die.Vec2.Y = ROLLZONE.MinHeight + 1
			}

			// Wall normal for horizontal walls is purely vertical.
			// Reflect the velocity vector across the normal.
			die.Velocity.Y *= -1 * BounceFactor
		}
	}
}

// const hoverAdjust = .1
//
// TODO: make this 'flick' the die based on mouse velocity?
// small hover away effect
// func (d *DieRenderable) HoverFromFromFixed() {
// if d.Vec2.X > d.Fixed.X {
// d.Velocity.X = hoverAdjust
// } else {
// d.Velocity.X = -hoverAdjust
// }
// if d.Vec2.Y > d.Fixed.Y {
// d.Velocity.Y = hoverAdjust
// } else {
// d.Velocity.Y = -hoverAdjust
// }
// }
//
