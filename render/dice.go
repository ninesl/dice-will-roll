package render

import (
	"math"
	"math/rand/v2"
)

type DieRenderable struct {
	Vec2     Vec2    // current position
	Velocity Vec2    // traveling speed xy +-
	Fixed    Vec2    // specific coordinates
	TileSize float64 // inside here saves size? unsure

	ColorSpot    int // base color for spritesheet
	IndexOnSheet int // corresponds to the Xth tile on the spritesheet
	// Direction Direction
	Colliding bool // flag for collisions
}

// func (d *DieRenderable) Sprite()

var (
	DiceBottom    float64
	DampingFactor float64 = 0.7
	BounceFactor  float64 = .9
)

// gross code
func HandleDiceCollisions(dice []*DieRenderable) {
	for i, die := range dice {
		for q, die2 := range dice {
			if i == q {
				continue
			}

			if die.Colliding {
				die.Vec2.X += die.Velocity.X
				die.Vec2.Y += die.Velocity.Y

				die.Velocity.Y *= rand.Float64() + .2
				die.Colliding = false
			}

			if die2.Colliding {
				die2.Vec2.X += die2.Velocity.X
				die2.Vec2.Y += die2.Velocity.Y

				die2.Velocity.Y *= rand.Float64() + .2
				die2.Colliding = false
			}

			xCollide := die.Vec2.X < die2.Vec2.X+die2.TileSize && die.Vec2.X > die2.Vec2.X
			yCollide := die.Vec2.Y < die2.Vec2.Y+die2.TileSize && die.Vec2.Y > die2.Vec2.Y

			if yCollide && xCollide {
				die.Colliding = true
				die2.Colliding = true

				// die.Velocity.Y *= rand.Float64() + 1
				// die2.Velocity.Y *= rand.Float64() + 1

				BounceOffEachother(die, die2)

				// die.Velocity.X *= -BounceFactor
				// die2.Velocity.X *= -BounceFactor
				// die.Velocity.Y *= -BounceFactor
				// die2.Velocity.Y *= -BounceFactor
			}
		}

		if math.Abs(die.Velocity.X) < .3 && math.Abs(die.Velocity.Y) < .3 {
			die.Velocity.X = 0
			die.Velocity.Y = 0
		}
	}
}

func BounceOffEachother(die *DieRenderable, die2 *DieRenderable) {
	if die.Velocity.X < BounceFactor && die.Velocity.Y < BounceFactor {
		return
	}

	factor := -1.1

	die.Velocity.X *= factor * DampingFactor
	die.Velocity.Y *= factor * DampingFactor

	if die.Velocity.X < 0 {
		die.Velocity.X += 1
	} else {
		die.Velocity.X -= 1
	}
	if die.Velocity.Y < 0 {
		die.Velocity.Y += 1
	} else {
		die.Velocity.Y -= 1
	}

	die2.Velocity.X *= factor * DampingFactor
	die2.Velocity.Y *= factor * DampingFactor

	if die2.Velocity.X < 0 {
		die2.Velocity.X += 10
	} else {
		die2.Velocity.X -= 10
	}
	if die2.Velocity.Y < 0 {
		die2.Velocity.Y += 10
	} else {
		die2.Velocity.Y -= 10
	}

}

// Update handles the movement and bouncing of the DieSprite under gravity.
func UpdateDie(d *DieRenderable) {
	var hitLeft, hitRight, hitTop, hitBottom bool

	hitTop = d.Vec2.Y < MinHeight
	hitBottom = d.Vec2.Y+d.TileSize >= MaxHeight

	hitLeft = d.Vec2.X < MinWidth
	hitRight = d.Vec2.X+d.TileSize >= MaxWidth

	if hitLeft || hitRight {
		d.Velocity.X *= -1.0 * BounceFactor

		if hitLeft {
			d.Vec2.X = MinWidth
		} else {
			d.Vec2.X = MaxWidth - d.TileSize
		}
		// d.IndexOnSheet = d.ColorSpot + rand.IntN(5)
	}

	if hitTop || hitBottom {
		d.Velocity.Y *= -1.0 * BounceFactor //* DampingFactor

		if hitTop {
			d.Vec2.Y = MinHeight
		} else {
			d.Vec2.Y = MaxHeight - d.TileSize
		}
		// d.IndexOnSheet = rand.IntN(5)
	}

	d.Velocity.X *= .95
	d.Velocity.Y *= .95

	d.Vec2.X += d.Velocity.X
	d.Vec2.Y += d.Velocity.Y
}

func BounceAndClamp(die *DieRenderable) {
	if die.Vec2.X+die.TileSize >= MaxWidth {
		die.Vec2.X = MaxWidth - die.TileSize - 1
		die.Velocity.X = math.Abs(die.Velocity.X) * -1
		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.X < MinWidth {
		die.Vec2.X = MinWidth + 1
		die.Velocity.X = math.Abs(die.Velocity.X)
		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.Y+die.TileSize >= MaxHeight {
		die.Vec2.Y = MaxHeight - die.TileSize - 1
		die.Velocity.Y = math.Abs(die.Velocity.Y) * -1
		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
	if die.Vec2.Y < MinHeight {
		die.Vec2.Y = MinHeight + 1
		die.Velocity.Y = math.Abs(die.Velocity.Y)
		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	}
}
