package rocks

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

func (r *SimpleRock) Update() {
	sizeData := r.SizeData()

	// Wall bouncing
	if r.Position.X+sizeData.Size >= render.GAME_BOUNDS_X {
		r.Position.X = render.GAME_BOUNDS_X - sizeData.Size
		r.BounceX()
	} else if r.Position.X <= 0 {
		r.Position.X = 0
		r.BounceX()
	}

	if r.Position.Y+sizeData.Size >= render.GAME_BOUNDS_Y {
		r.Position.Y = render.GAME_BOUNDS_Y - sizeData.Size
		r.BounceY()
	} else if r.Position.Y <= 0 {
		r.Position.Y = 0
		r.BounceY()
	}

	r.Position.Y += BaseVelocity * float32(r.SlopeY)
	r.Position.X += BaseVelocity * float32(r.SlopeX)

	r.UpdateAnimation()
}

// updateAllBufferTransitions decrements transition counters for all buffer types
func (r *RocksRenderer) updateAllBufferTransitions() {
	// Update HeldColorBuffers transitions (stop at 0, don't go negative)
	for _, buffer := range r.HeldColorBuffers {
		if buffer.Transition > 0 {
			buffer.Transition--
		}
	}

	// Update TransitionBuffers transitions and move completed ones to the active base buffer
	for i := len(r.TransitionBuffers) - 1; i >= 0; i-- {
		buffer := r.TransitionBuffers[i]

		// Decrement transition counter (stop at 0)
		if buffer.Transition > 0 {
			buffer.Transition--
		}

		// If transition complete, move rocks to the active base buffer
		if buffer.Transition <= 0 {
			r.ActiveBaseBuffer.Rocks = append(r.ActiveBaseBuffer.Rocks, buffer.Rocks...)

			// Remove this transition buffer
			r.TransitionBuffers = append(r.TransitionBuffers[:i], r.TransitionBuffers[i+1:]...)
		}
	}
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
	filtered := make([]*RockBuffer, 0, len(r.TransitionBuffers))

	for _, buf := range r.TransitionBuffers {
		if len(buf.Rocks) > 0 {
			filtered = append(filtered, buf)
		}
	}
	r.TransitionBuffers = filtered
	return collected
}

// countTransitionRocks returns sum rocks in transition buffers/slices
// debug util
func (r *RocksRenderer) countTransitionRocks() int {
	total := 0
	for _, buf := range r.TransitionBuffers {
		total += len(buf.Rocks)
	}
	return total
}

// debug util
func (r *RocksRenderer) countHeldBuffersWithRocks() int {
	total := 0
	for _, buffer := range r.HeldColorBuffers {
		if buffer != nil && len(buffer.Rocks) > 0 {
			total++
		}
	}
	return total
}

// takeRocksFromActiveBaseBuffer takes rocks from the top (end) of the active base buffer.
func (r *RocksRenderer) takeRocksFromActiveBaseBuffer(needed int) []SimpleRock {
	if r.ActiveBaseBuffer == nil || needed <= 0 || len(r.ActiveBaseBuffer.Rocks) == 0 {
		return []SimpleRock{}
	}

	takeCount := needed
	if takeCount > len(r.ActiveBaseBuffer.Rocks) {
		takeCount = len(r.ActiveBaseBuffer.Rocks)
	}

	startIdx := len(r.ActiveBaseBuffer.Rocks) - takeCount
	collected := append([]SimpleRock{}, r.ActiveBaseBuffer.Rocks[startIdx:]...)
	r.ActiveBaseBuffer.Rocks = r.ActiveBaseBuffer.Rocks[:startIdx]

	return collected
}

var (
	rocksBeingSelected = []SimpleRock{}
)

// SelectRocksColor assigns rocks from base/transition buffers to a die's held buffer
func (r *RocksRenderer) SelectRocksColor(color render.Vec3, dieIdentity render.DieIdentity, numDice int, diePips int) {
	rocksBeingSelected = append(rocksBeingSelected,
		r.takeRocksFromTransitionBuffers(diePips)...)

	if len(rocksBeingSelected) < diePips {
		left := diePips - len(rocksBeingSelected)

		rocksBeingSelected = append(rocksBeingSelected,
			r.takeRocksFromActiveBaseBuffer(left)...)
	}

	// Give each collected rock movement/bounce based on index for psuedo-random
	for i := range rocksBeingSelected {
		rock := &rocksBeingSelected[i]

		// Convert SpriteSlopeX/Y (0..7) back to SlopeX/Y (-4..+4)
		rock.SlopeX = rock.SpriteSlopeX + MIN_SLOPE
		rock.SlopeY = rock.SpriteSlopeY + MIN_SLOPE

		// Use index to determine bounce direction (faster than random)
		if i%2 == 0 {
			rock.BounceY()
		} else {
			rock.BounceX()
		}
	}

	heldBuffer := r.ensureHeldBuffer(dieIdentity)
	heldBuffer.Rocks = append(heldBuffer.Rocks[:0], rocksBeingSelected...)
	heldBuffer.Color = color                              // Die color (target)
	heldBuffer.TransitionColor = r.ActiveBaseBuffer.Color // Random base color? or sequential
	heldBuffer.Transition = r.config.ColorTransitionFrames

	if !r.selectionOrderContains(dieIdentity) {
		r.selectionOrder = append(r.selectionOrder, dieIdentity) // Track selection order for draw order
	}

	rocksBeingSelected = rocksBeingSelected[:0]
}

// DeselectAll returns all held rocks back to base buffers
func (r *RocksRenderer) DeselectAll() {
	for identity := range r.HeldColorBuffers {
		r.DeselectRocks(identity)
	}
}

// DeselectRocks returns rocks from a die's held buffer back to base buffers via transition buffers
func (r *RocksRenderer) DeselectRocks(dieIdentity render.DieIdentity) {
	heldBuffer := r.ensureHeldBuffer(dieIdentity)
	if len(heldBuffer.Rocks) == 0 {
		return
	}

	// TODO: this is a temp check, not part of game logic
	numBaseBuffers := len(r.config.BaseColors)
	if numBaseBuffers == 0 {
		// no base buffers, can't return rocks
		return
	}

	transitionBuffer := &RockBuffer{
		Rocks:           append([]SimpleRock{}, heldBuffer.Rocks...),
		Color:           r.ActiveBaseBuffer.Color, // Target: this base color
		TransitionColor: heldBuffer.Color,         // Source: die color
		Transition:      r.config.ColorTransitionFrames,
	}

	r.TransitionBuffers = append(r.TransitionBuffers, transitionBuffer)
	r.clearHeldBuffer(dieIdentity)
}

func (r *RocksRenderer) selectionOrderContains(dieIdentity render.DieIdentity) bool {
	for _, id := range r.selectionOrder {
		if id == dieIdentity {
			return true
		}
	}
	return false
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

// drawBufferWithColorShader applies color tint shader to a temp image and draws to screen
// Handles color transitions via shader uniforms (GPU-side mixing)
func (r *RocksRenderer) drawBufferWithColorShader(
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
