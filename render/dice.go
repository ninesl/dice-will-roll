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
	DiceBottom      float64
	DampingFactor   float64 = 0.7
	BounceFactor    float64 = .9
	BounceThreshold float64 = 4.0
	VelocityMin     float64 = BounceThreshold * 2
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

				FixOverlap(die, die2)

				// die.Velocity.X *= -BounceFactor
				// die2.Velocity.X *= -BounceFactor
				// die.Velocity.Y *= -BounceFactor
				// die2.Velocity.Y *= -BounceFactor
			}
		}
	}
}

func FixOverlap(die *DieRenderable, die2 *DieRenderable) {
	// die.IndexOnSheet = rand.IntN1(5)
	// die2.IndexOnSheet = rand.IntN(5)

	if die.Velocity.X < BounceThreshold && die.Velocity.Y < BounceThreshold {
		return
	}

	if die2.Velocity.X < BounceThreshold && die2.Velocity.Y < BounceThreshold {
		return
	}

	xOverlap := (die.Vec2.X - die2.Vec2.X) / 1.5
	yOverlap := (die.Vec2.Y - die2.Vec2.Y) / 1.5

	if xOverlap > yOverlap {
		die.Vec2.X += xOverlap
		die2.Vec2.X -= xOverlap

		var speed float64

		if math.Abs(die.Velocity.X) > math.Abs(die2.Velocity.X) {
			speed = die.Velocity.X * DampingFactor

			// die.Velocity.X -= speed / 2
			die2.Velocity.X += speed / 2
		} else {
			speed = die2.Velocity.X * DampingFactor

			// die2.Velocity.X -= speed / 2
			die.Velocity.X += speed / 2
		}

		die.Velocity.X *= -1
		die2.Velocity.X *= -1
	} else {
		die.Vec2.Y += yOverlap
		die2.Vec2.Y -= yOverlap

		die.Velocity.Y *= -1
		die2.Velocity.Y *= -1
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
