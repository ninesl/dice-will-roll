package main

import "fmt"

// idk if this is best practice. This is to help me make unit tests, etc.
// EVERYTHING PANICS !
//
// crash gets caught in main game loop and then an error screen appears. likely will have ways to push bug reports, etc.

func MustLen(length int, expectedLength int, sourceMsg ...string) {
	if length != expectedLength {
		var msg string
		if len(sourceMsg) > 0 {
			msg = fmt.Sprintln(msg, sourceMsg)
		}
		msg = fmt.Sprintf("%s\nPANIC %d found. expected %d", msg, length, expectedLength)
		panic(msg)
	}
}
