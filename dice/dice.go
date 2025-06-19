package dice

import (
	"fmt"
	"log"
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
//
// is used to populate [6]mat3 for shader uniforms
// func (d *Die) LocationsPips() [6][9]float32 {
// 	pipLocations := [6][9]float32{}
// 	for i := range len(pipLocations) {
// 		locs := d.faces[i].pipLocations()
// 		pipLocations[i] = locs
// 	}

// 	return pipLocations
// }

// ONLY WORKS FOR 6 SIDES
//
// is used to populate [6]mat3 for shader uniforms
// This now returns a flat []float32, with each 9-float segment representing a mat3 in column-major order.
func (d *Die) LocationsPips() []float32 {
	// Each mat3 has 9 floats. 6 faces * 9 floats/face = 54 floats total.
	flattenedPipLayouts := make([]float32, 6*9)
	flatIdx := 0

	for faceIndex := 0; faceIndex < 6; faceIndex++ {
		var currentFacePipsRowMajor [9]float32
		// Ensure we don't go out of bounds if a die somehow has fewer than 6 faces defined,
		// though NewDie(6) should prevent this.
		if faceIndex < len(d.faces) {
			currentFacePipsRowMajor = d.faces[faceIndex].pipLocations() // This is row-major based on iota constants
		} else {
			// Fallback for safety: an empty face (all zeros)
			currentFacePipsRowMajor = [9]float32{}
		}

		// Transpose from row-major (from pipLocations) to column-major for the shader.
		// Visual pip grid: V[row][col]
		// Your iota constants produce row-major order for currentFacePipsRowMajor:
		// Index: 0   1   2    3   4   5    6   7   8
		// Visual:V00 V01 V02  V10 V11 V12  V20 V21 V22
		// (e.g., currentFacePipsRowMajor[topLeft] which is currentFacePipsRowMajor[0] stores V00)
		// (e.g., currentFacePipsRowMajor[topMiddle] which is currentFacePipsRowMajor[1] stores V01)

		// We need to store it in column-major order in the flattened slice for Kage:
		// For one mat3: [V00,V10,V20, V01,V11,V21, V02,V12,V22]

		// Column 0
		flattenedPipLayouts[flatIdx+0] = currentFacePipsRowMajor[topLeft]    // V00
		flattenedPipLayouts[flatIdx+1] = currentFacePipsRowMajor[middleLeft] // V10
		flattenedPipLayouts[flatIdx+2] = currentFacePipsRowMajor[bottomLeft] // V20
		// Column 1
		flattenedPipLayouts[flatIdx+3] = currentFacePipsRowMajor[topMiddle]    // V01
		flattenedPipLayouts[flatIdx+4] = currentFacePipsRowMajor[middle]       // V11
		flattenedPipLayouts[flatIdx+5] = currentFacePipsRowMajor[bottomMiddle] // V21
		// Column 2
		flattenedPipLayouts[flatIdx+6] = currentFacePipsRowMajor[topRight]    // V02
		flattenedPipLayouts[flatIdx+7] = currentFacePipsRowMajor[middleRight] // V12
		flattenedPipLayouts[flatIdx+8] = currentFacePipsRowMajor[bottomRight] // V22

		flatIdx += 9
	}
	return flattenedPipLayouts
}

// gives values from 1-9 for each face
func New6SidedDie(values [6]int) Die {
	faces := []Face{}
	for _, val := range values {
		faces = append(faces, NewFace(val))
	}
	return Die{
		faces: faces,
	}
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
	if pips < 1 || pips > MAX_PIPS {
		log.Fatal("could not make a dieface with %d pips. Must be between 1 - %d", pips, MAX_PIPS)
	}

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

// Returns the index of the face that is active.
//
// Used for which side is visible for uniforms
func (d *Die) ActiveFaceIndex() int {
	return d.activeFace
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

// Returns the sum of all faces' .NumPips()
func (d *Die) NumPips() int {
	sum := 0
	for _, face := range d.faces {
		sum += face.NumPips()
	}
	return sum
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
func (f *Face) pipLocations() [9]float32 {
	pipLoc := [9]float32{}

	switch f.NumPips() {
	case 1:
		pipLoc[middle] = 1.0
	case 2:
		pipLoc[topRight] = 1.0
		pipLoc[bottomLeft] = 1.0
	case 3:
		pipLoc[topLeft] = 1.0
		pipLoc[middle] = 1.0
		pipLoc[bottomRight] = 1.0
	case 4:
		pipLoc[topLeft] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[bottomRight] = 1.0
		pipLoc[bottomLeft] = 1.0
	case 5:
		pipLoc[topLeft] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[middle] = 1.0
		pipLoc[bottomRight] = 1.0
		pipLoc[bottomLeft] = 1.0
	case 6:
		pipLoc[topLeft] = 1.0
		pipLoc[middleLeft] = 1.0
		pipLoc[bottomLeft] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[middleRight] = 1.0
		pipLoc[bottomRight] = 1.0
	case 7:
		pipLoc[middle] = 1.0
		pipLoc[topLeft] = 1.0
		pipLoc[middleLeft] = 1.0
		pipLoc[bottomLeft] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[middleRight] = 1.0
		pipLoc[bottomRight] = 1.0
	case 8:
		pipLoc[topLeft] = 1.0
		pipLoc[middleLeft] = 1.0
		pipLoc[bottomLeft] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[middleRight] = 1.0
		pipLoc[bottomRight] = 1.0
		pipLoc[topMiddle] = 1.0
		pipLoc[bottomMiddle] = 1.0
	case 9:
		pipLoc[topLeft] = 1.0
		pipLoc[topMiddle] = 1.0
		pipLoc[topRight] = 1.0
		pipLoc[middleLeft] = 1.0
		pipLoc[middle] = 1.0
		pipLoc[middleRight] = 1.0
		pipLoc[bottomRight] = 1.0
		pipLoc[bottomMiddle] = 1.0
		pipLoc[bottomLeft] = 1.0
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
