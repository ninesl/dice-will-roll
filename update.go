package main

import (
	"fmt"
	"math"
	"math/rand/v2"
)

type Direction uint8

const (
	UP = iota
	DOWN
	LEFT
	RIGHT
	UPLEFT
	UPRIGHT
	DOWNRIGHT
	DOWNLEFT
)

const BUFFER float64 = 2.0

// Renderable for the graphical dice.Die
type DieRenderable struct {
	// Sprite
	// Fixed		 Vec2 	// specific coordinates
	Vec2         Vec2    // the top left of the sprite x,y
	Velocity     Vec2    // how much gets added a frame when rolling
	TileSize     float64 // inside here saves size? unsure
	IndexOnSheet int     // corresponds to the Xth tile on the spritesheet
	// Direction Direction
	// IsResting bool
}

var DiceBottom float64 = MaxHeight / 4.0

var DampingFactor float64 = 0.7
var BounceFactor float64 = 0.9
var BounceThreshold float64 = 6.0

// gross code
func CheckCollisions(dice []*DieRenderable) {
	for i, die := range dice {
		for q, die2 := range dice {
			if i == q {
				continue
			}

			rightSide := die.Vec2.X + die.TileSize
			leftSide := die2.Vec2.X + die2.TileSize

			hitRight := die.Vec2.X < die2.Vec2.X && rightSide >= die2.Vec2.X
			hitLeft := die.Vec2.X > die2.Vec2.X && die.Vec2.X <= leftSide
			//  := die.Vec2.Y+die.TileSize >= die2.Vec2.Y || die.Vec2.Y <= die2.Vec2.Y+die2.TileSize

			// could flag being hit here on sprite?
			if hitRight {
				die.Velocity.X = math.Abs(die.Velocity.X) * -1 // force left
				die2.Velocity.X = math.Abs(die.Velocity.X)     // force right

				overlap := (rightSide - die2.Vec2.X) / 2.0
				die.Vec2.X -= overlap
				die2.Vec2.X += overlap

			} else if hitLeft {
				die.Velocity.X = math.Abs(die.Velocity.X) * -1 // force left
				die2.Velocity.X = math.Abs(die.Velocity.X)     // force right

				overlap := (die2.Vec2.X - leftSide) / 2.0
				die.Vec2.X += overlap
				die2.Vec2.X -= overlap
			}
		}
	}
}

// Update handles the movement and bouncing of the DieSprite under gravity.
func UpdateDie(d *DieRenderable) {
	fmt.Printf("%.4f %.4f\n", d.Velocity.X, d.Velocity.Y)

	var hitLeft, hitRight, hitTop, hitBottom bool

	if math.Abs(d.Velocity.Y) < BounceFactor*2 {
		// return to center
		// hitTop = d.Vec2.Y <
		return
	} else {
		hitTop = d.Vec2.Y < 0
		hitBottom = d.Vec2.Y+d.TileSize >= MaxHeight
	}

	if math.Abs(d.Velocity.X) < BounceFactor*16 {
		return
	} else {
		hitLeft = d.Vec2.X < MinWidth
		hitRight = d.Vec2.X+d.TileSize >= MaxWidth
	}

	// if math.Abs(d.Velocity.X) < BounceThreshold || math.Abs(d.Velocity.Y) < BounceFactor {
	// 	return
	// }

	// could add damping, gravity, etc
	// 3. Check for boundary collisions

	// 4. Handle collisions: Reverse velocity, apply damping, clamp position
	// Horizontal bounce (Walls)
	if hitLeft || hitRight {
		d.Velocity.X *= -1.0 * BounceFactor * DampingFactor

		if hitLeft {
			d.Vec2.X = MinWidth
		} else {
			d.Vec2.X = MaxWidth - d.TileSize
		}

	}

	if hitTop || hitBottom {
		d.Velocity.Y *= -1.0 * BounceFactor //* DampingFactor

		if hitTop {
			d.Vec2.Y = 0
		} else {
			d.Vec2.Y = MaxHeight - d.TileSize
		}
	}

	if hitLeft || hitRight || hitBottom || hitTop {
		d.IndexOnSheet = rand.IntN(5)
	}

	d.Vec2.X = d.Vec2.X + d.Velocity.X
	d.Vec2.Y = d.Vec2.Y + d.Velocity.Y

	// fmt.Println(d.Velocity)

	// log.Printf("%v", d.Vec2)
	// // Vertical bounce (Floor)
	// if hitBottom {
	// 	// d.Velocity.X *= DampingFactor

	// 	// Only reverse and dampen if it has significant downward velocity
	// 	// This helps prevent getting stuck if gravity pulls it slightly below floor
	// 	if math.Abs(d.Velocity.Y) > 0.1 { // Threshold to prevent tiny bounces when settling
	// 		d.Velocity.Y *= -1.0 //* DampingFactor // Reverse and dampen Y velocity (now points up)
	// 	} else {
	// 		d.Velocity.Y = 0 // Stop vertical motion if it was barely moving down
	// 	}
	// 	nextY = MaxHeight - d.TileSize // Clamp position firmly to the floor

	// 	// Check if it should be considered resting
	// 	if math.Abs(d.Velocity.Y) < 0.1 {
	// 		d.IsResting = true
	// 		d.Velocity.Y = 0 // Ensure it stops fully
	// 		// Optionally stop X velocity on rest too, or let it slide
	// 		// d.Velocity.X *= 0.9 // Example friction while resting/sliding

	// 	}
	// 	if math.Abs(d.Velocity.X) < 0.1 {
	// 		d.Velocity.X = 0
	// 	}
	// }

	// // 5. Update the actual position using the (potentially clamped) next coordinates
	// d.Vec2.X = nextX
	// d.Vec2.Y = nextY

	// // // 6. Optional: Update Direction enum based on final velocity (if you use it)
	// // if !d.IsResting { // Only update direction if moving
	// // 	if d.Velocity.X < 0 && d.Velocity.Y < 0 {
	// // 		d.Direction = UPLEFT
	// // 	} else if d.Velocity.X > 0 && d.Velocity.Y < 0 {
	// // 		d.Direction = UPRIGHT
	// // 	} else if d.Velocity.X < 0 && d.Velocity.Y > 0 {
	// // 		d.Direction = DOWNLEFT
	// // 	} else if d.Velocity.X > 0 && d.Velocity.Y > 0 {
	// // 		d.Direction = DOWNRIGHT
	// // 	}
	// // 	// ... (add cases for horizontal/vertical only if needed)
	// }
}
