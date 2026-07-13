package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/music"
	"github.com/ninesl/dice-will-roll/render"
)

func (g *Game) Update() error {
	g.time = float32(time.Since(g.startTime).Milliseconds()) / float32(ebiten.TPS())
	g.UpdateMusic()
	g.UpdateMouseInput()
	g.SetActiveDieIndex(g.Dice...)

	g.UpdateDice()

	g.PlayerInput()

	g.AnimateDice()
	g.AnimateRocks()

	return nil
}

func (g *Game) UpdateMusic() {
	if g.Music == nil {
		return
	}

	g.Music.Tick()
}

func DEBUGTitleFPS(x, y float32) {
	ebiten.SetWindowTitle("Dice Will Roll " +
		fmt.Sprintf("T%0.2f F%0.2f x%4.0f y%4.0f ", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y))
}

var activeDieWiggleFollowFactor float32 = 0.2
var activeDieWiggleSpeed float32 = 0.25

func (g *Game) UpdateDice() {
	g.heldDie = g.heldDie[:0]
	g.hold = g.hold[:0]

	//DEBUGTitleFPS(g.Mouse.Position.X, g.Mouse.Position.Y)
	for _, d := range g.Dice {
		if d.Mode == HELD {
			d.Height = -.5
			g.hold = append(g.hold, d.Die)
			g.heldDie = append(g.heldDie, d)
		}
	}

	//closest.DieRenderable.Velocity.X

	g.ActiveLevel.Hand = dice.DetermineHandRank(g.hold)
	g.ActiveLevel.ScoringHand = FindHandRankDice(g.heldDie, g.ActiveLevel.Hand)
	for _, die := range g.ActiveLevel.ScoringHand {
		die.Height = .1
	}
}

func (g *Game) PlayerInput() {
	action := g.Controls()
	// cant make an action if scoring
	if action != ROLLING || g.ActiveLevel.scoringState != SCORING_IDLE {
		switch action {
		case ROLL:
			if g.ActiveLevel.RollsLeft > 0 {
				g.ActiveLevel.RollsLeft--
				for _, die := range g.Dice {
					die.Roll()
				}

			} else {
				for _, die := range g.Dice {
					if die.Mode == ROLLING {
						// specific impl if roll was pressed and no more rolls
						die.ZRotation = rand.Float32() + -rand.Float32() // rotate changes
					} else {
						die.Roll()
					}
				}

				// for rockType := range render.NUM_ROCK_TYPES {
				// 	for _, rock := range g.RocksRenderer.Rocks[rockType] {
				// 		// Toggle bounce direction for rocks
				// 		if rock.SpriteIndex%2 == 1 {
				// 			rock.BounceX()
				// 		} else {
				// 			rock.BounceY()
				// 		}
				// 	}
				// }
			}
		case PRESS:
			g.Press(g.ActiveDie())
		case SELECT:
			g.Select()
		case SCORE:
			if g.ActiveLevel.HandsLeft > 0 {
				g.ActiveLevel.HandsLeft--
				g.ActiveLevel.RollsLeft = g.ActiveLevel.MaxRolls
				g.SetDiceToScore()
			}
		}
	}
}

// if no dice are given, uses g.Dice as default
//
// active die Index is the closest die to the cursor
func (g *Game) SetActiveDieIndex(dice ...*Die) {
	g.activeDieIdx, _ = g.ClosestDieToPoint(g.Mouse.Position, g.Dice...)
}

func (g *Game) ClosestDieToPoint(point render.Vec2, dice ...*Die) (int, *Die) {
	if len(g.Dice) == 0 {
		return -1, nil
	}

	idxOfClosest := 0
	dx := point.X - (dice[idxOfClosest].Vec2.X + render.HalfDieTileSize)
	dy := point.Y - (dice[idxOfClosest].Vec2.Y + render.HalfDieTileSize)
	closestDistance := dx*dx + dy*dy

	for i := 1; i < len(dice); i++ {
		dx = point.X - (dice[i].Vec2.X + render.HalfDieTileSize)
		dy = point.Y - (dice[i].Vec2.Y + render.HalfDieTileSize)
		distance := dx*dx + dy*dy
		if distance < closestDistance {
			idxOfClosest = i
			closestDistance = distance
		}
	}

	return idxOfClosest, dice[idxOfClosest]
}

var (
	rolling     = make([]*render.DieRenderable, 0)
	held        = make([]*render.DieRenderable, 0)
	moving      = make([]*render.DieRenderable, 0)
	resetting   = make([]*render.DieRenderable, 0)
	scoringDice = make([]*Die, 0)
)

func (g *Game) AnimateDice() {

	rolling = rolling[:0]
	held = held[:0]
	moving = moving[:0]
	resetting = resetting[:0]
	scoringDice = scoringDice[:0]

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		d := &die.DieRenderable

		// when logic for a d.Mode gets too complex put it in render/
		if die.Mode == ROLLING {
			// Check if this die has a Fixed position set (meaning it's resetting)
			if d.Fixed.X != 0 || d.Fixed.Y != 0 {
				resetting = append(resetting, d)
			} else {
				// Normal rolling behavior
				d.Velocity.X *= render.BounceFactor
				d.Velocity.Y *= render.BounceFactor
				d.Vec2.X += d.Velocity.X
				d.Vec2.Y += d.Velocity.Y
			}

			rolling = append(rolling, d)
		} else if die.Mode == DRAG {
			d.Fixed.X = g.Mouse.Position.X - render.HalfDieTileSize
			d.Fixed.Y = g.Mouse.Position.Y - render.HalfDieTileSize

			d.Velocity.X = (d.Fixed.X - d.Vec2.X) * render.MoveFactor
			d.Velocity.Y = (d.Fixed.Y - d.Vec2.Y) * render.MoveFactor

			d.Vec2.X += d.Velocity.X
			d.Vec2.Y += d.Velocity.Y
			d.Velocity.X = 0
			d.Velocity.Y = 0
			moving = append(moving, d)
		} else if die.Mode == HELD {
			held = append(held, d)
		} else if die.Mode == SCORING {
			scoringDice = append(scoringDice, die)
		}
	}

	// if g.ActiveDie().Fixed.X != 0 {
	//
	// }
	moving = append(moving, rolling...)

	render.HandleResettingDice(resetting)
	render.HandleMovingHeldDice(held)
	render.HandleDiceCollisions(moving)
	render.BounceAndClamp(rolling)

	g.ActiveLevel.HandleScoring(scoringDice, g.RocksRenderer, g.Music)

	// Populate dice center and velocity buffers after all dice physics are resolved
	g.diceCenterBuffer = g.diceCenterBuffer[:0]
	g.diceVelocityBuffer = g.diceVelocityBuffer[:0]
	for _, d := range g.Dice {
		// Die center positions
		g.diceCenterBuffer = append(g.diceCenterBuffer, render.Vec3{
			X: d.Vec2.X + render.HalfDieTileSize,
			Y: d.Vec2.Y + render.HalfDieTileSize,
			Z: d.ZRotation,
		})

		// Die velocities (for determining bounce direction)
		g.diceVelocityBuffer = append(g.diceVelocityBuffer, render.Vec2{
			X: d.Velocity.X,
			Y: d.Velocity.Y,
		})
	}

	g.updateActiveDieWiggle()
}

// updateActiveDieWiggle updates cursor-follow wobble for the focused die and
// lets every other die's existing wobble fade out naturally.
func (g *Game) updateActiveDieWiggle() {
	for i, die := range g.Dice {
		if i == g.activeDieIdx && g.dieUsesCursorWiggle(die) {
			g.updatePrimaryHoverSwing(die)
			die.Wiggle.ZRotationFx = 0
		} else if die.Mode == SCORING || g.dieInActiveHand(die) {
			g.updatePrimaryHoverSwing(die)
			die.Wiggle.ZRotationFx = 0
		} else {
			die.Wiggle.SwingActive = false
			die.Wiggle.SwingSpinning = false
			die.Wiggle.SwingWaitHooks = 0
			die.Wiggle.ZRotationFx *= 0.25
		}

		die.DieRenderable.ZRotation = die.Wiggle.ZRotation + sinf(g.time*activeDieWiggleSpeed)*die.Wiggle.ZRotationFx
	}
}

func (g *Game) dieInActiveHand(die *Die) bool {
	if g.ActiveLevel == nil {
		return false
	}
	for _, activeDie := range g.ActiveLevel.ScoringHand {
		if activeDie == die {
			return true
		}
	}
	return false
}

func (g *Game) updatePrimaryHoverSwing(die *Die) {
	if g.Music == nil {
		return
	}

	nowMS := g.Music.MS()
	nowRealMS := time.Since(g.startTime).Milliseconds()
	// SwingHookMS tracks this selected next hook. The selected hook is whichever
	// of LandingLane or LaneOne is coming up first, without consuming either lane.
	// primaryMS := minNonZeroMS(g.Music.UpcomingMS(music.LandingLane), g.Music.UpcomingMS(music.LaneOne))
	primaryMS := g.Music.UpcomingMS(music.LandingLane)
	if !die.Wiggle.SwingActive || nowMS < die.Wiggle.SwingLastMS {
		// First frame for this die's hover swing: seed rotation state from the
		// rendered die and remember the currently selected hook as the baseline.
		die.Wiggle.ZRotation = wrapZRotation(die.DieRenderable.ZRotation)
		die.Wiggle.SwingStart = die.Wiggle.ZRotation
		die.Wiggle.SwingLastMS = nowMS
		die.Wiggle.SwingStartRealMS = nowRealMS
		die.Wiggle.SwingHookMS = primaryMS
		die.Wiggle.SwingDurationMS = 0
		die.Wiggle.SwingWaitHooks = 0
		die.Wiggle.SwingActive = true
		die.Wiggle.SwingSpinning = false
		if die.Wiggle.SwingDir == 0 {
			die.Wiggle.SwingDir = 1
		}
		return
	}
	if primaryMS != 0 && primaryMS != die.Wiggle.SwingHookMS {
		// The selected lane hook advanced. Start a new eased segment from the
		// current rotation, then flip direction if this die was already spinning.
		die.Wiggle.SwingHookMS = primaryMS
		die.Wiggle.SwingStart = die.Wiggle.ZRotation
		die.Wiggle.SwingLastMS = nowMS
		die.Wiggle.SwingStartRealMS = nowRealMS
		if !die.Wiggle.SwingSpinning {
			die.Wiggle.SwingWaitHooks++
			if die.Wiggle.SwingWaitHooks < 1 {
				return
			}
			die.Wiggle.SwingSpinning = true
		} else {
			die.Wiggle.SwingDir *= -1
		}
		// Finish this half-spin exactly when the selected hook arrives.
		die.Wiggle.SwingDurationMS = primaryMS - nowMS
	}
	if !die.Wiggle.SwingSpinning {
		// Before the first hook starts the spin, keep time current so the first
		// elapsed step does not include all the waiting time.
		die.Wiggle.SwingLastMS = nowMS
		return
	}

	durationMS := die.Wiggle.SwingDurationMS
	if durationMS <= 0 {
		die.Wiggle.SwingLastMS = nowMS
		return
	}

	// Ease out from the segment start toward one full normalized turn. The next
	// lane hook starts a new segment and reverses SwingDir.
	elapsedMS := nowRealMS - die.Wiggle.SwingStartRealMS
	progress := clampf(float32(elapsedMS)/float32(durationMS), 0, 1)
	die.Wiggle.ZRotation = wrapZRotation(die.Wiggle.SwingStart + die.Wiggle.SwingDir*easeOutBack(progress))
}

func (g *Game) upcomingHookMS(lane music.HookLane, count uint8) int64 {
	if count == 0 {
		count = 1
	}
	if g.Music == nil || g.Music.LaneIndexes == nil {
		return 0
	}
	if lane < 0 || int(lane) >= len(g.Music.Track.Hooks) || int(lane) >= len(g.Music.LaneIndexes) {
		return 0
	}

	hooks := g.Music.Track.Hooks[lane]
	if len(hooks) == 0 {
		return 0
	}

	idx := int(g.Music.LaneIndexes[lane])
	loops := int64(0)
	for range count - 1 {
		idx++
		if idx >= len(hooks) {
			idx = 0
			loops++
		}
	}

	return hooks[idx] + loops*g.Music.DurationMS
}

// dieUsesCursorWiggle defines which focused die modes are allowed to retarget
// toward the cursor; tweak this when changing focus behavior by mode.
func (g *Game) dieUsesCursorWiggle(die *Die) bool {
	return die.Mode == ROLLING || die.Mode == HELD || (die.Mode == DRAG && g.cursorWithinDie(die))
}

// wrappedDelta returns the shortest difference between normalized rotations.
func wrappedDelta(next, current float32) float32 {
	delta := next - current
	if delta > 0.5 {
		return delta - 1
	}
	if delta < -0.5 {
		return delta + 1
	}
	return delta
}

func minNonZeroMS(a, b int64) int64 {
	if a == 0 {
		return b
	}
	if b == 0 || a < b {
		return a
	}
	return b
}

func easeOutBack(x float32) float32 {
	const c1 float32 = 1.70158
	const c3 float32 = c1 + 1

	x -= 1
	return 1 + c3*x*x*x + c1*x*x
}

func clampf(x, minValue, maxValue float32) float32 {
	if x < minValue {
		return minValue
	}
	if x > maxValue {
		return maxValue
	}
	return x
}

func wrapZRotation(rotation float32) float32 {
	for rotation >= 1 {
		rotation--
	}
	for rotation < 0 {
		rotation++
	}
	return rotation
}

// cursorWithinDie checks the cursor against the die's current screen bounds.
func (g *Game) cursorWithinDie(die *Die) bool {
	return g.Mouse.Position.X > die.Vec2.X && g.Mouse.Position.X < die.Vec2.X+render.DieTileSize && g.Mouse.Position.Y > die.Vec2.Y && g.Mouse.Position.Y < die.Vec2.Y+render.DieTileSize
}

// atan2f keeps callsites in float32 even though Go's math package uses float64.
func atan2f(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

// sinf keeps callsites in float32 even though Go's math package uses float64.
func sinf(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

// absf returns the absolute value without converting through math.Abs.
func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func (g *Game) AnimateRocks() {
	// Pass pre-computed dice collision data to renderer
	g.RocksRenderer.CollideAndAnimateRocks(g.Mouse.Position.X, g.Mouse.Position.Y, g.diceCenterBuffer, g.diceVelocityBuffer)
}
