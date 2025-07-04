package main

import (
	"fmt"
	"math"

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
	Rocks        int           // how many rocks are left
	scoringIndex int           // which die is currently being scored
	ScoringHand  []*Die        // the dice that are being scored
	Hand         dice.HandRank // current hand for the level

	// State machine fields
	scoringState ScoringState // The current state of the scoring animation
	scoringTimer int          // A tick-based timer for delays
	scoringDelay int          // How many ticks to wait between scoring each die
}

func NewLevel(startRocks int) *Level {
	return &Level{
		Rocks:        startRocks,
		scoringState: SCORING_IDLE,
		scoringDelay: 15, // e.g., pause for 15 ticks (1/4 of a second at 60 TPS)
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
		l.scoringTimer--
		return // Wait until the timer is done
	}

	// --- State Machine Logic ---

	// Are we done with all the dice?
	if l.scoringIndex >= len(heldDice) {
		for _, die := range heldDice {
			die.Mode = ROLLING
			die.Roll()
		}
		l.ScoringHand = nil // Clear the hand
		l.scoringState = SCORING_IDLE
		return
	}

	die := heldDice[l.scoringIndex]

	if l.scoringState == SCORING_MOVING {
		// --- Positioning and Movement (Your existing logic) ---
		// Center the dice
		totalWidth := float64(len(heldDice)) * TileSize
		startX := (render.SCOREZONE.MaxWidth - render.SCOREZONE.MinWidth - totalWidth) / 2
		x := float64(l.scoringIndex)*TileSize + TileSize + startX
		y := render.SCOREZONE.MaxHeight - TileSize

		die.Fixed.X = x
		die.Fixed.Y = y

		// Apply velocity to move towards the fixed point
		die.Velocity.X = (die.Fixed.X - die.Vec2.X) * render.MoveFactor
		die.Velocity.Y = (die.Fixed.Y - die.Vec2.Y) * render.MoveFactor

		die.Vec2.X += die.Velocity.X
		die.Vec2.Y += die.Velocity.Y

		// --- Arrival Check ---
		// The "code smell" is still here, but its job has changed.
		// Instead of rushing to the next die, it now transitions to a pause.
		if math.Abs(die.Vec2.Y-die.Fixed.Y) < 0.01 && math.Abs(die.Vec2.X-die.Fixed.X) < 0.01 {
			// Die has arrived. Stop it and start the pause.
			die.Velocity.X = 0
			die.Velocity.Y = 0
			l.Rocks -= die.ActiveFace().Value() // Score it
			l.scoringState = SCORING_PAUSING
			l.scoringTimer = l.scoringDelay // Start the timer
		}
	} else if l.scoringState == SCORING_PAUSING {
		// The timer has finished. Move to the next die.
		l.scoringIndex++
		l.scoringState = SCORING_MOVING
	}
}

// We need to adjust the String() method as well
func (l Level) String() string {
	return fmt.Sprintf("%-4d rocks : %s %.2fx", l.Rocks, l.Hand.String(), l.Hand.Multiplier())
}
