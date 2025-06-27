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
	Height    float64
	ZRotation float32 // 0.0 - 1.0 uniform, final angle it lands on for a natural 'spin'
	// Theta        float64 // turning to the right opts.GeoM.Rotate(theta)
	// SpinningLeft bool    // left or right when rotating

	// ColorSpot    int // base color for spritesheet
	// IndexOnSheet int // corresponds to the Xth tile on the spritesheet
	Colliding bool // flag for collisions
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

	minX := int(math.Round(d.Vec2.X + insetAmount))
	minY := int(math.Round(d.Vec2.Y + insetAmount))
	maxX := int(math.Round(d.Vec2.X + TileSize - insetAmount))
	maxY := int(math.Round(d.Vec2.Y + TileSize - insetAmount))

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

		// pus it back to 0
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
	// Calculate centers of the dice
	// Assuming Vec2 is top-left corner and TileSize is width/height
	c1X := die1.Vec2.X + HalfTileSize
	c1Y := die1.Vec2.Y + HalfTileSize
	c2X := die2.Vec2.X + HalfTileSize
	c2Y := die2.Vec2.Y + HalfTileSize

	// Calculate distance between centers
	distCX := c1X - c2X
	distCY := c1Y - c2Y

	// Calculate minimum non-overlapping distance (sum of half-sizes)
	// This assumes dice have the same TileSize TODO: if needed: use die1.TileSize/2 + die2.TileSize/2
	minDist := TileSize

	// Calculate overlap on each axis
	overlapX := minDist - math.Abs(distCX)
	overlapY := minDist - math.Abs(distCY)

	// Resolve collision based on the axis of minimum penetration
	if overlapX > 0 && overlapY > 0 { // Check if they are actually overlapping
		// Store original velocities for clarity
		v1x, v1y := die1.Velocity.X, die1.Velocity.Y
		v2x, v2y := die2.Velocity.X, die2.Velocity.Y

		//TODO: FIXME: get rid of this e shit?
		if overlapX < overlapY {
			// Horizontal collision
			// 1D collision formulas for equal mass:
			// v1_new = (v1*(1-e) + v2*(1+e)) / 2
			// v2_new = (v1*(1+e) + v2*(1-e)) / 2
			new_v1x := (v1x + v2x) / 2.0
			new_v2x := (v1x + v2x) / 2.0

			die1.Velocity.X = new_v1x
			die2.Velocity.X = new_v2x

			// Positional correction to resolve overlap
			// Move each die by half the overlap
			correction := overlapX / 2.0
			if distCX > 0 { // die1 is to the right of die2
				die1.Vec2.X += correction
				die2.Vec2.X -= correction
			} else { // die1 is to the left of die2 (or exactly centered)
				die1.Vec2.X -= correction
				die2.Vec2.X += correction
			}

		} else {
			// Vertical collision
			// new_v1y := (v1y*(1-e) + v2y*(1+e)) / 2.0
			// new_v2y := (v1y*(1+e) + v2y*(1-e)) / 2.0
			new_v1y := (v1y + v2y) / 2.0
			new_v2y := (v1y + v2y) / 2.0

			die1.Velocity.Y = new_v1y
			die2.Velocity.Y = new_v2y

			// Positional correction
			correction := overlapY / 2.0
			if distCY > 0 { // die1 is below die2 (Y typically increases downwards)
				die1.Vec2.Y += correction
				die2.Vec2.Y -= correction
			} else { // die1 is above die2 (or exactly centered)
				die1.Vec2.Y -= correction
				die2.Vec2.Y += correction
			}
		}
	}
}

func BounceAndClamp(dice []*DieRenderable) {
	for _, die := range dice {
		if die.Vec2.X+TileSize >= ROLLZONE.MaxWidth {
			die.Vec2.X = ROLLZONE.MaxWidth - TileSize - 1
			die.Velocity.X = math.Abs(die.Velocity.X) * -1
		}
		if die.Vec2.X < ROLLZONE.MinWidth {
			die.Vec2.X = ROLLZONE.MinWidth + 1
			die.Velocity.X = math.Abs(die.Velocity.X)
		}
		if die.Vec2.Y+TileSize >= ROLLZONE.MaxHeight {
			die.Vec2.Y = ROLLZONE.MaxHeight - TileSize - 1
			die.Velocity.Y = math.Abs(die.Velocity.Y) * -1
		}
		if die.Vec2.Y < ROLLZONE.MinHeight {
			die.Vec2.Y = ROLLZONE.MinHeight + 1
			die.Velocity.Y = math.Abs(die.Velocity.Y)
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
