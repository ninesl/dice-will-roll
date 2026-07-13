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
	rotationZ                  float64 // Die's visual rotation (0.0-1.0 = 0°-360°) in radians
	isMoving                   bool    // Flag to skip expensive math for stationary dice
	skipMe                     bool    // flag to use to skip colliding
}

// Precomputed cell offset patterns based on die center quadrant within its cell
// Die center position determines which 2x2 quadrant it's in, then we add the + arms
//
// Cell quadrants:
// ┌─────────┬─────────┐
// │ Q0      │ Q1      │
// │top-left │top-right│
// ├─────────┼─────────┤
// │ Q2      │ Q3      │
// │bot-left │bot-right│
// └─────────┴─────────┘
//
// Each pattern = 2x2 quadrant (4 cells) + opposite 2 cells from + pattern = 6 cells total
var quadrantCellPatterns = [4][6][2]int{
	// Q0 top-left: 2x2(top-left, top, left, center) + right, bottom
	{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}, {1, 0}, {0, 1}},
	// Q1 top-right: 2x2(top, top-right, center, right) + left, bottom
	{{0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {0, 1}},
	// Q2 bottom-left: 2x2(left, center, bottom-left, bottom) + top, right
	{{0, -1}, {-1, 0}, {0, 0}, {1, 0}, {-1, 1}, {0, 1}},
	// Q3 bottom-right: 2x2(center, right, bottom, bottom-right) + top, left
	{{0, -1}, {-1, 0}, {0, 0}, {1, 0}, {0, 1}, {1, 1}},
}

// collectCollisionCandidatesFromGrid uses spatial grid to find rocks near dice/cursor
// O(numDice * 5cells * rocksPerCell) instead of O(numRocks * numDice)
// Uses die rotation to determine which 5 cells to check (instead of all 9)
func (r *RocksRenderer) collectCollisionCandidatesFromGrid(
	cursorX, cursorY float32,
	diceCenters []render.Vec3,
) {
	invCellSize := 1.0 / r.gridCellSize

	// Collect rocks near cursor (single cell check)
	cursorCellX := int(cursorX * invCellSize)
	cursorCellY := int(cursorY * invCellSize)

	if cursorCellX >= 0 && cursorCellX < r.gridCols && cursorCellY >= 0 && cursorCellY < r.gridRows {
		cellIdx := cursorCellY*r.gridCols + cursorCellX
		offset := r.gridOffsets[cellIdx]
		count := r.gridCounts[cellIdx]

		for i := uint16(0); i < uint16(count); i++ {
			rockID := r.gridRocks[offset+i]
			r.cursorCollisionBuffer = append(r.cursorCollisionBuffer, rockID)
		}
	}

	// Collect rocks near each die using quadrant-based cell lookup
	// Instead of 3x3 (9 cells), we check 6 cells based on die center position
	for _, dieCenter := range diceCenters {
		dieCellX := int(dieCenter.X * invCellSize)
		dieCellY := int(dieCenter.Y * invCellSize)

		// Determine which quadrant of the cell the die center is in
		// localX/Y = position within cell (0.0 to 1.0)
		localX := (dieCenter.X * invCellSize) - float32(dieCellX)
		localY := (dieCenter.Y * invCellSize) - float32(dieCellY)

		// Calculate quadrant index: 0=top-left, 1=top-right, 2=bottom-left, 3=bottom-right
		var quadrant int
		if localX < 0.5 {
			if localY < 0.5 {
				quadrant = 0 // top-left
			} else {
				quadrant = 2 // bottom-left
			}
		} else {
			if localY < 0.5 {
				quadrant = 1 // top-right
			} else {
				quadrant = 3 // bottom-right
			}
		}

		// Get precomputed cell offsets for this quadrant (6 cells)
		cellPattern := quadrantCellPatterns[quadrant]

		// Check the 6 cells (2x2 quadrant + 2 opposite + arms)
		for _, offset := range cellPattern {
			checkX := dieCellX + offset[0]
			checkY := dieCellY + offset[1]

			// Skip out of bounds
			if checkX < 0 || checkX >= r.gridCols || checkY < 0 || checkY >= r.gridRows {
				continue
			}

			cellIdx := checkY*r.gridCols + checkX
			gridOffset := r.gridOffsets[cellIdx]
			count := r.gridCounts[cellIdx]

			// Sequential iteration through contiguous memory (L1 cache friendly)
			for i := uint16(0); i < uint16(count); i++ {
				rockID := r.gridRocks[gridOffset+i]
				r.diceCollisionBuffer = append(r.diceCollisionBuffer, rockID)
			}
		}
	}
}

// sqrt32 computes square root for float32 (avoids float64 conversion overhead)
func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// CollideAndAnimateRocks performs all rock updates, wall bouncing, and collision detection/response
// diceCenters: die center positions (X=centerX, Y=centerY, Z=rotationZ)
// diceVelocities: die velocity vectors (X=velocityX, Y=velocityY) for bounce direction
func (r *RocksRenderer) CollideAndAnimateRocks(cursorX, cursorY float32, diceCenters []render.Vec3, diceVelocities []render.Vec2) {
	// Reset collision buffers to length 0 (keeps capacity - no allocation)
	r.diceCollisionBuffer = r.diceCollisionBuffer[:0]
	r.cursorCollisionBuffer = r.cursorCollisionBuffer[:0]
	r.updatingBuffers = append(r.updatingBuffers[:0], r.ActiveBaseBuffer)

	for _, buffer := range r.HeldColorBuffers {
		r.updatingBuffers = append(r.updatingBuffers, buffer)
	}
	r.updatingBuffers = append(r.updatingBuffers, r.TransitionBuffers...)
	for _, buffer := range r.updatingBuffers {
		for i := range buffer.RockIDs {
			rock := &r.Rocks[r.ActiveBaseBufferIdx][buffer.RockIDs[i]]

			rock.Position.Y += BaseVelocity * float32(rock.SlopeY)
			rock.Position.X += BaseVelocity * float32(rock.SlopeX)
			sizeData := rock.SizeData()

			// Wall bouncing
			if rock.Position.X+sizeData.Size >= render.GAME_BOUNDS_X {
				rock.Position.X = render.GAME_BOUNDS_X - sizeData.Size
				rock.BounceX()
			} else if rock.Position.X <= 0 {
				rock.Position.X = 0
				rock.BounceX()
			}

			if rock.Position.Y+sizeData.Size >= render.GAME_BOUNDS_Y {
				rock.Position.Y = render.GAME_BOUNDS_Y - sizeData.Size
				rock.BounceY()
			} else if rock.Position.Y <= 0 {
				rock.Position.Y = 0
				rock.BounceY()
			}
			rock.UpdateAnimation()
		}
	}

	// PASS 2: REBUILD GRID - Update spatial partitioning after positions changed
	r.rebuildGrid()

	// PASS 3: BROAD PHASE - Collect collision candidates using spatial grid
	// O(numDice * 9cells * avgRocksPerCell) instead of O(numRocks * numDice)
	r.collectCollisionCandidatesFromGrid(cursorX, cursorY, diceCenters)

	// PASS 4: NARROW PHASE - Precise collision checks and responses
	r.handleCursorCollisions(cursorX, cursorY)
	r.handleDieCollisions(diceCenters, diceVelocities)

	// PASS 5: DAMPING - Apply velocity reduction AFTER all collisions
	// Held color buffers DO NOT get damping (maintain speed while held by dice)
	r.updatingBuffers = append(r.updatingBuffers[:0], r.ActiveBaseBuffer)
	for _, buffer := range r.TransitionBuffers {
		r.updatingBuffers = append(r.updatingBuffers, buffer)
	}
	for _, buffer := range r.updatingBuffers {
		for i := range buffer.RockIDs {
			rock := &r.Rocks[r.ActiveBaseBufferIdx][buffer.RockIDs[i]]
			rock.ApplyDamping()
		}
	}

	// Update all buffer transitions and move completed transition buffers to base buffers
	r.updateAllBufferTransitions()

	// Update explosion animations
	r.updateExplosions()
}

// preprocessDiceCollisionData pre-computes all die-related collision data ONCE per frame
// This is called before processing rocks to avoid recalculating die bounds/velocities for each rock
// diceCenters: Vec3 where X=centerX, Y=centerY, Z=ZRotation (0.0-1.0)
// Uses pre-allocated buffer to avoid per-frame heap allocations
func (r *RocksRenderer) preprocessDiceCollisionData(diceCenters []render.Vec3, diceVelocities []render.Vec2) []dieCollisionData {
	// Reuse buffer, grow only if needed
	// if cap(r.diceCollisionDataBuffer) < len(diceCenters) {
	// 	r.diceCollisionDataBuffer = make([]dieCollisionData, len(diceCenters))
	// }
	r.diceCollisionDataBuffer = r.diceCollisionDataBuffer[:len(diceCenters)]

	for i, center := range diceCenters {
		// Skip dice in score zone that are moving
		// if center.Y < render.SCOREZONE.MaxHeight &&
		// 	(diceVelocities[i].X != 0 || diceVelocities[i].Y != 0) {

		// 	r.diceCollisionDataBuffer[i] = dieCollisionData{
		// 		skipMe: true,
		// 	}
		// 	continue
		// }

		vel := diceVelocities[i]
		isMoving := vel.X != 0 || vel.Y != 0

		// Convert ZRotation (0.0-1.0) to radians (0 to 2π)
		rotationRad := float64(center.Z) * 2 * math.Pi

		// Calculate rotated AABB half-extent: h × (|cos θ| + |sin θ|)
		// At 0°: halfExtent = h (axis-aligned square)
		// At 45°: halfExtent = h × √2 (corners extend further) FIXME: this isn't good here at all, not accurate enough
		cosR := math.Abs(math.Cos(rotationRad))
		sinR := math.Abs(math.Sin(rotationRad))
		halfExtent := float32(float64(render.HalfEffectiveDie) * (cosR + sinR))

		r.diceCollisionDataBuffer[i] = dieCollisionData{
			left:      center.X - halfExtent,
			right:     center.X + halfExtent,
			top:       center.Y - halfExtent,
			bottom:    center.Y + halfExtent,
			centerX:   center.X,
			centerY:   center.Y,
			velocityX: vel.X,
			velocityY: vel.Y,
			rotationZ: rotationRad,
			isMoving:  isMoving,
		}

		// Only compute expensive trig/sqrt if die is actually moving
		if isMoving {
			speed := sqrt32(vel.X*vel.X + vel.Y*vel.Y)
			r.diceCollisionDataBuffer[i].speed = speed
			r.diceCollisionDataBuffer[i].normalizedVX = vel.X / speed
			r.diceCollisionDataBuffer[i].normalizedVY = vel.Y / speed
			r.diceCollisionDataBuffer[i].bounceAngleRad = math.Atan2(float64(vel.Y), float64(vel.X))
		}
	}

	return r.diceCollisionDataBuffer
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

	return (r.Position.X+rockInset+effectiveRockSize > die.Vec2.X+dieInset &&
		r.Position.X+rockInset < die.Vec2.X+dieInset+effectiveDieTileSize) &&
		(r.Position.Y+rockInset+effectiveRockSize > die.Vec2.Y+dieInset &&
			r.Position.Y+rockInset < die.Vec2.Y+dieInset+effectiveDieTileSize)
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
	for _, rockID := range r.cursorCollisionBuffer {
		rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]
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

			rock.Bounce(rockCursorBounceSlope(rockCenterX-cursorX, rockCenterY-cursorY))
		}
	}
}

func rockCursorBounceSlope(dx, dy float32) (int8, int8) {
	if dx < 0 {
		if dy < 0 {
			return -MAX_SLOPE, -MAX_SLOPE
		}
		if dy > 0 {
			return -MAX_SLOPE, MAX_SLOPE
		}
		return -MAX_SLOPE, 0
	}

	if dx > 0 {
		if dy < 0 {
			return MAX_SLOPE, -MAX_SLOPE
		}
		if dy > 0 {
			return MAX_SLOPE, MAX_SLOPE
		}
		return MAX_SLOPE, 0
	}

	if dy < 0 {
		return 0, -MAX_SLOPE
	}
	return 0, MAX_SLOPE
}

// handleDieCollisions processes die-rock collision responses using die centers and velocities
// diceCenters: die center positions (X=centerX, Y=centerY)
// diceVelocities: die velocity vectors (X=velocityX, Y=velocityY) - determines bounce direction
// Each rock collides with at most one die per frame (first collision wins)
func (r *RocksRenderer) handleDieCollisions(diceCenters []render.Vec3, diceVelocities []render.Vec2) {
	if len(r.diceCollisionBuffer) == 0 {
		return
	}

	r.diceCollisionDieIndexesBuffer = r.diceCollisionDieIndexesBuffer[:0]

	// Pre-compute all die collision data ONCE (instead of per-rock)
	diceData := r.preprocessDiceCollisionData(diceCenters, diceVelocities)

	// OUTER LOOP: Each rock (allows break to work correctly)

	//TODO: make each collision buffer its own collection to not waste work
	// by checking each frame WHICH die it's in. we should already know

	// each rock needs a die index to bounce from

	for _, rockID := range r.diceCollisionBuffer {
		rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]
		sizeData := rock.SizeData()

		// Track the die with maximum total overlap
		// This prevents rocks from getting stuck between multiple dice
		var maxOverlap float32 = -1
		var bestDieIndex int = -1

		// PASS 1: Find die with maximum total overlap
		// Calculate rock center using full size, then AABB using effective size
		rockCenterX := rock.Position.X + sizeData.HalfSize
		rockCenterY := rock.Position.Y + sizeData.HalfSize
		rockLeft := rockCenterX - sizeData.HalfEffective
		rockRight := rockCenterX + sizeData.HalfEffective
		rockTop := rockCenterY - sizeData.HalfEffective
		rockBottom := rockCenterY + sizeData.HalfEffective

		//TODO: this is the collision buffer, each dice data should already know
		// which rocks it needs to check on, not this branching waste of work.
		// should be better for cache thruput as well
		for dieIdx, dieData := range diceData {
			if dieData.skipMe {
				continue
			}

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

		r.diceCollisionDieIndexesBuffer = append(r.diceCollisionDieIndexesBuffer, bestDieIndex)
	}

	var bestDieIndex int = -1

	for i, rockID := range r.diceCollisionBuffer {
		rock := &r.Rocks[r.ActiveBaseBufferIdx][rockID]
		bestDieIndex = r.diceCollisionDieIndexesBuffer[i]

		// If no collision found, skip this rock
		if bestDieIndex == -1 {
			continue
		}

		// PASS 2: Process collision with the die that has deepest penetration
		// Use pre-computed die data instead of recalculating
		dieData := diceData[bestDieIndex]
		sizeData := rock.SizeData()
		//TODO: recalcing this hsould be fine?
		rockCenterX := rock.Position.X + sizeData.HalfSize
		rockCenterY := rock.Position.Y + sizeData.HalfSize

		// NOTE: slop distance calc here. heaviest work

		// Calculate angle from die center to rock center (world space)
		angleToRock := math.Atan2(
			float64(rockCenterY-dieData.centerY),
			float64(rockCenterX-dieData.centerX),
		)

		// Calculate angle relative to die's rotated frame
		// This tells us which edge of the rotated die the rock is hitting
		relativeAngle := angleToRock - dieData.rotationZ

		// Normalize to 0-2π range
		for relativeAngle < 0 {
			relativeAngle += 2 * math.Pi
		}
		for relativeAngle >= 2*math.Pi {
			relativeAngle -= 2 * math.Pi
		}

		// Calculate distance from die center to edge of rotated square at this angle
		// For a square: edgeDistance = halfSize / max(|cos(θ)|, |sin(θ)|)
		halfSize := float64(render.HalfEffectiveDie)
		cosA := math.Abs(math.Cos(relativeAngle))
		sinA := math.Abs(math.Sin(relativeAngle))
		edgeDistance := halfSize / math.Max(cosA, sinA)

		// TRUE NARROW PHASE: Verify rock is actually colliding with rotated square
		// (AABB is an over-approximation, this catches false positives in corner regions)
		dx := float64(rockCenterX - dieData.centerX)
		dy := float64(rockCenterY - dieData.centerY)
		rockDistFromDie := math.Sqrt(dx*dx + dy*dy)

		// Collision threshold: die edge + rock's effective radius
		collisionThreshold := edgeDistance + float64(sizeData.HalfEffective)
		if rockDistFromDie >= collisionThreshold {
			continue // Rock is in AABB but outside actual rotated square - no collision
		}

		// Rock CENTER should be at: dieEdge + rockHalfEffective + 2px buffer
		rockCenterDistance := edgeDistance + float64(sizeData.HalfEffective) + 2

		// Snap rock center to that distance along the angle
		newCenterX := dieData.centerX + float32(math.Cos(angleToRock)*rockCenterDistance)
		newCenterY := dieData.centerY + float32(math.Sin(angleToRock)*rockCenterDistance)

		// NOTE: end of slop collision check

		// Rock position is top-left corner, so subtract half size
		rock.Position.X = newCenterX - sizeData.HalfSize
		rock.Position.Y = newCenterY - sizeData.HalfSize

		// Calculate bounce direction based on the edge normal of the rotated die
		// The edge normal depends on which side of the rotated square we hit
		var bounceAngleRad float64

		// The outward normal at the contact point is simply the angle from center to rock
		// (since we're pushing the rock directly away from the die center along the edge)
		bounceAngleRad = angleToRock
		bounceAngleDeg := bounceAngleRad * 180.0 / math.Pi

		// Normalize to 0-360 range
		if bounceAngleDeg < 0 {
			bounceAngleDeg += 360
		}

		rock.BounceTowardsAngle(int(bounceAngleDeg))

		xJitter, yJitter := RandomXORRockJitter(rock.Position.X, rock.Position.Y, 1)

		rock.SlopeX += xJitter
		rock.SlopeY += yJitter
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

// FIXME: we need to rethink how the grids work

// initSpatialGrid initializes the hybrid offset+count spatial grid
// Cell size is render.DieTileSize for optimal collision detection with dice
func (r *RocksRenderer) initSpatialGrid(config RocksConfig) {
	r.gridCellSize = render.DieTileSize
	r.gridCols = int(math.Ceil(float64(config.WorldBoundsX) / float64(r.gridCellSize)))
	r.gridRows = int(math.Ceil(float64(config.WorldBoundsY) / float64(r.gridCellSize)))

	totalCells := r.gridCols * r.gridRows

	// Pre-allocate arrays
	r.gridOffsets = make([]uint16, totalCells)
	r.gridCounts = make([]uint16, totalCells)
	r.gridRocks = make([]RockID, config.TotalRocks[r.ActiveBaseBufferIdx])
}

func clampCell(gridCols, gridRows, cellX, cellY int) int {
	if cellX < 0 {
		cellX = 0
	} else if cellX >= gridCols {
		cellX = gridCols - 1
	}
	if cellY < 0 {
		cellY = 0
	} else if cellY >= gridRows {
		cellY = gridRows - 1
	}
	return cellY*gridCols + cellX
}

// rebuildGrid rebuilds the spatial grid from current rock positions
// Called once per frame before collision detection
// Uses 2-pass algorithm: count rocks per cell, then place indices
// Includes ALL buffers: base, held, and transition
// IMPORTANT: Uses rock CENTER for grid cell assignment (not top-left corner)
func (r *RocksRenderer) rebuildGrid() {
	// Reset counts to 0
	for i := range r.gridCounts {
		r.gridCounts[i] = 0
	}

	// Inverse cell size for fast division (multiply instead of divide)
	invCellSize := 1.0 / r.gridCellSize

	// Helper to clamp cell coordinates
	// Phase 1: Count rocks per cell (all buffers)
	// Use rock CENTER for cell assignment to ensure consistent collision detection
	var totalRocks uint16 = 0

	// Count base buffer rocks
	for bufIdx := range r.BaseColorBuffers {
		buffer := &r.BaseColorBuffers[bufIdx]
		for i := range buffer.RockIDs {
			rock := r.Rocks[r.ActiveBaseBufferIdx][buffer.RockIDs[i]]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			r.gridCounts[cellIdx]++
			totalRocks++
		}
	}

	// Count held buffer rocks
	for _, buffer := range r.HeldColorBuffers {
		for i := range buffer.RockIDs {
			rock := r.Rocks[r.ActiveBaseBufferIdx][buffer.RockIDs[i]]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			r.gridCounts[cellIdx]++
			totalRocks++
		}
	}

	// Count transition buffer rocks
	for _, buffer := range r.TransitionBuffers {
		for i := range buffer.RockIDs {
			rock := r.Rocks[r.ActiveBaseBufferIdx][buffer.RockIDs[i]]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			r.gridCounts[cellIdx]++
			totalRocks++
		}
	}

	// Grow gridRocks if needed (rocks can move between buffers)
	if int(totalRocks) > len(r.gridRocks) {
		r.gridRocks = make([]RockID, totalRocks)
	}

	// Phase 2: Calculate offsets (prefix sum)
	var currentOffset uint16 = 0
	for i := range r.gridCounts {
		r.gridOffsets[i] = currentOffset
		currentOffset += uint16(r.gridCounts[i])
	}

	// Reset counts to use as placement indices
	for i := range r.gridCounts {
		r.gridCounts[i] = 0
	}

	// Phase 3: Place rock IDs into gridRocks (all buffers)
	// Use rock CENTER for cell assignment (must match Phase 1)
	// Place base buffer rocks
	for bufIdx := range r.BaseColorBuffers {
		buffer := &r.BaseColorBuffers[bufIdx]
		for i := range buffer.RockIDs {
			rockID := buffer.RockIDs[i]
			rock := r.Rocks[r.ActiveBaseBufferIdx][rockID]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			insertPos := r.gridOffsets[cellIdx] + uint16(r.gridCounts[cellIdx])
			r.gridRocks[insertPos] = rockID
			r.gridCounts[cellIdx]++
		}
	}

	// Place held buffer rocks
	for _, buffer := range r.HeldColorBuffers {
		for i := range buffer.RockIDs {
			rockID := buffer.RockIDs[i]
			rock := r.Rocks[r.ActiveBaseBufferIdx][rockID]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			insertPos := r.gridOffsets[cellIdx] + uint16(r.gridCounts[cellIdx])
			r.gridRocks[insertPos] = rockID
			r.gridCounts[cellIdx]++
		}
	}

	// Place transition buffer rocks
	for _, buffer := range r.TransitionBuffers {
		for i := range buffer.RockIDs {
			rockID := buffer.RockIDs[i]
			rock := r.Rocks[r.ActiveBaseBufferIdx][rockID]
			sizeData := rock.SizeData()
			centerX := rock.Position.X + sizeData.HalfSize
			centerY := rock.Position.Y + sizeData.HalfSize
			cellIdx := clampCell(r.gridCols, r.gridRows, int(centerX*invCellSize), int(centerY*invCellSize))
			insertPos := r.gridOffsets[cellIdx] + uint16(r.gridCounts[cellIdx])
			r.gridRocks[insertPos] = rockID
			r.gridCounts[cellIdx]++
		}
	}
}
