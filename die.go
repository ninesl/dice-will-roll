package main

import (
	"math/rand/v2"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

// func init() {
// 	// TODO: make this a commandline arguement to have fps listener

// 	// fpsChannel := make(chan string)
// 	// go func(c chan string) {
// 	// go func() {
// 	// 	// for range c {
// 	// 	for {
// 	// 		fmt.Printf("%.2f tps / %.2f fps\n", ebiten.ActualFPS(), ebiten.ActualTPS())
// 	// 	}
// 	// }() //(fpsChannel)
// 	// old way
// 	// for {
// 	// 	select {
// 	// 	case <-c:
// 	// 	}
// 	// }
// }

type Die struct {
	image *ebiten.Image
	render.DieRenderable
	dice.Die
	Mode Action // Current mode of the die, is modified thru player Controls()
}

func SetupNewDie(color render.Vec3) *Die {
	directionX := float64(rand.IntN(2)) + 1
	directionY := float64(rand.IntN(2)) + 1
	if directionX == 2 {
		directionX = -1.0
	}
	if directionY == 2 {
		directionY = -1.0
	}

	// random position
	pos := render.Vec2{
		X: render.ROLLZONE.MinWidth + TileSize*float64(rand.IntN(6))*2.0,
		Y: render.ROLLZONE.MaxHeight/2 - render.HalfTileSize,
	}

	dieRenderable := render.DieRenderable{
		Fixed: pos,
		Vec2:  pos,
		Velocity: render.Vec2{
			X: (rand.Float64()*40 + 20),
			Y: (rand.Float64()*40 + 20),
		},
		ZRotation: rand.Float32(),
		Color:     color,
		// ColorSpot: 1 * 6,
	}
	image := ebiten.NewImage(TILE_SIZE, TILE_SIZE)

	// set pips randomly 1-9
	// values := [6]int{}
	// for i := range len(values) {
	// 	// values[i] = rand.IntN(8) + 1
	// 	values[i] = 9
	// }

	die := &Die{
		Die: dice.NewDie(6),
		// Die:           dice.New6SidedDie(values),
		image:         image,
		DieRenderable: dieRenderable,
		Mode:          ROLLING,
	}
	die.Roll()

	return die
}

// TODO: numPlayerDice is a placeholder for future impl currently controlled by NUM_PLAYER_DICE
// func SetupPlayerDice(numPlayerDice int) []*Die {
func SetupPlayerDice() []*Die {
	var dice []*Die

	// var colors = []render.Vec3{
	// 	render.Color(150, 0, 0),    // red
	// 	render.Color(175, 127, 25), // orange
	// 	render.Color(160, 160, 0),  // yellow
	// 	render.Color(0, 150, 50),   // green
	// 	render.Color(50, 50, 200),  // blue
	// 	render.Color(75, 0, 130),   // indigo
	// 	render.Color(125, 50, 183), // purple
	// }

	// NUM_PLAYER_DICE = len(colors)

	for range NUM_PLAYER_DICE { // range NUM_PLAYER_DICE {
		// dice = append(dice, SetupNewDie(colors[i]))
		dice = append(dice, SetupNewDie(render.Color(
			rand.IntN(255),
			rand.IntN(255),
			rand.IntN(255),
		)))
	}

	return dice
}

// When spacebar/roll is pressed
//
// moves die around on the screen if applicable
//
// changes the face of the die if applicable
//
// logic based on Mode
func (d *Die) Roll() {
	switch d.Mode {
	case ROLLING:
		d.Height = 0 // reset to normal height no matter where it is

		d.Die.Roll()
		dir := render.Direction(rand.IntN(2) + render.UPLEFT) // random direction
		direction := render.DirectionMap[dir]

		d.Velocity.X = TileSize * rand.Float64() * direction.X
		d.Velocity.Y = TileSize * rand.Float64() * direction.Y
		d.Direction = direction

		d.ZRotation = rand.Float32()
		// d.Height = 16.0
	case HELD:
		// they spin a lil when you roll and they're held and are in g.ActiveLevel.ScoringHand. height is changed
		if d.Height > 0.0 {
			d.ZRotation = -rand.Float32() + rand.Float32()
		}
	}
}

/*

TOP LEVEL DIE HAND CHECKS

*/

// HAS IMPLEMENTATION FOR dice.Die in dice/score.go
//
// Find the hand that is associated with the given handrank.
//
// # The given handrank assumes that it is the BEST hand possible for the input dice
//
// Returns an error if not?? FIXME: not sure if this is worth doing
//
// # Returns the die that make up input handrank, assumes handrank is the best hand
func FindHandRankDice(handDice []*Die, hand dice.HandRank) []*Die {
	var foundDice []*Die
	switch hand {
	case dice.HIGH_DIE:
		var dieIndex, bestVal int
		for i, die := range handDice {
			thisVal := die.ActiveFace().Value()
			if thisVal > bestVal {
				bestVal = thisVal
				dieIndex = i
			}
		}
		foundDice = append(foundDice, handDice[dieIndex])
	case dice.ONE_PAIR, dice.SNAKE_EYES, dice.THREE_OF_A_KIND, dice.FOUR_OF_A_KIND, dice.FIVE_OF_A_KIND,
		dice.SIX_OF_A_KIND, dice.SEVEN_OF_A_KIND, dice.SEVEN_SEVENS:

		foundDice = findMatchingValues(handDice)
	case dice.FULL_HOUSE, dice.CROWDED_HOUSE, dice.TWO_PAIR, dice.TWO_THREE_OF_A_KIND,
		dice.OVERPOPULATED_HOUSE, dice.FULLEST_HOUSE: // broken up for readability

		foundDice = findMatchingValues(handDice)
	case dice.THREE_PAIR:

		foundDice = findMatchingValues(handDice)
		foundDice = filterThreePair(foundDice)
	case dice.STRAIGHT_SMALL, dice.STRAIGHT_LARGE, dice.STRAIGHT_LARGER, dice.STRAIGHT_LARGEST:
		foundDice = findBestSingleConsecutive(handDice)
	case dice.STRAIGHT_MAX:
		// TODO: impl
		// ? MustLen(len(foundDice), 7)
	case dice.UNKNOWN_HAND, dice.NO_HAND:
	default:
		return nil
	}

	return foundDice
}

// TODO: determine if this should return more info.
// maybe meta-data/better dice info available?
// don't want to enapsulate too much of the dice logic inside itself
func trackUniqueValues(dice []*Die) map[int][]*Die {
	tracker := map[int][]*Die{}

	for _, die := range dice {
		x := die.ActiveFace().Value()
		tracker[x] = append(tracker[x], die)
	}

	return tracker
}

// TODO:FIXME: this will need to be 100% sure it's the right die being passed
// makes sure threepair is ACTUALLY THREE pairs
func filterThreePair(dice []*Die) []*Die {
	tracker := trackUniqueValues(dice)

	var collect []*Die
	for _, curValueDice := range tracker {
		collect = append(collect, bestValues(curValueDice, 1)[:2]...) // top 2 hopefully
	}

	return collect
}

// TOP LEVEL DIE IMPLEMENTATION IN die.go
//
// TODO: make this more efficient
//
// returns slice of Die from input die that share valuess.
//
//	dice values [1, 2, 1, 3, 4, 2]
//	return pairs [1, 1, 2, 2] // order not guaranteed
//
//	dice := [1, 2, 2, 2, 3]
//	return [2, 2, 2]
//
//	dice [1, 3, 2, 2, 2, 3]
//	return [3, 3, 2, 2, 2]
func findMatchingValues(dice []*Die) []*Die {
	tracker := trackUniqueValues(dice)

	var matchingValues []*Die

	for _, diceThisValue := range tracker {
		if len(diceThisValue) > 1 {
			matchingValues = append(matchingValues, diceThisValue...)
		}
	}
	return matchingValues
}

// TOP LEVEL DIE IMPLEMENTATION IN /die.go
//
// returns straight with the BEST values for the conesecutive.
//
// # for modifiers, etc the tie breaker is ALWAYS the true number of .pips on the die.
//
// The dice given MUST be a straight
func findBestSingleConsecutive(checkDice []*Die) []*Die {
	tracker := trackUniqueValues(checkDice)

	// trackedLen := len(tracker)

	// if trackedLen < STRAIGHT_SMALL_LENGTH { // this check might not be needed bc dice WILL be straights when they get here
	// 	return []Die{} // explicitly empty
	// }

	// going from the top gets best straight - but idk how to do it smartly
	// dont love this. could be found in trackUniqueValues but would be wasted elsewhere?
	var topValue int
	for value := range tracker {
		if value > topValue {
			topValue = value
		}
	}

	var inARow int = 1
	for i := topValue; i > 0; i -= 1 {
		if len(tracker[i-1]) > 0 { // if there is dice below our current Value, 1 more in a row=
			inARow += 1
			if i > topValue {
				topValue = i - 1
			}
		} else {
			if inARow < dice.STRAIGHT_SMALL_LENGTH {
				topValue = 0 // set to 0 for topval find
				inARow = 0   // resets to 0. when going from a blank index it'll still add 1, so it's 0 to 1 in a row from a new found number
			} else {
				// could assume that there will never be a sequence once one is found
				// TODO: would not work with 1-3 straight or more than 7 dice
				break
			}
		}
	}

	var sequenceDice []*Die
	for i := topValue - inARow + 1; i <= topValue; i++ {
		var die *Die
		if len(tracker[i]) > 1 {
			die = bestValues(tracker[i], 1)[0]
		} else {
			die = tracker[i][0]
		}
		sequenceDice = append(sequenceDice, die)
	}

	// probably the most innefficient way to check for straights.
	//TODO: make this better. it just stinks
	return sequenceDice
}

// TODO: clean this up. refactor etc
//
// returns the X values with the most .Value() of input die.
//
//	dice [1, 1, 2, 2] x = 1
//	return [2, 2]
//	dice [1, 1, 2, 2, 2, 3, 3] x = 2
//	return [2, 2, 2, 3, 3] // order not guaranteed
func bestValues(dice []*Die, x int) []*Die {
	var uniqueValues []int
	var bestValues []*Die

	for _, die := range dice {
		pips := die.ActiveFace().NumPips()
		if !slices.Contains(uniqueValues, pips) {
			uniqueValues = append(uniqueValues, pips)
		}
	}

	sort.Slice(uniqueValues, func(i, j int) bool {
		return uniqueValues[i] < uniqueValues[j]
	})

	uniqueValues = uniqueValues[:x] // 0 - x exclusive

	for _, die := range dice {
		if slices.Contains(uniqueValues, die.ActiveFace().NumPips()) { // TODO: figure out if this should be numpip or value
			bestValues = append(bestValues, die)
		}
	}

	return bestValues
}
