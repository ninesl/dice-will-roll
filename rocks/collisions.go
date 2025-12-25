package rocks

import (
	"math"

	"github.com/ninesl/dice-will-roll/render"
)

// dieCollisionData holds pre-computed collision data for a die
// This eliminates redundant calculations when checking multiple rocks against the same die
type dieCollisionData struct {
	left, right, top, bottom   float32 // AABB bounds
	centerX, centerY           float32 // Center point
	velocityX, velocityY       float32 // Raw velocity
	normalizedVX, normalizedVY float32 // Pre-normalized velocity
	speed                      float32 // Pre-computed velocity magnitude
	bounceAngleRad             float64 // Pre-computed atan2 for bounce direction
	isMoving                   bool    // Flag to skip expensive math for stationary dice
}

// sqrt32 computes square root for float32 (avoids float64 conversion overhead)
func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// preprocessDiceCollisionData pre-computes all die-related collision data ONCE per frame
// This is called before processing rocks to avoid recalculating die bounds/velocities for each rock
func preprocessDiceCollisionData(diceCenters []render.Vec2, diceVelocities []render.Vec2) []dieCollisionData {
	data := make([]dieCollisionData, len(diceCenters))

	for i, center := range diceCenters {
		vel := diceVelocities[i]
		isMoving := vel.X != 0 || vel.Y != 0

		data[i] = dieCollisionData{
			left:      center.X - render.HalfEffectiveDie,
			right:     center.X + render.HalfEffectiveDie,
			top:       center.Y - render.HalfEffectiveDie,
			bottom:    center.Y + render.HalfEffectiveDie,
			centerX:   center.X,
			centerY:   center.Y,
			velocityX: vel.X,
			velocityY: vel.Y,
			isMoving:  isMoving,
		}

		// Only compute expensive trig/sqrt if die is actually moving
		if isMoving {
			speed := sqrt32(vel.X*vel.X + vel.Y*vel.Y)
			data[i].speed = speed
			data[i].normalizedVX = vel.X / speed
			data[i].normalizedVY = vel.Y / speed
			data[i].bounceAngleRad = math.Atan2(float64(vel.Y), float64(vel.X))
		}
	}

	return data
}

// RockWithinDie checks if a rock collides with a die using AABB collision detection
// AABB (Axis-Aligned Bounding Box) collision detection with 0.75 multiplier for tighter collision
// Checks if rock's bounding box overlaps with die's bounding box
func (r *SimpleRock) RockWithinDie(die *render.DieRenderable, rockSize float32) bool {
	effectiveDieTileSize := render.DieTileSize * 0.75
	effectiveRockSize := rockSize * 0.75

	// Center the effective collision boxes
	dieInset := (render.DieTileSize - effectiveDieTileSize) / 2
	rockInset := (rockSize - effectiveRockSize) / 2

	return (r.Position.X+rockInset+effectiveRockSize > die.Vec2.X+dieInset && r.Position.X+rockInset < die.Vec2.X+dieInset+effectiveDieTileSize) &&
		(r.Position.Y+rockInset+effectiveRockSize > die.Vec2.Y+dieInset && r.Position.Y+rockInset < die.Vec2.Y+dieInset+effectiveDieTileSize)
}

// XYWithinRock checks if a point (X, Y) is within the rock's bounding box
func (r *SimpleRock) XYWithinRock(X, Y float32, spriteSize float32) bool {
	return (Y >= r.Position.Y && Y <= r.Position.Y+spriteSize) && (X >= r.Position.X && X <= r.Position.X+spriteSize)
}

// IsNearPoint checks if rock is within radius of a point using Manhattan distance (fast, no sqrt)
// This is the BROAD PHASE collision check - cheaper than precise AABB collision
func (r *SimpleRock) IsNearPoint(rockSize, pointX, pointY, radius float32) bool {
	rockCenterX := r.Position.X + rockSize/2
	rockCenterY := r.Position.Y + rockSize/2

	dx := rockCenterX - pointX
	dy := rockCenterY - pointY

	// Inline abs (faster than math.Abs with float64 conversions)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	// Manhattan distance approximation (fast, no sqrt)
	manhattanDist := dx + dy
	return manhattanDist < radius
}

// handleCursorCollisions processes cursor-rock collision responses
func (r *RocksRenderer) handleCursorCollisions(cursorX, cursorY float32) {
	for _, rock := range r.cursorCollisionBuffer {
		sizeData := rock.SizeData()

		if rock.XYWithinRock(cursorX, cursorY, sizeData.Size) {
			// Determine which side of rock the cursor is on
			rockCenterX := rock.Position.X + sizeData.HalfSize
			rockCenterY := rock.Position.Y + sizeData.HalfSize
			dx := cursorX - rockCenterX
			dy := cursorY - rockCenterY

			// Calculate position cells BEFORE movement (for distance-based rotation)
			oldCellX := int(rock.Position.X / float32(DEGREES_PER_FRAME))
			oldCellY := int(rock.Position.Y / float32(DEGREES_PER_FRAME))

			// Inline abs for faster comparison
			absDx := dx
			if absDx < 0 {
				absDx = -absDx
			}
			absDy := dy
			if absDy < 0 {
				absDy = -absDy
			}

			// Push rock away from cursor on the primary collision axis
			if absDx > absDy {
				// Horizontal collision
				if dx > 0 {
					// Cursor is on right side, push rock left
					rock.Position.X = cursorX - sizeData.Size - 1
				} else {
					// Cursor is on left side, push rock right
					rock.Position.X = cursorX + 1
				}

				// Only update sprite rotation if rock moved to a different cell
				newCellX := int(rock.Position.X / float32(DEGREES_PER_FRAME))
				if newCellX != oldCellX {
					if dx > 0 {
						if rock.SpriteSlopeX == 0 {
							rock.SpriteSlopeX = MAX_SLOPE*2 - 1
						} else {
							rock.SpriteSlopeX--
						}
					} else {
						if rock.SpriteSlopeX == MAX_SLOPE*2-1 {
							rock.SpriteSlopeX = 0
						} else {
							rock.SpriteSlopeX++
						}
					}
				}
			} else {
				// Vertical collision
				if dy > 0 {
					// Cursor is below, push rock up
					rock.Position.Y = cursorY - sizeData.Size - 1
				} else {
					// Cursor is above, push rock down
					rock.Position.Y = cursorY + 1
				}

				// Only update sprite rotation if rock moved to a different cell
				newCellY := int(rock.Position.Y / float32(DEGREES_PER_FRAME))
				if newCellY != oldCellY {
					if dy > 0 {
						if rock.SpriteSlopeY == 0 {
							rock.SpriteSlopeY = MAX_SLOPE*2 - 1
						} else {
							rock.SpriteSlopeY--
						}
					} else {
						if rock.SpriteSlopeY == MAX_SLOPE*2-1 {
							rock.SpriteSlopeY = 0
						} else {
							rock.SpriteSlopeY++
						}
					}
				}
			}
		}
	}
}

// RandomXORRockJitter generates 2 psuedo-random numbers in range [-range, +range] using XOR-shift algorithm
// xSeed and ySeed are typically rock position.X and position.Y
// range specifies the jitter range (e.g., range=1 gives [-1,0,1], range=2 gives [-2,-1,0,1,2])
func RandomXORRockJitter(xSeed, ySeed float32, jitterRange int8) (int8, int8) {
	// Convert position to uint32 seed

	var s uint32
	if xSeed > ySeed {
		s = uint32(xSeed) - uint32(ySeed)
	} else {
		s = uint32(xSeed) + uint32(ySeed)
	}

	s = s ^ (s << 13)
	s = s ^ (s >> 7)
	s = s ^ (s << 17)

	// Calculate modulo value: range * 2 + 1 (e.g., range=1 -> 3 values, range=2 -> 5 values)
	modValue := uint32(jitterRange*2 + 1)

	// Extract X jitter from lower bits
	jitterX := int8((s % modValue) - uint32(jitterRange))

	// Extract Y jitter from upper bits
	jitterY := int8(((s >> 8) % modValue) - uint32(jitterRange))

	return jitterX, jitterY
}

// handleDieCollisions processes die-rock collision responses using die centers and velocities
// diceCenters: die center positions (X=centerX, Y=centerY)
// diceVelocities: die velocity vectors (X=velocityX, Y=velocityY) - determines bounce direction
// Each rock collides with at most one die per frame (first collision wins)
func (r *RocksRenderer) handleDieCollisions(diceCenters []render.Vec2, diceVelocities []render.Vec2) {
	if len(r.diceCollisionBuffer) == 0 {
		return
	}

	// Pre-compute all die collision data ONCE (instead of per-rock)
	diceData := preprocessDiceCollisionData(diceCenters, diceVelocities)

	// OUTER LOOP: Each rock (allows break to work correctly)
	for _, rock := range r.diceCollisionBuffer {
		sizeData := rock.SizeData()

		// Track the die with maximum total overlap (deepest penetration)
		// This prevents rocks from getting stuck between multiple dice
		var maxOverlap float32 = -1
		var bestDieIndex int = -1

		// PASS 1: Find die with deepest penetration
		// Calculate rock center using full size, then AABB using effective size (75%)
		rockCenterX := rock.Position.X + sizeData.HalfSize
		rockCenterY := rock.Position.Y + sizeData.HalfSize
		rockLeft := rockCenterX - sizeData.HalfEffective
		rockRight := rockCenterX + sizeData.HalfEffective
		rockTop := rockCenterY - sizeData.HalfEffective
		rockBottom := rockCenterY + sizeData.HalfEffective

		for dieIdx, dieData := range diceData {
			// NARROW PHASE: Check for actual AABB overlap using pre-computed die bounds
			if rockRight <= dieData.left || rockLeft >= dieData.right ||
				rockBottom <= dieData.top || rockTop >= dieData.bottom {
				continue // No collision with this die, try next die
			}

			// Calculate overlap amounts using min/max (branchless, CPU-friendly)
			xOverlap := min(rockRight, dieData.right) - max(rockLeft, dieData.left)
			yOverlap := min(rockBottom, dieData.bottom) - max(rockTop, dieData.top)

			// Calculate total overlap (deepest penetration wins)
			totalOverlap := xOverlap + yOverlap

			// Track die with maximum overlap
			if totalOverlap > maxOverlap {
				maxOverlap = totalOverlap
				bestDieIndex = dieIdx
			}
		}

		// If no collision found, skip this rock
		if bestDieIndex == -1 {
			continue
		}

		// PASS 2: Process collision with the die that has deepest penetration
		// Use pre-computed die data instead of recalculating
		dieData := diceData[bestDieIndex]

		// Calculate bounce direction based on die's velocity (use pre-computed values!)
		var bounceAngleRad float64
		var pushDirX, pushDirY float32

		if dieData.isMoving {
			// Die is moving - use pre-computed velocity data (NO sqrt/atan2 needed!)
			bounceAngleRad = dieData.bounceAngleRad
			pushDirX = dieData.normalizedVX
			pushDirY = dieData.normalizedVY
		} else {
			// Die is stationary - fall back to position-based bounce (from die center to rock center)
			dx := rockCenterX - dieData.centerX
			dy := rockCenterY - dieData.centerY
			bounceAngleRad = math.Atan2(float64(dy), float64(dx))

			// Inline abs instead of math.Abs(float64())
			absDx := dx
			if absDx < 0 {
				absDx = -absDx
			}
			absDy := dy
			if absDy < 0 {
				absDy = -absDy
			}
			distance := absDx + absDy // Manhattan distance (fast approximation)

			if distance > 0 {
				pushDirX = dx / distance
				pushDirY = dy / distance
			} else {
				pushDirX = 1
				pushDirY = 0
			}
		}

		bounceAngleDeg := bounceAngleRad * 180.0 / math.Pi

		// Normalize to 0-360 range
		if bounceAngleDeg < 0 {
			bounceAngleDeg += 360
		}

		// Calculate size-based speed boost (smaller rocks = faster)
		var sizeBoost int8
		switch rock.Score.GetScore() {
		case SmallScore: // 1 - smallest rocks
			sizeBoost = 4
		case MediumScore: // 3 - medium rocks
			sizeBoost = 3
		case BigScore: // 5 - big rocks
			sizeBoost = 2
		case HugeScore: // 10 - huge rocks
			sizeBoost = 1
		default:
			sizeBoost = 1
		}

		rock.BounceTowardsAngle(int(bounceAngleDeg))

		// Generate fast pseudo-random jitter using rock position (range: -1 to +1)
		jitterX, jitterY := RandomXORRockJitter(rock.Position.X, rock.Position.Y, ROCK_JITTER)

		// Apply to ALL bounces to prevent infinite bouncing loops
		if rock.SlopeX == 0 && rock.SlopeY != 0 {
			// Vertical bounce
			if bounceAngleDeg < 180 {
				rock.SlopeX = 1 + jitterX
			} else {
				rock.SlopeX = -1 + jitterX
			}
		} else if rock.SlopeY == 0 && rock.SlopeX != 0 {
			// Horizontal bounce
			if bounceAngleDeg < 90 || bounceAngleDeg >= 270 {
				rock.SlopeY = 1 + jitterY
			} else {
				rock.SlopeY = -1 + jitterY
			}
		} else {
			// Both slopes non-zero
			rock.SlopeX += jitterX
			rock.SlopeY += jitterY
		}

		// Apply size boost to the calculated slopes
		if rock.SlopeX > 0 {
			rock.SlopeX += sizeBoost
		} else if rock.SlopeX < 0 {
			rock.SlopeX -= sizeBoost
		}

		if rock.SlopeY > 0 {
			rock.SlopeY += sizeBoost
		} else if rock.SlopeY < 0 {
			rock.SlopeY -= sizeBoost
		}

		// Clamp slopes to valid range [MIN_SLOPE, MAX_SLOPE]
		if rock.SlopeX > MAX_SLOPE {
			rock.SlopeX = MAX_SLOPE
		} else if rock.SlopeX < MIN_SLOPE {
			rock.SlopeX = MIN_SLOPE
		}

		if rock.SlopeY > MAX_SLOPE {
			rock.SlopeY = MAX_SLOPE
		} else if rock.SlopeY < MIN_SLOPE {
			rock.SlopeY = MIN_SLOPE
		}

		// Position snap for rocks moving in same direction as die's push
		// This creates a "forceful push" effect instead of gentle nudging
		// ONLY apply if die is moving (not stationary)
		if dieData.isMoving && (rock.SlopeX != 0 || rock.SlopeY != 0) {
			// Normalize rock's current velocity for dot product
			rockSpeed := float32(math.Sqrt(float64(rock.SlopeX*rock.SlopeX + rock.SlopeY*rock.SlopeY)))
			if rockSpeed > 0 {
				rockDirX := float32(rock.SlopeX) / rockSpeed
				rockDirY := float32(rock.SlopeY) / rockSpeed

				// Dot product: tells us if rock is moving in same direction as die's push
				// Result: 1.0 = same direction, -1.0 = opposite, 0 = perpendicular
				dotProduct := rockDirX*pushDirX + rockDirY*pushDirY

				// If rock is moving in same direction (within 45° = dot product > 0.707)
				// cos(45°) ≈ 0.707
				if dotProduct > 0.707 {
					// SNAP position to push rock forcefully out of the way
					pushDistance := maxOverlap * 0.5
					rock.Position.X += pushDirX * pushDistance
					rock.Position.Y += pushDirY * pushDistance
				}
			}
		}
	}
}
