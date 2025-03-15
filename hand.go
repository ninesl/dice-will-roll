package main

// This pkg is used to determine hand outcome from die.Roll().Value()

// HandRank is the type of hand played, Straight, Full House, Five of a kind, etc.
//
// Use .String() to get the name of the hand
type HandRank uint8

// Each HandRank is based on the actual value associated with it.
//
//	TWO_PAIR > NO_HAND
//	STRAIGHT_LARGE < FOUR_OF_A_KIND
const (
	UNKNOWN_HAND HandRank = iota
	NO_HAND
	HIGH_DIE
	ONE_PAIR
	SNAKE_EYES
	TWO_PAIR
	THREE_OF_A_KIND
	STRAIGHT_SMALL // 4 consecutive
	STRAIGHT_LARGE // 5 consecutive
	FULL_HOUSE
	FOUR_OF_A_KIND
	FIVE_OF_A_KIND
	// 6 die
	THREE_PAIR    // 2 + 2 + 2
	CROWDED_HOUSE // 4 + 2
	SIX_OF_A_KIND
	STRAIGHT_LARGER     // 6 consecutive
	TWO_THREE_OF_A_KIND // 3 + 3
	// 7 die
	OVERPOPULATED_HOUSE // 4 + 3
	STRAIGHT_LARGEST    // 7 consecutive
	FULLEST_HOUSE       // 5 + 2
	SEVEN_OF_A_KIND
	SEVEN_SEVENS // 7 of a kind where all Value is 7
)

// first check in determining HandRank
func isStraight(dice []Die) bool {
	if len(dice) < 4 {
		return false
	}

	//TODO: implementation

	return false
}

// TODO: unit test
//
// Calculates given slice of dice's active face's values.
//
// returns a HandRank that corresponds to the input dice
func DetermineHandRank(dice []Die) HandRank {
	var (
		numDice  = len(dice)
		valueMap = map[int]int{}
		values   = []int{} // tracks the # of values' occurences from valueMap for comparisons. A slice of the values of valueMap
	)

	// gather occurences of unique values
	for i := range numDice {
		value := dice[i].ActiveFace().Value()
		// starts from 0 bc of nil int
		valueMap[value] = valueMap[value] + 1
	}

	// used in determining hand logic
	for value := range valueMap {
		values = append(values, value)
	}

	// if is a straight, return based off
	//  switch numDice {}
	if isStraight(dice) {
		switch numDice {
		case 4:
			return STRAIGHT_SMALL
		case 5:
			return STRAIGHT_LARGE
		case 6:
			return STRAIGHT_LARGER
		case 7:
			return STRAIGHT_LARGEST
		}
	}

	// determine type of hand based off # of unique values
	//
	// can assume no straight bc of earlier check
	//
	// annoying logic for each one. comments are placed where it isn't obv after a glance
	uniqueValues := len(valueMap)

	var bottomValue, topValue int // useful placeholder for hand logic
	switch uniqueValues {
	case 1:
		switch numDice {
		case 1:
			return HIGH_DIE
		case 2:
			if valueMap[1] == numDice {
				return SNAKE_EYES
			}
			return ONE_PAIR
		case 3:
			return THREE_OF_A_KIND
		case 4:
			return FOUR_OF_A_KIND
		case 5:
			return FIVE_OF_A_KIND
		case 6:
			return SIX_OF_A_KIND
		case 7:
			if valueMap[7] == 7 {
				return SEVEN_SEVENS
			} else {
				return SEVEN_OF_A_KIND
			}
		}
	case 2:
		bottomValue = values[0]
		topValue = values[1]

		switch numDice {
		case 2: // 4 2
			return HIGH_DIE
		case 3: // 2 2 1
			if valueMap[1] == 2 {
				return SNAKE_EYES
			}
			return ONE_PAIR
		case 4: // 1 1 2 2
			return TWO_PAIR
		case 5: // 1 2 2 2 2, 1 1 2 2 2
			// 4 - 1 = 3 != 1,
			if bottomValue+1 == topValue || topValue+1 == bottomValue {
				return FULL_HOUSE
			}

			return FOUR_OF_A_KIND
		case 6: // 3 3 3 5 5 5, 1 1 2 2 2 2, 4 3 3 3 3 3,
			if bottomValue == topValue {
				return TWO_THREE_OF_A_KIND
			}
			if bottomValue+2 == topValue || topValue+2 == bottomValue {
				return CROWDED_HOUSE
			}
			return FIVE_OF_A_KIND
		case 7: // 1 1 1 2 2 2 2, 1 1 2 2 2 2 2, 1 2 2 2 2 2 2
			if bottomValue+1 == topValue || topValue+1 == bottomValue {
				return OVERPOPULATED_HOUSE
			}
			if bottomValue+3 == topValue || topValue+3 == bottomValue {
				return FULLEST_HOUSE
			}
			return SIX_OF_A_KIND
		}
	case 3:
		switch numDice {
		case 3:
			return HIGH_DIE
		case 4: // 1 1 2 3
			if valueMap[1] == 2 {
				return SNAKE_EYES
			}
			return ONE_PAIR
		case 5: // 1 1 1 2 3, 1 1 2 2 3
			if values[0] == 3 || values[1] == 3 || values[2] == 3 {
				return THREE_OF_A_KIND
			}
			return TWO_PAIR
		case 6: // 1 1 1 1 2 3, 1 1 1 2 2 3, 1 1 2 2 3 3
			if values[0] == 4 || values[1] == 4 || values[2] == 4 {
				return FOUR_OF_A_KIND
			}
			if values[0] == values[1] && values[1] == values[2] {
				return THREE_PAIR
			}
			return FULL_HOUSE
		case 7: // 1 1 1 1 1 2 3, 1 1 1 1 2 2 3, 1 1 1 2 2 2 3
			if values[0] == 5 || values[1] == 5 || values[2] == 5 {
				return FIVE_OF_A_KIND
			}
			if values[0] == values[1] || values[1] == values[2] {
				return TWO_THREE_OF_A_KIND
			}
			return FULLEST_HOUSE
		}
	case 4:
		switch numDice {
		case 4:
			return HIGH_DIE
		case 5: // 1 1 2 3 4
			if valueMap[1] == 2 {
				return SNAKE_EYES
			}
			return ONE_PAIR
		case 6: // 1 1 1 2 3 4, 1 1 2 2 3 4
			if values[0] == 3 || values[1] == 3 || values[2] == 3 || values[4] == 3 {
				return THREE_OF_A_KIND
			}
			return TWO_PAIR
		case 7: // 1 1 1 1 2 3 4, 1 1 1 2 2 3 4
			if values[0] == 4 || values[1] == 4 || values[2] == 4 || values[3] == 4 {
				return FOUR_OF_A_KIND
			}
			return FULL_HOUSE
		}
	case 5:
		switch numDice {
		case 5:
			return HIGH_DIE
		case 6: // 1 1 2 3 4 5
			if valueMap[1] == 2 {
				return SNAKE_EYES
			}
			return ONE_PAIR
		case 7: // 1 1 1 2 3 4 5, 1 1 2 2 3 4 5
			if values[0] == 3 || values[1] == 3 || values[2] == 3 || values[4] == 3 || values[5] == 3 {
				return THREE_OF_A_KIND
			}
			return TWO_PAIR
		}
	case 6:
		switch numDice {
		case 6:
			return HIGH_DIE
		case 7:
			if valueMap[1] == 2 {
				return SNAKE_EYES
			}
			return ONE_PAIR
		}
	case 7:
		return HIGH_DIE
	default: // shouldn't get here
		return UNKNOWN_HAND
	}

	return UNKNOWN_HAND
}

var (
	handRankStringMap = map[HandRank]string{
		UNKNOWN_HAND:        "Unknown Hand",
		NO_HAND:             "No Hand",
		HIGH_DIE:            "High Die",
		ONE_PAIR:            "One Pair",
		SNAKE_EYES:          "Snake Eyes",
		TWO_PAIR:            "Two Pair",
		THREE_OF_A_KIND:     "Three of a Kind",
		STRAIGHT_SMALL:      "Small Straight",
		STRAIGHT_LARGE:      "Large Straight",
		FULL_HOUSE:          "Full House",
		FOUR_OF_A_KIND:      "Four of a Kind",
		FIVE_OF_A_KIND:      "Five of a Kind",
		THREE_PAIR:          "Three Pair",
		CROWDED_HOUSE:       "Crowded House",
		SIX_OF_A_KIND:       "Six of a Kind",
		STRAIGHT_LARGER:     "Larger Straight",
		TWO_THREE_OF_A_KIND: "Three's a Crowd",
		OVERPOPULATED_HOUSE: "Overpopulated House",
		STRAIGHT_LARGEST:    "Ultra Straight",
		FULLEST_HOUSE:       "Fullest House",
		SEVEN_OF_A_KIND:     "Seven of a Kind",
		SEVEN_SEVENS:        "Lucky Sevens",
	}
)

func (h *HandRank) String() string {
	return handRankStringMap[*h]
}
