package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// A Modifier is the buff (or debuff) that gets applied to
// all varieties of items throughout DWR.
//
// Usage: use the 'enums' available for all valid Modifiers
//
// The default Modifier (0) is
type Modifier uint8

const (
	ModNONE Modifier = iota // ModNONE is the default modifier. It has no additonal effects by default
)

type Die struct {
	activeFace int // activeFace is the face that is 'showing' on the Die
	faces      []Face
}

// A Face is from Die's faces
//
// each side has a number of pips, which correlates to the Value of the face
//
// Value, when refering to Faces, is the literal number of Pips (dots) on a face
type Face struct {
	// mod  Modifier
	pips []Modifier
}

// The highest value a face can have. (not the highest it count for??)
const MAX_PIPS = 9

// Makes a blank die with each face being one more than the last, starting from 1
func NewDie(sides int) Die {
	faces := []Face{}
	for i := range sides {
		faces = append(faces, NewFace(i+1)) // so we dont have 0-5 pips
	}
	return Die{
		faces: faces,
	}
}

func NewFace(pips int) Face {
	facePips := []Modifier{}
	for range pips {
		facePips = append(facePips, ModNONE)
	}

	return Face{
		pips: facePips,
	}
}

// Returns X blank (ModNONE) dice with 6 faces, each.
//
// Generally used to populate player's starting dice
func BlankDice(numDice int) []Die {
	dice := []Die{}
	for range numDice {
		dice = append(dice, NewDie(6))
	}
	return dice
}

// Returns x blank (ModNONE) dice with maxValue faces, each.
func BlankDiceRange(numDice int, maxValue int) []Die {
	dice := []Die{}
	for range numDice {
		dice = append(dice, NewDie(maxValue))
	}
	return dice
}

// output of each side and how many pips on each side
func (d *Die) String() string {
	var sb strings.Builder
	for i := range d.faces {
		sb.WriteString(fmt.Sprintf("side %d - %d pips\n",
			i, len(d.faces[i].pips)))
	}
	return sb.String()
}

// Returns a pointer to the currently acctive face.
//
// Used to access the given Face for scoring, modifiers, etc.
func (d *Die) ActiveFace() *Face {
	return &d.faces[d.activeFace]
}

// Set the active face to a random 0-len(faces)
//
//	d.ActiveFace() # is called to return the pointer to Face
//
// SHOULD NOT BE USED TO MODIFY THE FACE IT RETURNS! (except in specific cases)
func (d *Die) Roll() *Face {
	d.activeFace = rand.Intn(len(d.faces))
	return d.ActiveFace()
}

// Value is the literal NUMBER of pips on the face and relevant modifiers (to the die, not enviornment) are applied
func (f *Face) Value() int {
	value := f.NumPips()

	// modifier?

	return value
}

// NumPips returns len(f.pips)
func (f *Face) NumPips() int {
	return len(f.pips)
}

// Score is the number that gets added to the total when the player plays a hand
func (f *Face) Score() int {
	/*
		var (
			score = 0
		)
		// for i := range f.pips {
		for range f.pips {
			score += 1
			// score += f.pips[i] * f.mod

			// TODO: modifier math
		}
	*/
	return f.Value()
}

func DiceString(dice []Die) string {
	s := "["
	for i := range dice {
		s = fmt.Sprintf("%s%d,", s, dice[i].ActiveFace().NumPips())
	}
	return s + "]"
}
