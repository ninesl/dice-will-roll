package dice

//	 FIXME: will need to figure out how this works with rendering
//	 Score takes a set of dice, does the calculations
//
//		// returns the total of the .Score()
//		// of every .ActiveFace() from the
//		// best HandRank of the input dice
func Score(dice []Die) int {
	var total int
	rank := DetermineHandRank(dice)
	scoringDice := FindHandRankDice(rank, dice)

	// fmt.Println(DiceString(scoringDice))

	for _, d := range scoringDice {
		total += d.ActiveFace().Score()
	}

	return total
}

// takes a set of dice, determines their value and returns the int depending on the hand mult
func ScoreHand(dice []Die, hand HandRank) int {
	var total int
	for _, d := range dice {
		total += d.ActiveFace().Score()
	}
	return int(float32(total) * hand.Multiplier())
}

// HAS IMPLEMENTATION FOR TOP LEVEL DIE IN /die.go
//
// Find the hand that is associated with the given handrank.
//
// # The given handrank assumes that it is the BEST hand possible for the input dice
//
// Returns an error if not?? FIXME: not sure if this is worth doing
//
// # Returns the die that make up input handrank, assumes handrank is the best hand
func FindHandRankDice(hand HandRank, dice []Die) []Die {
	var foundDice []Die
	switch hand {
	case HIGH_DIE:
		var dieIndex, bestVal int
		for i, die := range dice {
			thisVal := die.ActiveFace().Value()
			if thisVal > bestVal {
				bestVal = thisVal
				dieIndex = i
			}
		}
		foundDice = append(foundDice, dice[dieIndex])
	case ONE_PAIR, SNAKE_EYES, THREE_OF_A_KIND, FOUR_OF_A_KIND, FIVE_OF_A_KIND, SIX_OF_A_KIND, SEVEN_OF_A_KIND, SEVEN_SEVENS:
		foundDice = findMatchingValues(dice)
	case FULL_HOUSE, CROWDED_HOUSE, TWO_PAIR, TWO_THREE_OF_A_KIND, OVERPOPULATED_HOUSE, FULLEST_HOUSE: // broken up for readability
		foundDice = findMatchingValues(dice)
	case THREE_PAIR:
		foundDice = findMatchingValues(dice)
		foundDice = filterThreePair(foundDice)
	case STRAIGHT_SMALL, STRAIGHT_LARGE, STRAIGHT_LARGER, STRAIGHT_LARGEST:
		foundDice = findBestSingleConsecutive(dice)
	case STRAIGHT_MAX:
		// TODO: impl
		// ? MustLen(len(foundDice), 7)
	case UNKNOWN_HAND, NO_HAND:
	default:
		return nil
	}

	// length := len(foundDice)
	// switch hand {
	// case HIGH_DIE:
	// 	MustLen(length, 1, fmt.Sprintf("%d %s, %d length found, expected 1", hand, hand.String(), length))
	// case ONE_PAIR, SNAKE_EYES:
	// 	MustLen(length, 2, fmt.Sprintf("%d %s, %d length found, expected 2", hand, hand.String(), length))
	// case THREE_OF_A_KIND: //TODO: FIXME:

	// 	// MustLen(length, 3, fmt.Sprintf("%d %s, %d length found, expected 3", hand, hand.String(), length))
	// case TWO_PAIR, STRAIGHT_SMALL, FOUR_OF_A_KIND:
	// 	MustLen(length, 4, fmt.Sprintf("%d %s, %d length found, expected 4", hand, hand.String(), length))
	// case STRAIGHT_LARGE, FULL_HOUSE, FIVE_OF_A_KIND:
	// 	MustLen(length, 5, fmt.Sprintf("%d %s, %d length found, expected 5", hand, hand.String(), length))
	// case SIX_OF_A_KIND, STRAIGHT_LARGER, TWO_THREE_OF_A_KIND, CROWDED_HOUSE, THREE_PAIR:
	// 	MustLen(length, 6, fmt.Sprintf("%d %s, %d length found, expected 6", hand, hand.String(), length))
	// case OVERPOPULATED_HOUSE, FULLEST_HOUSE, SEVEN_OF_A_KIND, SEVEN_SEVENS:
	// 	MustLen(length, 7, fmt.Sprintf("%d %s, %d length found, expected 7", hand, hand.String(), length))
	// default:
	// MustLen(length, -1, hand.String()+" found. ???") // insta crash
	// }

	return foundDice
}

// Calculates given slice of dice's active face's values.
//
// returns a HandRank that corresponds to the input dice
func DetermineHandRank(dice []Die) HandRank {
	var (
		numDice       = len(dice)
		valueCount    = map[int]int{}
		values        = []int{} // tracks the # of values' occurences from valueCount for comparisons. A slice of the values of valueCount
		foundStraight HandRank
		handFound     HandRank
	)

	// gather occurences of unique values
	for i := range numDice {
		value := dice[i].ActiveFace().Value()
		// starts from 0 bc of nil int
		valueCount[value] = valueCount[value] + 1
	}

	// used in determining hand logic
	for value := range valueCount {
		values = append(values, value)
	}

	// if a straight exists, assign check variable
	if numDice >= STRAIGHT_SMALL_LENGTH { // has to be at least 4. STRAIGHT_SMALL
		foundStraight = checkStraight(values)
	}

	// determine type of hand based off # of unique values
	handFound = checkHandOtherThanStraight(valueCount, values, numDice)

	// found straight vs handfound. Edge case for full houses and straights in the same hand
	if foundStraight > handFound {
		return foundStraight
	}
	return handFound
}
