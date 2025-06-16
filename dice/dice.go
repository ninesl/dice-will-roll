package dice

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

// Most pips a face can have. Not the highest value
const MAX_PIPS = 9

// TODO:FIXME: will be obnoxious to add sides. maybe an interface for the die that works with all variations of faces?
const (
	FrontFace int = iota
	LeftFace
	BottomFace
	TopFace
	RightFace
	BehindFace

	// [0] front 1 pip
	// [1] left 2 pip
	// [2] bottom 3 pip
	// [3] top 4 pip
	// [4] right 5 pip
	// [5] behind 6 pip
)

// ONLY WORKS FOR 6 SIDES
func (d *Die) LocationsPips() [6][9]int {
	pipLocations := [6][9]int{}
	for i := range len(pipLocations) {
		locs := d.faces[i].pipLocations()
		pipLocations[i] = locs
	}

	return pipLocations
}

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

// 0 1 2
// 3 4 5
// 6 7 8
// pip locations on face
const (
	topLeft int = iota
	topMiddle
	topRight
	middleLeft
	middle
	middleRight
	bottomLeft
	bottomMiddle
	bottomRight
)

// used for shader uniforms
//
// TODO: bool might not work long term, we may need another check for gems/mods in pips
//
// returns a 9 length true/false array
//
// each index corresponds to where on the die face the pip is
//
//	[false, false, false,
//	 false, true, false,
//	 false, false, false] // die face with 1 pip
func (f *Face) pipLocations() [9]int {
	pipLoc := [9]int{}

	switch f.NumPips() {
	case 1:
		pipLoc[middle] = 1
	case 2:
		pipLoc[topRight] = 1
		pipLoc[bottomLeft] = 1
	case 3:
		pipLoc[topLeft] = 1
		pipLoc[middle] = 1
		pipLoc[bottomRight] = 1
	case 4:
		pipLoc[topLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[bottomRight] = 1
		pipLoc[bottomLeft] = 1
	case 5:
		pipLoc[topLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[middle] = 1
		pipLoc[bottomRight] = 1
		pipLoc[bottomLeft] = 1
	case 6:
		pipLoc[topLeft] = 1
		pipLoc[middleLeft] = 1
		pipLoc[bottomLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[middleRight] = 1
		pipLoc[bottomRight] = 1
	case 7:
		pipLoc[middle] = 1
		pipLoc[topLeft] = 1
		pipLoc[middleLeft] = 1
		pipLoc[bottomLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[middleRight] = 1
		pipLoc[bottomRight] = 1
	case 8:
		pipLoc[topLeft] = 1
		pipLoc[middleLeft] = 1
		pipLoc[bottomLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[middleRight] = 1
		pipLoc[bottomRight] = 1
		pipLoc[topMiddle] = 1
		pipLoc[bottomMiddle] = 1
	case 9:
		pipLoc[topLeft] = 1
		pipLoc[topRight] = 1
		pipLoc[middle] = 1
		pipLoc[bottomRight] = 1
		pipLoc[bottomLeft] = 1
	}

	return pipLoc
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
