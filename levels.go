package main

// A level keep track of state and scoring for an 'instance' of a run
// FIXME: better names
type Level struct {
	// StartRocks int // how many rocks to start
	Rocks int // how many rocks are left
}

func NewLevel(startRocks int) *Level {
	return &Level{
		// StartRocks: startRocks,
		Rocks: startRocks,
	}
}
