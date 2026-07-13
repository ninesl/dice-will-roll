package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/music"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/rocks"
)

// ScoringState defines the states for the scoring animation sequence.
type ScoringState uint8

const (
	SCORING_IDLE ScoringState = iota
	SCORING_WAIT
	SCORING_WAIT_FINAL
)

// A level keep track of state and scoring for an 'instance' of a run
type Level struct {
	ScoringHand  []*Die // the dice that are being scored
	Rocks        int    // how many rocks are left
	CurrentScore int    // current amount of rocks that are getting removed
	scoringIndex int    // which die is currently being scored
	MaxRolls     int    // max rolls per hand
	RollsLeft    int    // rolls left this hand
	MaxHands     int    // max hands this level
	HandsLeft    int    // hands remaining this level

	// State machine fields
	scoringState          ScoringState // The current state of the scoring animation
	scoringMoves          []scoringMove
	finalScoringHookCount uint8
	finalScoringArmed     bool

	Hand      dice.HandRank // current hand for the level
	ScoreHand dice.HandRank // current hand that will apply mult to the score
}

type scoringMove struct {
	die    *Die
	frame  int
	frames int
	startX float32
	startY float32
	endX   float32
	endY   float32
	landed bool
}

const explosionMinLeadMS = 100

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
//
// TODO: make this animation better/more fun
func (l *Level) HandleScoring(heldDice []*Die, rockRenderer *rocks.RocksRenderer, musicState *music.NowPlaying) {
	if len(heldDice) == 0 || musicState == nil || musicState.LaneIndexes == nil {
		l.scoringState = SCORING_IDLE
		return
	}

	// when a die that just became HELD it's x/y is determined on it's position from
	// where the cursor was, essentially 'slotting' it between the other dice
	sort.Slice(heldDice, func(i, j int) bool {
		return heldDice[i].ActiveFace().NumPips() < heldDice[j].ActiveFace().NumPips()
	})

	if l.scoringState == SCORING_IDLE {
		l.scoringIndex = 0
		l.scoringMoves = l.scoringMoves[:0]
		l.finalScoringArmed = false
		musicState.ResetHooks(music.LaneOne)
		l.scoringState = SCORING_WAIT
		return
	}

	switch l.scoringState {
	case SCORING_WAIT:
		l.updateScoringMoves(rockRenderer)
		if l.scoringIndex < len(heldDice) && musicState.Hook(music.LaneOne) {
			l.startScoringDie(heldDice, musicState)
			l.scoringIndex++
		}
		if l.scoringIndex < len(heldDice) || !l.allScoringMovesLanded() {
			return
		}
		l.finalScoringArmed = false
		l.scoringState = SCORING_WAIT_FINAL

	case SCORING_WAIT_FINAL:
		if !l.finalScoringArmed {
			musicState.ResetHooks(music.LaneOne)
			l.finalScoringHookCount = 1
			if upcomingHookDeltaMS(musicState, music.LaneOne, 1) < explosionMinLeadMS {
				l.finalScoringHookCount = 2
			}
			l.finalScoringArmed = true
		}
		if !musicState.Hooks(music.LaneOne, l.finalScoringHookCount) {
			return
		}
		l.finishScoring(heldDice, rockRenderer)
	}
}

func (l *Level) startScoringDie(heldDice []*Die, musicState *music.NowPlaying) {
	die := heldDice[l.scoringIndex]
	x, y := l.scoringTargetPosition(heldDice, l.scoringIndex)
	die.Fixed.X = x
	die.Fixed.Y = y

	durationMS := upcomingHookDeltaMS(musicState, music.LaneOne, 1)

	if durationMS < 1 {
		durationMS = 1
	}
	frames := int(durationMS * int64(ebiten.TPS()) / 1000)
	if frames < 1 {
		frames = 1
	}

	l.scoringMoves = append(l.scoringMoves, scoringMove{
		die:    die,
		frames: frames,
		startX: die.Vec2.X,
		startY: die.Vec2.Y,
		endX:   x,
		endY:   y,
	})
}

func (l *Level) scoringLandingMS(musicState *music.NowPlaying) int64 {
	return upcomingHookDeltaMS(musicState, music.LaneOne, 1)
}

func (l *Level) activeScoringMoveCount() uint8 {
	var count uint8
	for i := range l.scoringMoves {
		if !l.scoringMoves[i].landed {
			count++
		}
	}
	return count
}

func upcomingHookMS(musicState *music.NowPlaying, lane music.HookLane, count uint8) int64 {
	if count == 0 {
		count = 1
	}
	if musicState == nil || musicState.LaneIndexes == nil {
		return 0
	}
	if lane < 0 || int(lane) >= len(musicState.Track.Hooks) || int(lane) >= len(musicState.LaneIndexes) {
		return 0
	}

	hooks := musicState.Track.Hooks[lane]
	if len(hooks) == 0 {
		return 0
	}

	idx := int(musicState.LaneIndexes[lane])
	loops := int64(0)
	for range count - 1 {
		idx++
		if idx >= len(hooks) {
			idx = 0
			loops++
		}
	}

	return hooks[idx] + loops*musicState.DurationMS
}

func upcomingHookDeltaMS(musicState *music.NowPlaying, lane music.HookLane, count uint8) int64 {
	ms := upcomingHookMS(musicState, lane, count) - musicState.MS()
	if ms < 0 && musicState.DurationMS > 0 {
		ms += musicState.DurationMS
	}
	return ms
}

func upcomingHookMSAfter(musicState *music.NowPlaying, lane music.HookLane, count uint8, afterMS int64) int64 {
	if count == 0 {
		count = 1
	}
	if musicState == nil || musicState.LaneIndexes == nil {
		return 0
	}
	if lane < 0 || int(lane) >= len(musicState.Track.Hooks) || int(lane) >= len(musicState.LaneIndexes) {
		return 0
	}

	hooks := musicState.Track.Hooks[lane]
	if len(hooks) == 0 {
		return 0
	}

	idx := int(musicState.LaneIndexes[lane])
	loops := int64(0)
	for hooks[idx]+loops*musicState.DurationMS <= afterMS {
		idx++
		if idx >= len(hooks) {
			idx = 0
			loops++
		}
	}

	for range count - 1 {
		idx++
		if idx >= len(hooks) {
			idx = 0
			loops++
		}
	}

	return hooks[idx] + loops*musicState.DurationMS
}

func (l *Level) scoringTargetPosition(heldDice []*Die, index int) (float32, float32) {
	num := len(heldDice)
	x := float32(GAME_BOUNDS_X)/2 - render.HalfDieTileSize
	y := render.SmallRollZone.MaxHeight + render.SCOREZONE.MinHeight/2 + render.DieTileSize/5
	if num > 1 {
		x -= render.DieTileSize * (float32(num) - 1.0)
	}
	x += render.DieTileSize * 2 * float32(index)
	return x, y
}

func (l *Level) updateScoringMoves(rockRenderer *rocks.RocksRenderer) {
	for i := range l.scoringMoves {
		move := &l.scoringMoves[i]
		if move.landed {
			continue
		}

		progress := float32(move.frame) / float32(move.frames)
		if progress > 1 {
			progress = 1
		}
		progress = progress * progress * (3 - 2*progress)
		move.die.Vec2.X = move.startX + (move.endX-move.startX)*progress
		move.die.Vec2.Y = move.startY + (move.endY-move.startY)*progress
		move.frame++

		if move.frame <= move.frames {
			continue
		}

		l.landScoringDie(move, rockRenderer)
	}
}

func (l *Level) allScoringMovesLanded() bool {
	if len(l.scoringMoves) == 0 {
		return false
	}
	for i := range l.scoringMoves {
		if !l.scoringMoves[i].landed {
			return false
		}
	}
	return true
}

func (l *Level) landScoringDie(move *scoringMove, rockRenderer *rocks.RocksRenderer) {
	die := move.die
	die.Vec2.X = move.endX
	die.Vec2.Y = move.endY
	die.Velocity.X = 0
	die.Velocity.Y = 0
	l.CurrentScore += die.ActiveFace().Value()
	rockRenderer.ExplodeRocks(die.Identifier, die.ActiveFace().NumPips())
	move.landed = true
}

func (l *Level) finishScoring(heldDice []*Die, rockRenderer *rocks.RocksRenderer) {
	l.CurrentScore = int(float32(l.CurrentScore) * l.ScoreHand.Multiplier())
	l.Rocks -= l.CurrentScore

	sumOfNumPips := 0
	for _, d := range heldDice {
		sumOfNumPips += d.ActiveFace().NumPips()
	}
	remainingScore := l.CurrentScore - sumOfNumPips

	if remainingScore > 0 {
		numDice := len(heldDice)
		rocksPerDie := remainingScore / numDice
		remainder := remainingScore % numDice
		rocksByIdentity := make(map[render.DieIdentity]int, numDice)

		for _, die := range heldDice {
			if rocksPerDie > 0 {
				rocksByIdentity[die.Identifier] += rocksPerDie
			}
		}

		for i := 0; remainder > 0; i++ {
			die := heldDice[i%numDice]
			rocksByIdentity[die.Identifier]++
			remainder--
		}

		identities := make([]render.DieIdentity, 0, len(rocksByIdentity))
		for identity := range rocksByIdentity {
			identities = append(identities, identity)
		}
		sort.Slice(identities, func(i, j int) bool {
			return identities[i] < identities[j]
		})

		for _, identity := range identities {
			rockRenderer.ExplodeRocks(identity, rocksByIdentity[identity])
		}
	}

	l.CurrentScore = 0
	for _, die := range heldDice {
		die.Mode = ROLLING
		die.Roll()
		die.Velocity.Y = render.DieTileSize * 2
		die.Direction = render.DirectionArr[render.DOWN]
	}

	rockRenderer.DeselectAll()
	l.ScoringHand = l.ScoringHand[:0]
	l.scoringState = SCORING_IDLE
}

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
