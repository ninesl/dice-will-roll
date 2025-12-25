package rocks

import (
	"math/rand"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

// updateBufferRocks updates all rocks in a buffer: physics, wall bouncing, and collision detection
func (r *RocksRenderer) updateBufferRocks(
	buffer *RockBuffer,
	cursorX, cursorY float32,
	diceCenters []render.Vec2,
	diceVelocities []render.Vec2,
) {
	buffer.FrameCounter++

	for i := range buffer.Rocks {
		rock := &buffer.Rocks[i]
		rock.Update(buffer.FrameCounter)

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

		// BROAD PHASE: Collect rocks near cursor
		if rock.IsNearPoint(sizeData.Size, cursorX, cursorY, r.CursorCheckRadius) {
			r.cursorCollisionBuffer = append(r.cursorCollisionBuffer, rock)
		}

		// BROAD PHASE: Collect rocks near any die center
		for j := range diceCenters {
			if rock.IsNearPoint(sizeData.Size, diceCenters[j].X, diceCenters[j].Y, r.DieCheckRadius) {
				r.diceCollisionBuffer = append(r.diceCollisionBuffer, rock)
				break // Only add once even if near multiple dice
			}
		}
	}
}

// updateAllBufferTransitions decrements transition counters for all buffer types
func (r *RocksRenderer) updateAllBufferTransitions() {
	// Update HeldColorBuffers transitions (stop at 0, don't go negative)
	for _, buffer := range r.HeldColorBuffers {
		if buffer.Transition > 0 {
			buffer.Transition--
		}
	}

	// Update TransitionBuffers transitions and move completed ones to base buffers
	for i := len(r.TransitionBuffers) - 1; i >= 0; i-- {
		buffer := r.TransitionBuffers[i]

		// Decrement transition counter (stop at 0)
		if buffer.Transition > 0 {
			buffer.Transition--
		}

		// If transition complete, move rocks to base buffer
		if buffer.Transition <= 0 {
			// Find matching base buffer by color
			for j := range r.BaseColorBuffers {
				baseBuffer := &r.BaseColorBuffers[j]

				if render.IsCloseTo(baseBuffer.Color, buffer.Color) {
					baseBuffer.Rocks = append(baseBuffer.Rocks, buffer.Rocks...)
					break
				}
			}

			// Remove this transition buffer
			r.TransitionBuffers = append(r.TransitionBuffers[:i], r.TransitionBuffers[i+1:]...)
		}
	}
}

// UpdateRocksAndCollide performs all rock updates, wall bouncing, and collision detection/response
// diceCenters: die center positions (X=centerX, Y=centerY)
// diceVelocities: die velocity vectors (X=velocityX, Y=velocityY) for bounce direction
func (r *RocksRenderer) UpdateRocksAndCollide(cursorX, cursorY float32, diceCenters []render.Vec2, diceVelocities []render.Vec2) {
	r.ActiveRockFlag = !r.ActiveRockFlag

	// Reset collision buffers to length 0 (keeps capacity - no allocation!)
	r.diceCollisionBuffer = r.diceCollisionBuffer[:0]
	r.cursorCollisionBuffer = r.cursorCollisionBuffer[:0]

	// PASS 1: BROAD PHASE - Update all rocks and collect collision candidates

	// Update base color buffers
	for k := range r.BaseColorBuffers {
		r.updateBufferRocks(&r.BaseColorBuffers[k], cursorX, cursorY, diceCenters, diceVelocities)
	}

	// Update held color buffers
	for _, buffer := range r.HeldColorBuffers {
		r.updateBufferRocks(buffer, cursorX, cursorY, diceCenters, diceVelocities)
	}

	// Update transition buffers
	for _, buffer := range r.TransitionBuffers {
		r.updateBufferRocks(buffer, cursorX, cursorY, diceCenters, diceVelocities)
	}

	// PASS 2: NARROW PHASE - Precise collision checks and responses
	r.handleCursorCollisions(cursorX, cursorY)
	r.handleDieCollisions(diceCenters, diceVelocities)

	// PASS 3: DAMPING - Apply velocity reduction AFTER all collisions
	// Base color buffers get damping
	for k := range r.BaseColorBuffers {
		buffer := &r.BaseColorBuffers[k]
		for i := range buffer.Rocks {
			buffer.Rocks[i].ApplyDamping(buffer.FrameCounter)
		}
	}

	// Held color buffers DO NOT get damping (maintain speed while held by dice)

	// Transition buffers get damping
	for _, buffer := range r.TransitionBuffers {
		for i := range buffer.Rocks {
			buffer.Rocks[i].ApplyDamping(buffer.FrameCounter)
		}
	}

	// Update all buffer transitions and move completed transition buffers to base buffers
	r.updateAllBufferTransitions()
}

// countAvailableRocks returns total rocks in base color buffers
func (r *RocksRenderer) countAvailableRocks() int {
	total := 0
	for i := 0; i < len(r.config.BaseColors); i++ {
		if i < len(r.BaseColorBuffers) {
			total += len(r.BaseColorBuffers[i].Rocks)
		}
	}
	return total
}

// takeRocksFromTransitionBuffers takes rocks from transition buffers when base buffers depleted
// Prioritizes buffers with lowest Transition value (closest to completion)
func (r *RocksRenderer) takeRocksFromTransitionBuffers(needed int) []SimpleRock {
	if len(r.TransitionBuffers) == 0 || needed <= 0 {
		return []SimpleRock{}
	}

	// Sort transition buffers by Transition value (ascending - lowest first)
	slices.SortFunc(r.TransitionBuffers, func(a, b *RockBuffer) int {
		return a.Transition - b.Transition
	})

	// Take rocks from sorted buffers
	collected := make([]SimpleRock, 0, needed)
	for _, buf := range r.TransitionBuffers {
		if needed <= 0 {
			break
		}

		takeCount := needed
		if takeCount > len(buf.Rocks) {
			takeCount = len(buf.Rocks)
		}

		collected = append(collected, buf.Rocks[:takeCount]...)
		buf.Rocks = buf.Rocks[takeCount:]
		needed -= takeCount
	}

	// Clean up empty transition buffers
	r.cleanEmptyTransitionBuffers()

	return collected
}

// cleanEmptyTransitionBuffers removes transition buffers with no rocks
func (r *RocksRenderer) cleanEmptyTransitionBuffers() {
	filtered := make([]*RockBuffer, 0, len(r.TransitionBuffers))
	for _, buf := range r.TransitionBuffers {
		if len(buf.Rocks) > 0 {
			filtered = append(filtered, buf)
		}
	}
	r.TransitionBuffers = filtered
}

// countTransitionRocks returns total rocks in transition buffers
func (r *RocksRenderer) countTransitionRocks() int {
	total := 0
	for _, buf := range r.TransitionBuffers {
		total += len(buf.Rocks)
	}
	return total
}

// SelectRocksColor assigns rocks from base/transition buffers to a die's held buffer
func (r *RocksRenderer) SelectRocksColor(color render.Vec3, dieIdentity render.DieIdentity, numDice int) {
	// Calculate how many dice are NOT currently holding rocks
	numDiceNotHolding := numDice - len(r.HeldColorBuffers)
	if numDiceNotHolding <= 0 {
		return // All dice already holding rocks, nothing to do
	}

	// Calculate rocks needed - divide by dice that don't have rocks yet
	availableInBase := r.countAvailableRocks()
	rocksToTake := availableInBase / numDiceNotHolding

	if rocksToTake <= 0 {
		// Try taking from transition buffers if base is empty
		rocksToTake = r.countTransitionRocks() / numDiceNotHolding
		if rocksToTake <= 0 {
			return // No rocks available at all
		}
	}

	// Collect rocks evenly from base color buffers
	numBaseBuffers := len(r.config.BaseColors)
	if numBaseBuffers == 0 {
		return // Safety check
	}

	rocksPerBuffer := rocksToTake / numBaseBuffers
	remainder := rocksToTake % numBaseBuffers

	collectedRocks := make([]SimpleRock, 0, rocksToTake)

	// Track how many rocks taken from each base buffer for random transition color
	rockCounts := make([]int, numBaseBuffers)

	for i := 0; i < numBaseBuffers && i < len(r.BaseColorBuffers); i++ {
		buffer := &r.BaseColorBuffers[i]

		// Calculate how many to take from this buffer
		numToTake := rocksPerBuffer
		if i < remainder {
			numToTake++ // Distribute remainder rocks
		}

		// Take what's available
		actualTake := numToTake
		if actualTake > len(buffer.Rocks) {
			actualTake = len(buffer.Rocks)
		}

		if actualTake > 0 {
			collectedRocks = append(collectedRocks, buffer.Rocks[:actualTake]...)
			buffer.Rocks = buffer.Rocks[actualTake:]
			rockCounts[i] = actualTake
			numToTake -= actualTake
		}

		// If base buffer was insufficient, try transition buffers
		if numToTake > 0 {
			fromTransition := r.takeRocksFromTransitionBuffers(numToTake)
			collectedRocks = append(collectedRocks, fromTransition...)
		}
	}

	if len(collectedRocks) == 0 {
		return // No rocks collected
	}

	// Give each collected rock movement/bounce based on index
	for i := range collectedRocks {
		rock := &collectedRocks[i]

		// Convert SpriteSlopeX/Y (0..7) back to SlopeX/Y (-4..+4)
		rock.SlopeX = rock.SpriteSlopeX + MIN_SLOPE
		rock.SlopeY = rock.SpriteSlopeY + MIN_SLOPE

		// Use index to determine bounce direction (much faster than random)
		if i%2 == 0 {
			rock.BounceY()
		} else {
			rock.BounceX()
		}
	}

	// Random transition color from base colors, weighted by how many rocks taken
	transitionColor := r.config.BaseColors[0] // Default to first base color
	if len(r.config.BaseColors) > 1 {
		// Build weighted list of colors
		totalTaken := 0
		for _, count := range rockCounts {
			totalTaken += count
		}

		if totalTaken > 0 {
			// Pick random rock index
			randomIdx := rand.Intn(totalTaken)

			// Find which buffer this rock came from
			cumulative := 0
			for i, count := range rockCounts {
				cumulative += count
				if randomIdx < cumulative {
					transitionColor = r.config.BaseColors[i]
					break
				}
			}
		}
	}

	// Create held buffer
	heldBuffer := &RockBuffer{
		Rocks:           collectedRocks,
		Color:           color,           // Die color (target)
		TransitionColor: transitionColor, // Random base color
		Transition:      r.config.ColorTransitionFrames,
		FrameCounter:    0,
	}

	r.HeldColorBuffers[dieIdentity] = heldBuffer
}

// DeselectAll returns all held rocks back to base buffers
func (r *RocksRenderer) DeselectAll() {
	for identity := range r.HeldColorBuffers {
		r.DeselectRocks(identity)
	}
}

// DeselectRocks returns rocks from a die's held buffer back to base buffers via transition buffers
func (r *RocksRenderer) DeselectRocks(dieIdentity render.DieIdentity) {
	// Get held buffer
	heldBuffer, exists := r.HeldColorBuffers[dieIdentity]
	if !exists || len(heldBuffer.Rocks) == 0 {
		return
	}

	numRocks := len(heldBuffer.Rocks)
	numBaseBuffers := len(r.config.BaseColors)

	if numBaseBuffers == 0 {
		// Safety: no base buffers, can't return rocks
		delete(r.HeldColorBuffers, dieIdentity)
		return
	}

	// Calculate distribution across all base buffers
	rocksPerBuffer := numRocks / numBaseBuffers
	remainder := numRocks % numBaseBuffers

	offset := 0

	// Create one transition buffer per base color buffer
	for i := 0; i < numBaseBuffers; i++ {
		// Calculate how many rocks go to this buffer
		numToReturn := rocksPerBuffer
		if i < remainder {
			numToReturn++ // Distribute remainder
		}

		if numToReturn == 0 {
			continue // Skip if no rocks for this buffer
		}

		endIdx := offset + numToReturn
		if endIdx > numRocks {
			endIdx = numRocks // Safety clamp
		}

		// Create transition buffer for this portion
		transitionBuffer := &RockBuffer{
			Rocks:           append([]SimpleRock{}, heldBuffer.Rocks[offset:endIdx]...),
			Color:           r.BaseColorBuffers[i].Color, // Target: this base color
			TransitionColor: heldBuffer.Color,            // Source: die color
			Transition:      r.config.ColorTransitionFrames,
			FrameCounter:    0,
		}

		r.TransitionBuffers = append(r.TransitionBuffers, transitionBuffer)
		offset = endIdx
	}

	// Remove held buffer
	delete(r.HeldColorBuffers, dieIdentity)
}

// drawBufferToImage draws all rocks from a buffer to a temporary image
func (r *RocksRenderer) drawBufferToImage(
	buffer *RockBuffer,
	tempImage *ebiten.Image,
	opts *ebiten.DrawImageOptions,
) {
	// Note: tempImage is already cleared by imagePool.GetNext()

	for _, rock := range buffer.Rocks {
		sprite := r.sprites[rock.SpriteSlopeX][rock.SpriteSlopeY]
		frameRect := sprite.SpriteSheet.Rect(int(rock.SpriteIndex))
		frameImage := sprite.Image.SubImage(frameRect).(*ebiten.Image)

		opts.GeoM.Reset()
		scale := rock.Score.SizeMultiplier()
		opts.GeoM.Scale(float64(scale), float64(scale))
		opts.GeoM.Translate(float64(rock.Position.X), float64(rock.Position.Y))
		tempImage.DrawImage(frameImage, opts)
	}
}

// drawWithColorShader applies color tint shader to a temp image and draws to screen
// Handles color transitions via shader uniforms (GPU-side mixing)
func (r *RocksRenderer) drawWithColorShader(
	buffer *RockBuffer,
	tempImage *ebiten.Image,
	screen *ebiten.Image,
) {
	// Calculate transition amount (0.0 to 1.0)
	var transitionAmount float32
	if buffer.Transition > 0 {
		// Transition counter counts down from max to 0
		// At start: Transition = max, transitionAmount = 1.0 (full TransitionColor)
		// At end: Transition = 0, transitionAmount = 0.0 (full Color)
		transitionAmount = float32(buffer.Transition) / float32(r.config.ColorTransitionFrames)
	} else {
		transitionAmount = 0.0 // No transition, use ColorFrom only
	}

	colorOpts := &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{tempImage, nil, nil, nil},
		Uniforms: map[string]interface{}{
			"ColorFrom":        buffer.Color.KageVec3(),           // Target color (where we end up)
			"ColorTo":          buffer.TransitionColor.KageVec3(), // Source color (where we start)
			"TransitionAmount": transitionAmount,                  // 1.0 at start, 0.0 at end
		},
	}
	screen.DrawRectShader(int(render.GAME_BOUNDS_X), int(render.GAME_BOUNDS_Y), r.colorShader, colorOpts)
}
