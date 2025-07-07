package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

// ScoringState defines the states for the scoring animation sequence.
type ScoringState uint8

const (
	SCORING_IDLE ScoringState = iota
	SCORING_MOVING
	SCORING_PAUSING
)

// A level keep track of state and scoring for an 'instance' of a run
type Level struct {
	ScoringHand  []*Die        // the dice that are being scored
	Rocks        int           // how many rocks are left
	CurrentScore int           // current amount of rocks that are getting removed
	scoringIndex int           // which die is currently being scored
	Hand         dice.HandRank // current hand for the level
	ScoreHand    dice.HandRank // current hand that will apply mult to the score

	MaxRolls  int // max rolls per hand
	RollsLeft int // rolls left this hand
	MaxHands  int // max hands this level
	HandsLeft int // hands remaining this level

	// State machine fields
	// gets decremented/incremented every call bc it hits through a tick each time
	scoringTimer int          // A tick-based timer for delays
	scoringState ScoringState // The current state of the scoring animation

}

// TODO:FIXME: should be determined in options?
const scoringDelay int = 15 //  How many ticks to wait between scoring each die e.g., pause for 15 ticks (1/4 of a second at 60 TPS)

// parameters for a level. Used in NewLevel(levelOps)
type LevelOptions struct {
	Rocks int // number of rocks to start on this level
	Hands int // number of hands that can be scored (level specific, player)
	Rolls int // number of rolls that can be made a hand (level specific, player)
}

func NewLevel(ops LevelOptions) *Level {
	return &Level{
		Rocks:        ops.Rocks,
		MaxHands:     ops.Hands,
		HandsLeft:    ops.Hands,
		MaxRolls:     ops.Rolls,
		RollsLeft:    ops.Rolls,
		scoringState: SCORING_IDLE, // default
	}
}

// handles scoring and render changes, used in g.ActiveLevel
func (l *Level) HandleScoring(heldDice []*Die) {
	// If there are no dice to score, ensure we are idle.
	if len(heldDice) == 0 {
		l.scoringState = SCORING_IDLE
		return
	}

	// If we are idle and have dice, let's start the process.
	if l.scoringState == SCORING_IDLE {
		l.scoringIndex = 0
		l.scoringState = SCORING_MOVING
	}

	// Decrement the timer if it's running.
	if l.scoringTimer > 0 {
		// fmt.Printf("%2d %d %d\n", l.scoringTimer, len(heldDice), l.scoringIndex)
		l.scoringTimer--
		return // Wait until the timer is done
	}

	// when a die that just became HELD it's x/y is determined on it's position from
	// where the cursor was, essentially 'slotting' it between the other dice
	sort.Slice(heldDice, func(i, j int) bool {
		return heldDice[i].ActiveFace().NumPips() < heldDice[j].ActiveFace().NumPips()
	})

	// --- State Machine Logic ---

	// Are we done with all the dice?
	if l.scoringIndex >= len(heldDice) {
		if l.scoringIndex == len(heldDice) {
			l.scoringIndex++ // skips this final mult animation
			l.scoringTimer = scoringDelay * 3
			l.CurrentScore = int(float32(l.CurrentScore) * l.ScoreHand.Multiplier())
			return
		}

		l.Rocks -= l.CurrentScore
		l.CurrentScore = 0
		//
		for _, die := range heldDice {
			die.Mode = ROLLING
			die.Roll()
		}
		l.ScoringHand = l.ScoringHand[:0] // Clear the hand
		l.scoringState = SCORING_IDLE
		return
	}

	die := heldDice[l.scoringIndex]

	if l.scoringState == SCORING_MOVING {
		// positioning
		x := float64(GAME_BOUNDS_X)/2 - render.HalfTileSize
		y := render.SCOREZONE.MinHeight/2 + TileSize/5
		if len(heldDice) > 1 {
			x += TileSize * (float64(len(heldDice) - l.scoringIndex))
		}

		die.Fixed.X = x
		die.Fixed.Y = y

		// Apply velocity to move towards the fixed point
		die.Velocity.X = (die.Fixed.X - die.Vec2.X) * 0.15
		die.Velocity.Y = (die.Fixed.Y - die.Vec2.Y) * 0.15

		die.Vec2.X += die.Velocity.X
		die.Vec2.Y += die.Velocity.Y

		// --- Arrival Check ---
		// The "code smell" is still here, but its job has changed.
		// Instead of rushing to the next die, it now transitions to a pause.
		if math.Abs(die.Vec2.Y-die.Fixed.Y) < 0.05 && math.Abs(die.Vec2.X-die.Fixed.X) < 0.05 {
			// Die has arrived. Stop it and start the pause.
			die.Velocity.X = 0
			die.Velocity.Y = 0
			// l.Rocks -= die.ActiveFace().Value() // Score it

			l.CurrentScore += die.ActiveFace().Value()

			l.scoringState = SCORING_PAUSING
			l.scoringTimer = scoringDelay // Start the timer

			if l.scoringIndex == len(heldDice)-1 {
				l.scoringTimer *= 2
			}
		}
	} else if l.scoringState == SCORING_PAUSING {
		// The timer has finished. Move to the next die.
		l.scoringIndex++
		l.scoringState = SCORING_MOVING
	}
}

// We need to adjust the String() method as well
func (l Level) String() string {
	if l.ScoreHand != dice.NO_HAND {
		return fmt.Sprintf("%-2d/%2d hands | %-2d/%2d rolls | %-4d rocks | %s %.2fx | %d",
			l.HandsLeft, l.MaxHands, l.RollsLeft, l.MaxRolls, l.Rocks,
			l.ScoreHand.String(), l.ScoreHand.Multiplier(), l.CurrentScore)

	}

	return fmt.Sprintf("%-2d/%2d hands | %-2d/%2d rolls | %-4d rocks | %s %.2fx | %d",
		l.HandsLeft, l.MaxHands, l.RollsLeft, l.MaxRolls, l.Rocks,
		l.Hand.String(), l.Hand.Multiplier(), l.CurrentScore)
}
