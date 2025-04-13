package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// GeneratePermutations generates all permutations of numbers 1 to x, each of length n
func generatePermutations(x, n int) [][]int {
	result := [][]int{}
	current := make([]int, n)

	// Call helper function to generate permutations recursively
	generateHelper(x, n, 0, current, &result)

	return result
}

// Helper function for recursive generation
func generateHelper(x, n, position int, current []int, result *[][]int) {
	// Base case: if we've filled all positions
	if position == n {
		// Create a copy of current permutation and add to result
		permutation := make([]int, n)
		copy(permutation, current)
		*result = append(*result, permutation)
		return
	}

	// Try each possible value (1 to x) at the current position
	for i := 1; i <= x; i++ {
		current[position] = i
		generateHelper(x, n, position+1, current, result)
	}
}

// takes a slice of ints, returns a slice of Die where
//
//	die[i].ActivateFace().Value() == values[i]
func generateDiceValues(values []int, maxValue int) []Die {
	dice := BlankDiceRange(len(values), maxValue)

	for i := 0; i < len(dice); i += 1 {
		dice[i].activeFace = values[i] - 1 // 0 index
	}

	return dice
}

func randDiceHand(x, n int) {
	perms := generatePermutations(x, n)
	index := rand.Intn(len(perms))
	values := perms[index]
	dice := generateDiceValues(values, x)

	// log.Println(values)
	for i := 0; i < len(dice); i++ {
		fmt.Print(dice[i].ActiveFace().Value(), " ")
	}
	fmt.Print("\n")

	handRank := DetermineHandRank(dice)
	fmt.Println(handRankStringMap[handRank] + "\n")
}

//	func TestSetRandomDice(t *testing.T) {
//		randDiceHand(6, 1)
//		randDiceHand(6, 2)
//		randDiceHand(6, 3)
//		randDiceHand(6, 4)
//		randDiceHand(6, 5)
//		randDiceHand(6, 6)
//		randDiceHand(7, 7)
//
// Helper function to extract the numerical values from a slice of Die
func extractDiceValues(dice []Die) []int {
	values := make([]int, len(dice))
	for i, d := range dice {
		values[i] = d.ActiveFace().Value() // Or NumPips() if that's more appropriate for comparison
	}
	return values
}

// Helper function to compare expected dice values with actual returned dice, ignoring order.
func compareDiceSlicesUnordered(t *testing.T, gotDice []Die, wantValues []int, testName string, inputValues []int, hand HandRank) {
	t.Helper() // Marks this as a helper function for better error reporting

	gotValues := extractDiceValues(gotDice)

	// Sort both slices for comparison
	sort.Ints(gotValues)
	sort.Ints(wantValues)

	if !reflect.DeepEqual(gotValues, wantValues) {
		t.Errorf("%s: FindHandRankDice(%s, %v) returned dice with values %v; want values %v",
			testName, hand.String(), inputValues, gotValues, wantValues)
	}

	// Additionally, check if the length assertion in FindHandRankDice would pass
	// This requires manually implementing the length checks from FindHandRankDice
	// This part is optional but can help diagnose issues related to the MustLen checks
	expectedLen := -1 // Default for unknown
	switch hand {
	case HIGH_DIE:
		expectedLen = 1
	case ONE_PAIR, SNAKE_EYES:
		expectedLen = 2
	case THREE_OF_A_KIND:
		expectedLen = 3
	case TWO_PAIR, STRAIGHT_SMALL, FOUR_OF_A_KIND:
		expectedLen = 4
	case STRAIGHT_LARGE, FULL_HOUSE, FIVE_OF_A_KIND:
		expectedLen = 5
	case SIX_OF_A_KIND, STRAIGHT_LARGER, TWO_THREE_OF_A_KIND, CROWDED_HOUSE, THREE_PAIR:
		expectedLen = 6
	case OVERPOPULATED_HOUSE, FULLEST_HOUSE, SEVEN_OF_A_KIND, SEVEN_SEVENS:
		expectedLen = 7
		// Add STRAIGHT_LARGEST, STRAIGHT_MAX if needed
	}

	if expectedLen != -1 && len(gotDice) != expectedLen {
		// This check happens *after* the value comparison, primarily to indicate
		// if MustLen *would* have failed inside the original function based on the dice found.
		// The test already failed above if the values were wrong.
		// If the values *were* correct but the length is wrong, this adds context.
		if reflect.DeepEqual(gotValues, wantValues) { // Only log this extra error if values matched but length is wrong
			t.Errorf("%s: FindHandRankDice(%s, %v) returned correct values %v but wrong count %d; expected count %d for this hand rank",
				testName, hand.String(), inputValues, gotValues, len(gotDice), expectedLen)
		}
	}
}

// }

// Main test for all basic hand types
func TestDetermineHandRank(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		maxPips    int
		expected   HandRank
	}{
		// Single die
		{"HIGH_DIE with single die", []int{6}, 6, HIGH_DIE},

		// Two dice
		{"ONE_PAIR with two identical dice", []int{3, 3}, 6, ONE_PAIR},
		{"SNAKE_EYES with two ones", []int{1, 1}, 6, SNAKE_EYES},
		{"HIGH_DIE with different dice", []int{2, 5}, 6, HIGH_DIE},

		// Three dice
		{"THREE_OF_A_KIND with three identical dice", []int{4, 4, 4}, 6, THREE_OF_A_KIND},
		{"ONE_PAIR with one pair", []int{2, 2, 5}, 6, ONE_PAIR},
		{"SNAKE_EYES with 1-1-x", []int{1, 1, 5}, 6, SNAKE_EYES},
		{"HIGH_DIE with all different", []int{2, 3, 6}, 6, HIGH_DIE},

		// Four dice
		{"FOUR_OF_A_KIND with four identical dice", []int{2, 2, 2, 2}, 6, FOUR_OF_A_KIND},
		{"THREE_OF_A_KIND with three of a kind", []int{3, 3, 3, 5}, 6, THREE_OF_A_KIND},
		{"TWO_PAIR with two pairs", []int{1, 1, 4, 4}, 6, TWO_PAIR},
		{"STRAIGHT_SMALL with 1-2-3-4", []int{1, 2, 3, 4}, 6, STRAIGHT_SMALL},
		{"ONE_PAIR with one pair", []int{5, 5, 2, 3}, 6, ONE_PAIR},
		{"HIGH_DIE with no patterns", []int{1, 3, 5, 6}, 6, HIGH_DIE},

		// Five dice
		{"FIVE_OF_A_KIND with five identical dice", []int{3, 3, 3, 3, 3}, 6, FIVE_OF_A_KIND},
		{"FOUR_OF_A_KIND with four identical", []int{5, 5, 5, 5, 2}, 6, FOUR_OF_A_KIND},
		{"FULL_HOUSE with three and two", []int{1, 1, 1, 4, 4}, 6, FULL_HOUSE},
		{"STRAIGHT_LARGE with 1-2-3-4-5", []int{1, 2, 3, 4, 5}, 6, STRAIGHT_LARGE},
		{"THREE_OF_A_KIND with three", []int{2, 2, 2, 3, 6}, 6, THREE_OF_A_KIND},
		{"TWO_PAIR with two pairs", []int{3, 3, 5, 5, 6}, 6, TWO_PAIR},
		{"ONE_PAIR with one pair", []int{4, 4, 1, 2, 6}, 6, ONE_PAIR},

		// Six dice
		{"SIX_OF_A_KIND with six identical dice", []int{2, 2, 2, 2, 2, 2}, 6, SIX_OF_A_KIND},
		{"CROWDED_HOUSE with four and two", []int{4, 4, 4, 4, 1, 1}, 6, CROWDED_HOUSE},
		{"STRAIGHT_LARGER with 1-2-3-4-5-6", []int{1, 2, 3, 4, 5, 6}, 6, STRAIGHT_LARGER},
		{"TWO_THREE_OF_A_KIND with two sets of three", []int{2, 2, 2, 5, 5, 5}, 6, TWO_THREE_OF_A_KIND},
		{"THREE_PAIR with three pairs", []int{1, 1, 3, 3, 6, 6}, 6, THREE_PAIR},
		{"FOUR_OF_A_KIND with four identical", []int{3, 3, 3, 3, 4, 5}, 6, FOUR_OF_A_KIND},
		{"FULL_HOUSE with three and two", []int{1, 1, 1, 6, 6, 3}, 6, FULL_HOUSE},

		// Seven dice
		{"SEVEN_OF_A_KIND with seven identical dice", []int{4, 4, 4, 4, 4, 4, 4}, 6, SEVEN_OF_A_KIND},
		{"SEVEN_SEVENS with all sevens", []int{7, 7, 7, 7, 7, 7, 7}, 7, SEVEN_SEVENS},
		{"FULLEST_HOUSE with five and two", []int{3, 3, 3, 3, 3, 2, 2}, 6, FULLEST_HOUSE},
		{"OVERPOPULATED_HOUSE with four and three", []int{5, 5, 5, 5, 2, 2, 2}, 6, OVERPOPULATED_HOUSE},
		{"STRAIGHT_LARGEST with 1-2-3-4-5-6-7", []int{1, 2, 3, 4, 5, 6, 7}, 7, STRAIGHT_LARGEST},
		{"SIX_OF_A_KIND with extra", []int{3, 3, 3, 3, 3, 3, 5}, 6, SIX_OF_A_KIND},
		{"FIVE_OF_A_KIND with extras", []int{1, 1, 1, 1, 1, 4, 6}, 6, FIVE_OF_A_KIND},

		// Duplicate values - the straight should still be detected
		{"4-straight with duplicate", []int{1, 2, 2, 3, 4}, 6, STRAIGHT_SMALL},
		{"5-straight with duplicate", []int{1, 2, 3, 3, 4, 5}, 6, STRAIGHT_LARGE},
		{"6-straight with duplicate", []int{1, 2, 3, 4, 4, 5, 6}, 6, STRAIGHT_LARGER},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dice := generateDiceValues(tc.diceValues, tc.maxPips)
			got := DetermineHandRank(dice)
			if got != tc.expected {
				t.Errorf("DetermineHandRank(%v) = %s; want %s",
					tc.diceValues, got.String(), tc.expected.String())
			}
		})

	}
}

// Test specifically for straight detection
func TestStraightDetection(t *testing.T) {
	tests := []struct {
		name     string
		values   []int
		expected HandRank
	}{
		// Basic straights
		{"4-straight basic", []int{1, 2, 3, 4}, STRAIGHT_SMALL},
		{"5-straight basic", []int{2, 3, 4, 5, 6}, STRAIGHT_LARGE},
		{"6-straight basic", []int{1, 2, 3, 4, 5, 6}, STRAIGHT_LARGER},
		{"7-straight basic", []int{1, 2, 3, 4, 5, 6, 7}, STRAIGHT_LARGEST},

		// Out of order
		{"4-straight out of order", []int{4, 2, 1, 3}, STRAIGHT_SMALL},
		{"5-straight out of order", []int{5, 3, 1, 4, 2}, STRAIGHT_LARGE},
		{"6-straight out of order", []int{6, 4, 2, 1, 5, 3}, STRAIGHT_LARGER},
		{"7-straight out of order", []int{4, 7, 1, 6, 2, 5, 3}, STRAIGHT_LARGEST},

		// Non-straights
		{"Non-consecutive values", []int{1, 3, 5, 7}, NO_HAND},
		{"Almost 4-straight", []int{1, 2, 4, 5}, NO_HAND},
		{"Two pairs of consecutive values", []int{1, 2, 4, 5}, NO_HAND},
		{"Three consecutive only", []int{3, 4, 5}, NO_HAND},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := checkStraight(tc.values)
			if got != tc.expected {
				t.Errorf("checkStraight(%v) = %s; want %s",
					tc.values, got.String(), tc.expected.String())
			}
		})
	}
}

// Test for complex hands with competing patterns
func TestComplexHandInteractions(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		maxPips    int
		expected   HandRank
	}{
		// Cases where both straight and other patterns exist together
		{"STRAIGHT_SMALL beats THREE_OF_A_KIND",
			[]int{1, 2, 3, 4, 3, 3}, 6, STRAIGHT_SMALL},

		{"STRAIGHT_SMALL beats THREE_OF_A_KIND",
			[]int{1, 2, 3, 4, 4, 4}, 6, STRAIGHT_SMALL},

		// Precedence between straight types
		{"STRAIGHT_LARGE wins over STRAIGHT_SMALL",
			[]int{1, 2, 3, 4, 5, 1, 2}, 6, STRAIGHT_LARGE},

		{"STRAIGHT_LARGER wins over STRAIGHT_LARGE",
			[]int{1, 2, 3, 4, 5, 6, 1}, 6, STRAIGHT_LARGER},

		{"STRAIGHT_LARGEST wins over all other straights",
			[]int{1, 2, 3, 4, 5, 6, 7}, 7, STRAIGHT_LARGEST},

		// Special cases with straights
		{"STRAIGHT_LARGE with duplicate values",
			[]int{1, 2, 3, 4, 5, 5, 5}, 6, STRAIGHT_LARGE},

		{"STRAIGHT_SMALL with higher duplicates",
			[]int{1, 2, 3, 4, 6, 6, 6}, 6, STRAIGHT_SMALL},

		// Edge cases
		{"Discontinuous not a straight", []int{1, 2, 3, 5, 6, 7}, 7, HIGH_DIE},
		{"STRAIGHT_SMALL in larger set", []int{1, 2, 3, 4, 6, 7}, 7, STRAIGHT_SMALL},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dice := generateDiceValues(tc.diceValues, tc.maxPips)
			got := DetermineHandRank(dice)
			if got != tc.expected {
				t.Errorf("Complex hand test failed for %v: got %s, want %s",
					tc.diceValues, got.String(), tc.expected.String())

				// Debug output to help diagnose issues
				straightResult := checkStraight(tc.diceValues)
				valueCount := map[int]int{}
				for _, val := range tc.diceValues {
					valueCount[val]++
				}
				values := make([]int, 0, len(valueCount))
				for v := range valueCount {
					values = append(values, v)
				}
				otherResult := checkHandOtherThanStraight(valueCount, values, len(tc.diceValues))

				t.Logf("Straight check: %s, Other check: %s",
					straightResult.String(), otherResult.String())
			}
		})
	}
}

// Test specifically for straight vs other patterns
func TestStraightVsOtherPatterns(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		maxPips    int
		expected   HandRank
	}{
		{"FOUR_OF_A_KIND beats FOUR_OF_A_KIND",
			[]int{1, 2, 3, 4, 2, 2, 2}, 6, FOUR_OF_A_KIND},

		{"STRAIGHT_SMALL beats THREE_OF_A_KIND",
			[]int{3, 3, 3, 1, 2, 4}, 6, STRAIGHT_SMALL},

		{"STRAIGHT_LARGER beats THREE_OF_A_KIND",
			[]int{1, 2, 3, 4, 5, 6, 6}, 6, STRAIGHT_LARGER},

		{"STRAIGHT_LARGER beats extra values",
			[]int{1, 2, 3, 4, 5, 6, 3}, 6, STRAIGHT_LARGER},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dice := generateDiceValues(tc.diceValues, tc.maxPips)
			got := DetermineHandRank(dice)
			if got != tc.expected {
				t.Errorf("Straight vs pattern test failed for %v: got %s, want %s",
					tc.diceValues, got.String(), tc.expected.String())
			}
		})
	}
}

// Test for all possible 7-dice hands
func TestSevenDiceHands(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		maxPips    int
		expected   HandRank
	}{
		// 7-dice specific hands
		{"SEVEN_OF_A_KIND", []int{3, 3, 3, 3, 3, 3, 3}, 6, SEVEN_OF_A_KIND},
		{"SEVEN_SEVENS special", []int{7, 7, 7, 7, 7, 7, 7}, 7, SEVEN_SEVENS},
		{"FULLEST_HOUSE (5+2)", []int{2, 2, 4, 4, 4, 4, 4}, 6, FULLEST_HOUSE},
		{"OVERPOPULATED_HOUSE (4+3)", []int{1, 1, 1, 1, 6, 6, 6}, 6, OVERPOPULATED_HOUSE},
		{"STRAIGHT_LARGEST", []int{1, 2, 3, 4, 5, 6, 7}, 7, STRAIGHT_LARGEST},

		// Other valid 7-dice hands
		{"SIX_OF_A_KIND with extra", []int{2, 2, 2, 2, 2, 2, 5}, 6, SIX_OF_A_KIND},
		{"FIVE_OF_A_KIND with extras", []int{3, 3, 3, 3, 3, 1, 2}, 6, FIVE_OF_A_KIND},
		{"CROWDED_HOUSE (4+2+1)", []int{4, 4, 4, 4, 5, 5, 6}, 6, CROWDED_HOUSE},
		{"TWO_THREE_OF_A_KIND (3+3+1)", []int{2, 2, 2, 5, 5, 5, 6}, 6, TWO_THREE_OF_A_KIND},
		{"THREE_PAIR with single (2+2+2+1)", []int{1, 1, 3, 3, 5, 5, 6}, 6, THREE_PAIR},
		{"STRAIGHT_SMALL beats FOUR_OF_A_KIND", []int{1, 2, 3, 4, 4, 4, 4}, 6, FOUR_OF_A_KIND},
		{"FULL_HOUSE with extras", []int{5, 5, 5, 6, 6, 1, 2}, 6, FULL_HOUSE},
		{"STRAIGHT_LARGER with pair", []int{1, 2, 3, 4, 5, 6, 6}, 6, STRAIGHT_LARGER},
		{"STRAIGHT_LARGER with different dice", []int{1, 2, 3, 4, 5, 6, 1}, 6, STRAIGHT_LARGER},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dice := generateDiceValues(tc.diceValues, tc.maxPips)
			got := DetermineHandRank(dice)
			if got != tc.expected {
				t.Errorf("7-dice test failed for %v: got %s, want %s",
					tc.diceValues, got.String(), tc.expected.String())
			}
		})
	}
}

// Test for SNAKE_EYES special case
func TestSnakeEyes(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		expected   HandRank
	}{
		{"SNAKE_EYES basic", []int{1, 1}, SNAKE_EYES},
		{"SNAKE_EYES with extra die", []int{1, 1, 3}, SNAKE_EYES},
		{"ONE_PAIR that's not snake eyes", []int{2, 2}, ONE_PAIR},
		{"STRAIGHT_SMALL beats SNAKE_EYES", []int{1, 1, 2, 3, 4}, STRAIGHT_SMALL},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dice := generateDiceValues(tc.diceValues, 6)
			got := DetermineHandRank(dice)
			if got != tc.expected {
				t.Errorf("Snake Eyes test failed for %v: got %s, want %s",
					tc.diceValues, got.String(), tc.expected.String())
			}
		})
	}
}

// Random test cases to help find edge cases or bugs
// func TestRandomHandCombinations(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping extended random testing in short mode")
// 	}

// 	seed := time.Now().UnixNano()
// 	rng := rand.New(rand.NewSource(seed))

// 	testCases := 1000
// 	maxDice := 7
// 	maxPips := 7

// 	for i := 0; i < testCases; i++ {
// 		numDice := rng.Intn(maxDice) + 1 // 1 to 7 dice
// 		values := make([]int, numDice)

// 		for j := 0; j < numDice; j++ {
// 			values[j] = rng.Intn(maxPips) + 1 // 1 to 7 pips
// 		}

// 		dice := generateDiceValues(values, maxPips)
// 		handRank := DetermineHandRank(dice)

// 		// Verify we don't get UNKNOWN_HAND
// 		if handRank == UNKNOWN_HAND {
// 			t.Errorf("Got UNKNOWN_HAND for dice values: %v", values)
// 		}

// 		// Verify hand makes sense for dice count
// 		t.Run(fmt.Sprintf("Random test %d", i), func(t *testing.T) {
// 			if numDice < 7 && handRank == SEVEN_OF_A_KIND {
// 				t.Errorf("Got SEVEN_OF_A_KIND with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 7 && handRank == SEVEN_SEVENS {
// 				t.Errorf("Got SEVEN_SEVENS with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 6 && handRank == SIX_OF_A_KIND {
// 				t.Errorf("Got SIX_OF_A_KIND with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 5 && handRank == FIVE_OF_A_KIND {
// 				t.Errorf("Got FIVE_OF_A_KIND with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 5 && handRank == STRAIGHT_LARGE {
// 				t.Errorf("Got STRAIGHT_LARGE with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 6 && handRank == STRAIGHT_LARGER {
// 				t.Errorf("Got STRAIGHT_LARGER with only %d dice: %v", numDice, values)
// 			}
// 			if numDice < 7 && handRank == STRAIGHT_LARGEST {
// 				t.Errorf("Got STRAIGHT_LARGEST with only %d dice: %v", numDice, values)
// 			}
// 		})
// 	}

// 	t.Logf("Completed %d random test cases with seed %d", testCases, seed)
// }

// Debug helper function - useful for test diagnosis
func debugHandAnalysis(t *testing.T, values []int) {
	maxPips := 0
	for _, v := range values {
		if v > maxPips {
			maxPips = v
		}
	}
	if maxPips < 6 {
		maxPips = 6
	}

	dice := generateDiceValues(values, maxPips)

	// Track unique values and their counts
	valueCount := map[int]int{}
	for _, v := range values {
		valueCount[v]++
	}

	t.Logf("Input values: %v", values)
	t.Logf("Value counts: %v", valueCount)

	// Check for straights
	straightResult := checkStraight(values)
	t.Logf("Straight check result: %s", straightResult.String())

	// Check for other hands
	valuesForOtherHands := []int{}
	for v := range valueCount {
		valuesForOtherHands = append(valuesForOtherHands, v)
	}
	otherHandResult := checkHandOtherThanStraight(valueCount, valuesForOtherHands, len(values))
	t.Logf("Other hand check result: %s", otherHandResult.String())

	// Final determination
	finalResult := DetermineHandRank(dice)
	t.Logf("Final hand determination: %s", finalResult.String())
}

// TestFindHandRankDice_MatchingValues tests the FindHandRankDice function
// specifically for hand ranks determined primarily by finding matching dice values.
// It ASSUMES the inputHandRank is the CORRECT rank previously determined for the inputDiceValues.
func TestFindHandRankDice_MatchingValues(t *testing.T) {
	tests := []struct {
		name               string
		inputHandRank      HandRank // Assumed correct rank for the dice
		inputDiceValues    []int    // Dice that result in inputHandRank
		maxPips            int
		expectedDiceValues []int // EXACT dice values making up the inputHandRank
	}{
		{"ONE_PAIR", ONE_PAIR, []int{1, 3, 3, 4, 5}, 6, []int{3, 3}},
		// --- Single Group Matches ---
		{"SNAKE_EYES", SNAKE_EYES, []int{1, 1, 3, 4, 5}, 6, []int{1, 1}},
		{"THREE_OF_A_KIND", THREE_OF_A_KIND, []int{3, 3, 3, 1, 5}, 6, []int{3, 3, 3}},
		{"FOUR_OF_A_KIND", FOUR_OF_A_KIND, []int{4, 4, 4, 4, 1}, 6, []int{4, 4, 4, 4}},
		{"FIVE_OF_A_KIND", FIVE_OF_A_KIND, []int{5, 5, 5, 5, 5, 2}, 6, []int{5, 5, 5, 5, 5}},
		{"SIX_OF_A_KIND", SIX_OF_A_KIND, []int{2, 2, 2, 2, 2, 2, 3}, 6, []int{2, 2, 2, 2, 2, 2}},
		{"SEVEN_OF_A_KIND", SEVEN_OF_A_KIND, []int{1, 1, 1, 1, 1, 1, 1}, 6, []int{1, 1, 1, 1, 1, 1, 1}},
		{"SEVEN_SEVENS special", SEVEN_SEVENS, []int{7, 7, 7, 7, 7, 7, 7}, 7, []int{7, 7, 7, 7, 7, 7, 7}},

		// --- Multiple Group Matches ---
		// findMatchingValues should return *all* dice involved in these specific hands.
		{"TWO_PAIR", TWO_PAIR, []int{2, 2, 4, 4, 5}, 6, []int{2, 2, 4, 4}},
		{"THREE_PAIR", THREE_PAIR, []int{1, 1, 3, 3, 5, 5, 6}, 6, []int{1, 1, 3, 3, 5, 5}},

		// Full House: Uses findMatchingValues then bestValues(..., 2).
		// Assuming bestValues correctly identifies the two groups (or doesn't incorrectly filter them),
		// it should return the dice comprising the 3-of-a-kind and the pair.
		{"FULL_HOUSE (3+2)", FULL_HOUSE, []int{3, 3, 3, 5, 5}, 6, []int{3, 3, 3, 5, 5}},
		// {"FULL_HOUSE (3+2) with larger pairs", FULL_HOUSE, []int{1, 1, 1, 3, 3, 6, 6}, 7, []int{1, 1, 1, 6, 6}}, // Expect only the FH dice
		// can skip because it ends up being TWO_THREE_PAIR anyway

		{"CROWDED_HOUSE (4+2)", CROWDED_HOUSE, []int{4, 4, 4, 4, 1, 1, 3}, 6, []int{4, 4, 4, 4, 1, 1}},
		{"TWO_THREE_OF_A_KIND (3+3)", TWO_THREE_OF_A_KIND, []int{2, 2, 2, 5, 5, 5, 1}, 6, []int{2, 2, 2, 5, 5, 5}},
		{"OVERPOPULATED_HOUSE (4+3)", OVERPOPULATED_HOUSE, []int{5, 5, 5, 5, 2, 2, 2}, 7, []int{5, 5, 5, 5, 2, 2, 2}},
		{"FULLEST_HOUSE (5+2)", FULLEST_HOUSE, []int{3, 3, 3, 3, 3, 2, 2}, 6, []int{3, 3, 3, 3, 3, 2, 2}},

		// --- Edge Cases (that are valid scenarios) ---
		// Case: Hand is ONE_PAIR, but includes a higher non-matching die
		{"ONE_PAIR with higher single", ONE_PAIR, []int{2, 2, 6}, 6, []int{2, 2}},

		// --- Cases where findMatchingValues needs filtering (implicitly done by MustLen? Or should findMatchingValues be smarter?)
		// Q: If the hand is ONE_PAIR, but dice are [2, 2, 4, 4, 5] (actual best hand TWO_PAIR), what happens?
		// A: This scenario is invalid based on the user requirement. We assume inputHandRank is TWO_PAIR.
		// Q: If the hand is THREE_OF_A_KIND, but dice are [3, 3, 3, 5, 5] (actual best hand FULL_HOUSE), what happens?
		// A: Invalid scenario. We assume inputHandRank is FULL_HOUSE.

		// Therefore, tests where findMatchingValues returns *more* dice than the specific hand requires (e.g., returning two pairs when only one is asked for) are removed,
		// because FindHandRankDice will only be called with the *correct*, highest-ranking hand.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare input dice
			dice := generateDiceValues(tc.inputDiceValues, tc.maxPips)

			// Verify our assumption: DetermineHandRank should yield tc.inputHandRank
			// This is a sanity check for the test case itself.
			determinedRank := DetermineHandRank(dice)
			if determinedRank != tc.inputHandRank {
				// If this fails, the test case setup is wrong. The inputDiceValues
				// don't actually produce the inputHandRank.
				// Use debugHandAnalysis for complex cases if needed.
				// debugHandAnalysis(t, tc.inputDiceValues) // Uncomment to debug test case setup
				t.Fatalf("Test Case Setup Error for '%s': DetermineHandRank(%v) returned %s, but test expected %s. Adjust inputDiceValues or inputHandRank.",
					tc.name, tc.inputDiceValues, determinedRank.String(), tc.inputHandRank.String())
			}

			// Call the function under test with the *correct* hand rank
			foundDice := FindHandRankDice(tc.inputHandRank, dice)

			// Compare the values of the returned dice (order-independent)
			compareDiceSlicesUnordered(t, foundDice, tc.expectedDiceValues, tc.name, tc.inputDiceValues, tc.inputHandRank)
		})
	}
}

// TestFindBestSingleConsecutive tests the findBestSingleConsecutive function directly.
// It checks if the function correctly identifies and returns the dice forming the
// highest-value consecutive sequence of at least STRAIGHT_SMALL_LENGTH.
func TestFindBestSingleConsecutive(t *testing.T) {
	tests := []struct {
		name               string
		inputDiceValues    []int // Dice values to feed into the function
		maxPips            int
		expectedDiceValues []int // EXACT dice values expected for the *best* straight found
	}{
		// --- Basic Straights ---
		{"Small Straight 1-4", []int{1, 2, 3, 4}, 6, []int{1, 2, 3, 4}},
		{"Small Straight 3-6", []int{3, 4, 5, 6}, 6, []int{3, 4, 5, 6}},
		{"Large Straight 1-5", []int{1, 2, 3, 4, 5}, 6, []int{1, 2, 3, 4, 5}},
		{"Large Straight 2-6", []int{2, 3, 4, 5, 6}, 6, []int{2, 3, 4, 5, 6}},
		{"Larger Straight 1-6", []int{1, 2, 3, 4, 5, 6}, 6, []int{1, 2, 3, 4, 5, 6}},
		{"Largest Straight 1-7", []int{1, 2, 3, 4, 5, 6, 7}, 7, []int{1, 2, 3, 4, 5, 6, 7}},

		// --- Straights with Duplicates ---
		// Only one die for each value in the sequence should be returned
		{"Small Straight 1-4 with duplicate 2", []int{1, 2, 2, 3, 4}, 6, []int{1, 2, 3, 4}},
		{"Large Straight 2-6 with duplicate 4", []int{2, 3, 4, 4, 4, 5, 6}, 6, []int{2, 3, 4, 5, 6}},
		{"Larger Straight 1-6 with duplicate 1", []int{1, 1, 2, 3, 4, 5, 6}, 6, []int{1, 2, 3, 4, 5, 6}},

		// --- Straights with Extra Dice (Outside the Best Straight) ---
		{"Small Straight 1-4 with extra 6", []int{1, 2, 3, 4, 6}, 6, []int{1, 2, 3, 4}},
		{"Larger Straight 1-6 with extra 1", []int{1, 2, 3, 4, 5, 6, 1}, 6, []int{1, 2, 3, 4, 5, 6}},

		// --- Unsorted Input ---
		{"Unsorted Small Straight 1-4 with extras", []int{4, 1, 3, 2, 4}, 6, []int{1, 2, 3, 4}},
		{"Unsorted Larger Straight 1-6 with extras", []int{5, 1, 3, 6, 2, 4, 3}, 6, []int{1, 2, 3, 4, 5, 6}},

		// --- Edge Case: Minimum Length ---
		{"Exactly Small Straight 1-4", []int{1, 2, 3, 4}, 6, []int{1, 2, 3, 4}},

		// technically not possible within constraints of game
		// {"2 unsorted straights", []int{1, 3, 4, 5, 2, 8, 7, 9, 10}, 10, []int{1, 2, 3, 4, 5}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare input dice
			dice := generateDiceValues(tc.inputDiceValues, tc.maxPips)

			// --- Optional: Add prints inside the test if needed ---
			// fmt.Printf("--- Running Test: %s ---\n", tc.name)
			// fmt.Println("Input Dice Values:", tc.inputDiceValues)

			// Call the specific function under test
			foundDice := findBestSingleConsecutive(dice)

			// fmt.Println("Expected Values:", tc.expectedDiceValues)
			// fmt.Println("Found Dice Values:", extractDiceValues(foundDice))
			// fmt.Println("-----------------------------")

			// Compare the values of the returned dice (order-independent)
			// Use a slightly modified comparison for this specific test
			compareStraightResult(t, foundDice, tc.expectedDiceValues, tc.name, tc.inputDiceValues)
		})
	}
}

// compareStraightResult is a helper specific to TestFindBestSingleConsecutive
// as the expected length isn't tied to a HandRank here.
func compareStraightResult(t *testing.T, gotDice []Die, wantValues []int, testName string, inputValues []int) {
	t.Helper() // Marks this as a helper function

	gotValues := extractDiceValues(gotDice)

	// Sort both slices for comparison
	sort.Ints(gotValues)
	sort.Ints(wantValues)

	// Handle the case where no straight is expected (empty slice)
	if len(wantValues) == 0 {
		if len(gotValues) != 0 {
			t.Errorf("%s: findBestSingleConsecutive(%v) returned dice with values %v; want empty slice []",
				testName, inputValues, gotValues)
		}
		// If both are empty, it's a pass for this case, so return.
		return
	}

	// If a non-empty slice is expected, compare values
	if !reflect.DeepEqual(gotValues, wantValues) {
		t.Errorf("%s: findBestSingleConsecutive(%v) returned dice with values %v; want values %v",
			testName, inputValues, gotValues, wantValues)
	}
}
