package render

// where to render die

import (
	"math"
	"math/rand/v2"
)

// TODO: determine if a 'uniforms' map is better than hardcoded consts
//
// DieRenderable is a container class for
type DieRenderable struct {
	Vec2     Vec2    // current position
	Velocity Vec2    // traveling speed xy +-
	Fixed    Vec2    // specific coordinates
	TileSize float64 // inside here saves size? unsure
	// Theta        float64 // turning to the right opts.GeoM.Rotate(theta)
	// SpinningLeft bool    // left or right when rotating

	// ColorSpot    int // base color for spritesheet
	// IndexOnSheet int // corresponds to the Xth tile on the spritesheet
	// Direction Direction
	Colliding bool // flag for collisions
}

// func (d *DieRenderable) Sprite()

var (
	DampingFactor float64 = 0.7
	BounceFactor  float64 = .95
	MoveFactor    float64 = .2
)

// Moves dice to fixed pos based on num of moving dice from being selected
func HandleMovingHeldDice(dice []*DieRenderable) {
	num := len(dice)
	if num == 0 {
		return
	}

	var x, y float64

	mod := dice[0].TileSize

	x = GAME_BOUNDS_X/2 - mod/2
	y = SCOREZONE.MaxHeight/2 - mod/2

	if num > 1 {
		x -= mod * (float64(num) - 1.0)
	}

	// find where the moving dice should be going towards
	for i := 0; i < num; i++ {
		die := dice[i]

		die.Fixed.X = x
		die.Fixed.Y = y

		x += mod * 2
	}

	for i := 0; i < num; i++ {
		die := dice[i]

		// should be a gradual slowdown in the direction
		die.Velocity.X = (die.Fixed.X - die.Vec2.X) * MoveFactor
		die.Velocity.Y = (die.Fixed.Y - die.Vec2.Y) * MoveFactor

		die.Vec2.X += die.Velocity.X
		die.Vec2.Y += die.Velocity.Y
	}

}

func HandleHeldDice(dice []*DieRenderable) {
	num := len(dice)
	if num == 0 {
		return
	}

	var x, y float64

	mod := dice[0].TileSize

	x = GAME_BOUNDS_X/2 - mod/2
	y = SCOREZONE.MaxHeight/2 - mod/2

	if num > 1 {
		x -= mod * (float64(num) - 1.0)
	}

	// go from right to left? i := len(dice)?
	for i := 0; i < num; i++ {
		die := dice[i]

		die.Vec2.X = x
		die.Vec2.Y = y

		x += mod * 2
	}
}

// gross code
func HandleDiceCollisions(dice []*DieRenderable) {
	var die, die2 *DieRenderable
	for i := 0; i < len(dice); i++ {
		die = dice[i]
		for q := 0; q < len(dice); q++ {
			if i == q {
				continue
			}
			die2 = dice[q]

			if die.Colliding {
				die.Vec2.X += die.Velocity.X
				die.Vec2.Y += die.Velocity.Y

				die.Velocity.Y *= rand.Float64() + .5
				die.Colliding = false
			}

			if die2.Colliding {
				die2.Vec2.X += die2.Velocity.X
				die2.Vec2.Y += die2.Velocity.Y

				die2.Velocity.Y *= rand.Float64() + .5
				die2.Colliding = false
			}

			xCollide := die.Vec2.X < die2.Vec2.X+die2.TileSize && die.Vec2.X > die2.Vec2.X
			yCollide := die.Vec2.Y < die2.Vec2.Y+die2.TileSize && die.Vec2.Y > die2.Vec2.Y

			if yCollide && xCollide {
				BounceOffEachother(die, die2)
			}
		}

		if math.Abs(die.Velocity.X) < .3 && math.Abs(die.Velocity.Y) < .3 {
			die.Velocity.X = 0
			die.Velocity.Y = 0
		}
	}

	for _, die := range dice {
		BounceAndClamp(die)
	}
}

func BounceOffEachother(die *DieRenderable, die2 *DieRenderable) {
	die.Colliding = true
	die2.Colliding = true

	if die.Velocity.X < BounceFactor && die.Velocity.Y < BounceFactor {
		return
	}

	factor := -1.1

	die.Velocity.X *= factor * DampingFactor
	die.Velocity.Y *= factor * DampingFactor

	if die.Velocity.X < 0 {
		die.Velocity.X += 6
	} else {
		die.Velocity.X -= 6
	}
	if die.Velocity.Y < 0 {
		die.Velocity.Y += 6
	} else {
		die.Velocity.Y -= 6
	}

	die2.Velocity.X *= factor * DampingFactor
	die2.Velocity.Y *= factor * DampingFactor

	if die2.Velocity.X < 0 {
		die2.Velocity.X += 6
	} else {
		die2.Velocity.X -= 6
	}

	if die2.Velocity.Y < 0 {
		die2.Velocity.Y += 6
	} else {
		die2.Velocity.Y -= 6
	}

}

func BounceAndClamp(die *DieRenderable) {
	if die.Vec2.X+die.TileSize >= ROLLZONE.MaxWidth {
		die.Vec2.X = ROLLZONE.MaxWidth - die.TileSize - 1
		die.Velocity.X = math.Abs(die.Velocity.X) * -1
		// die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.X < ROLLZONE.MinWidth {
		die.Vec2.X = ROLLZONE.MinWidth + 1
		die.Velocity.X = math.Abs(die.Velocity.X)
		// die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.Y+die.TileSize >= ROLLZONE.MaxHeight {
		die.Vec2.Y = ROLLZONE.MaxHeight - die.TileSize - 1
		die.Velocity.Y = math.Abs(die.Velocity.Y) * -1
		// die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.Y < ROLLZONE.MinHeight {
		die.Vec2.Y = ROLLZONE.MinHeight + 1
		die.Velocity.Y = math.Abs(die.Velocity.Y)
		// die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
}

// TODO: make this 'flick' the die based on mouse velocity?
// small hover away effect
func (d *DieRenderable) HoverFromFromFixed() {
	if d.Vec2.X > d.Fixed.X {
		d.Velocity.X = 3
	} else {
		d.Velocity.X = -3
	}
	if d.Vec2.Y > d.Fixed.Y {
		d.Velocity.Y = 3
	} else {
		d.Velocity.Y = -3
	}
}
