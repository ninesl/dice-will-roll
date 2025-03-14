// DICE WILL ROLL
/*
logic
	types
		die

render
*/
package main

import (
	"fmt"
	"unsafe"
)

// Player keeps track of the player's data during a run
type Player struct {
	Dice []Die
}

func main() {
	p := Player{
		Dice: BlankDice(5),
	}

	for i := range p.Dice {
		fmt.Println(unsafe.Sizeof(p.Dice[i]))
		fmt.Println(p.Dice[i].String())
	}

	fmt.Println(unsafe.Sizeof(p))
}
