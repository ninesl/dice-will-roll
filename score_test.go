package main

import "testing"

// TestScoreComprehensive covers scoring for all defined HandRank types.
// It assumes Score() correctly uses DetermineHandRank and FindHandRankDice
// to identify the contributing dice and sum their values.
// NOTE: Tests for straights assume findBestSingleConsecutive in FindHandRankDice is implemented correctly.
func TestScoreComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		diceValues []int
		maxPips    int // Important for hands involving 7s
		expected   int // Sum of values of dice *making up the specific hand*
	}{
		// --- Base/Non-Pattern ---
		{"NO_HAND (Empty Input)", []int{}, 6, 0}, // Score for empty dice should be 0
		{"HIGH_DIE (Single)", []int{5}, 6, 5},
		{"HIGH_DIE (Multiple, No Pattern)", []int{1, 3, 5, 2, 6}, 6, 6}, // Highest die scores

		// --- Pairs ---
		{"ONE_PAIR", []int{4, 4, 1, 2, 5}, 6, 8},       // 4+4
		{"SNAKE_EYES", []int{1, 1, 6}, 6, 2},           // 1+1
		{"TWO_PAIR", []int{2, 2, 6, 6, 1}, 6, 16},      // 2+2+6+6
		{"THREE_PAIR", []int{1, 1, 3, 3, 5, 5}, 6, 18}, // 1+1+3+3+5+5

		// --- N of a Kind ---
		{"THREE_OF_A_KIND", []int{5, 5, 5, 1, 2}, 6, 15},       // 5+5+5
		{"FOUR_OF_A_KIND", []int{2, 2, 2, 2, 5}, 6, 8},         // 2+2+2+2
		{"FIVE_OF_A_KIND", []int{6, 6, 6, 6, 6, 1}, 6, 30},     // 6*5
		{"SIX_OF_A_KIND", []int{3, 3, 3, 3, 3, 3, 5}, 6, 18},   // 3*6
		{"SEVEN_OF_A_KIND", []int{4, 4, 4, 4, 4, 4, 4}, 6, 28}, // 4*7
		{"SEVEN_SEVENS", []int{7, 7, 7, 7, 7, 7, 7}, 7, 49},    // 7*7 (Needs maxPips=7)

		// --- Straights (Assuming findBestSingleConsecutive works) ---
		{"STRAIGHT_SMALL", []int{1, 2, 3, 4, 6}, 6, 10},                    // 1+2+3+4
		{"STRAIGHT_SMALL (Out of Order)", []int{4, 6, 1, 3, 2}, 6, 10},     // 1+2+3+4
		{"STRAIGHT_LARGE", []int{2, 3, 4, 5, 6}, 6, 20},                    // 2+3+4+5+6
		{"STRAIGHT_LARGE (With Extras)", []int{1, 2, 3, 4, 5, 1}, 6, 15},   // 1+2+3+4+5
		{"STRAIGHT_LARGER", []int{1, 2, 3, 4, 5, 6}, 6, 21},                // 1+2+3+4+5+6
		{"STRAIGHT_LARGER (With Pair)", []int{1, 2, 3, 4, 5, 6, 6}, 6, 21}, // 1+2+3+4+5+6
		{"STRAIGHT_LARGEST", []int{1, 2, 3, 4, 5, 6, 7}, 7, 28},            // 1+2+3+4+5+6+7 (Needs maxPips=7)
		// {"STRAIGHT_MAX", []int{}, 7, 0}, // Cannot test without definition/modifiers

		// --- House Variants ---
		{"FULL_HOUSE (3+2)", []int{3, 3, 3, 5, 5}, 6, 19},                // 3+3+3+5+5
		{"CROWDED_HOUSE (4+2)", []int{4, 4, 4, 4, 1, 1, 3}, 6, 18},       // 4+4+4+4+1+1
		{"TWO_THREE_OF_A_KIND (3+3)", []int{2, 2, 2, 5, 5, 5, 1}, 6, 21}, // 2+2+2+5+5+5
		{"OVERPOPULATED_HOUSE (4+3)", []int{5, 5, 5, 5, 2, 2, 2}, 6, 26}, // 5+5+5+5+2+2+2
		{"FULLEST_HOUSE (5+2)", []int{3, 3, 3, 3, 3, 2, 2}, 6, 19},       // 3+3+3+3+3+2+2

		// --- Interaction Cases (Score reflects the BEST hand determined) ---
		{"Interaction: Straight wins over Pair", []int{1, 2, 3, 4, 4}, 6, 10},               // Scores STRAIGHT_SMALL (1+2+3+4), not the pair
		{"Interaction: 4oak wins over Straight", []int{1, 2, 3, 4, 4, 4, 4}, 6, 16},         // Scores FOUR_OF_A_KIND (4+4+4+4), not straight
		{"Interaction: Full House wins over Straight", []int{1, 2, 3, 3, 3, 2, 2}, 6, 15},   // Scores FULL_HOUSE (3+3+3+2+2), not straight 1-2-3
		{"Interaction: Full House wins over Two Pair", []int{5, 5, 5, 2, 2, 1}, 6, 19},      // Scores FULL_HOUSE (5+5+5+2+2), not two pair
		{"Interaction: Two Pair vs Snake Eyes", []int{1, 1, 2, 2}, 6, 6},                    // Scores TWO_PAIR (1+1+2+2=6) is higher than SNAKE_EYES (1+1=2)
		{"Interaction: Three Pair vs Three of a Kind", []int{2, 2, 4, 4, 6, 6, 6}, 6, 24},   // Scores THREE_PAIR (2+2+4+4+6+6=24), not THREE_OF_A_KIND (6+6+6=18) - Depends on relative rank order
		{"Interaction: Crowded House vs Four of a Kind", []int{5, 5, 5, 5, 2, 2, 1}, 6, 24}, // Scores CROWDED_HOUSE (5+5+5+5+2+2=24), not FOUR_OF_A_KIND (5+5+5+5=20)

	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// // Handle empty dice case separately as generateDiceValues might expect > 0
			// if len(tc.diceValues) == 0 {
			// 	if tc.expected != 0 {
			// 		t.Errorf("Test '%s': Score for empty dice expected 0, got check for %d", tc.name, tc.expected)
			// 	}
			// 	// Call score with nil or empty slice if that's how it handles it
			// 	score := Score(nil) // Or Score([]Die{})
			// 	if score != 0 {
			// 		t.Errorf("Test '%s': Score(nil) = %d; want 0", tc.name, score)
			// 	}
			// 	return // Skip rest of test for empty input
			// }

			dice := generateDiceValues(tc.diceValues, tc.maxPips)

			score := Score(dice)

			if score != tc.expected {
				t.Errorf("Test '%s': Score(%v) = %d; want %d",
					tc.name, tc.diceValues, score, tc.expected)
			}
		})
	}
}
