package main

import (
	"fmt"
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
type DieSprite struct {
	Sprite
	Velocity Vec2
	TileSize float64
	// Direction Direction
	// IsResting bool
}

var DiceBottom float64 = MaxHeight / 4.0

var DampingFactor float64 = 0.9
var BounceFactor float64 = 0.9

// Update handles the movement and bouncing of the DieSprite under gravity.
func UpdateDieSprite(d *DieSprite) {

	// could add damping, gravity, etc
	// d.Velocity.X =

	// 3. Check for boundary collisions
	hitLeft := d.Vec2.X < MinWidth
	hitRight := d.Vec2.X+d.TileSize >= MaxWidth

	hitTop := d.Vec2.Y < 0
	hitBottom := d.Vec2.Y+d.TileSize >= MaxHeight

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
		d.SpriteSheet.ActiveFrame = rand.IntN(5)
	}

	d.Vec2.X = d.Vec2.X + d.Velocity.X
	d.Vec2.Y = d.Vec2.Y + d.Velocity.Y

	fmt.Println(d.Velocity)

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
