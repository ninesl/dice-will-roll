package render

var (
	GAME_BOUNDS_X float64
	GAME_BOUNDS_Y float64
)

// A zone is where the cursor is
//
// zones specific implementation is specific to what it is, see parts.txt
//
//	// Zones are:
//	ROLL
//	GEMS
//	SCORE
//
// TODO: zones make up Game Screens, ie LOOP MINE BASE etc.
type Zone struct {
	MinWidth  float64
	MaxWidth  float64
	MinHeight float64
	MaxHeight float64
}

var (
	// bounds set during LoadGame()
	ROLLZONE  Zone
	SCOREZONE Zone
)

func SetZones() {
	minWidth := GAME_BOUNDS_X / 5
	minHeight := GAME_BOUNDS_Y / 5

	ROLLZONE = Zone{
		MinWidth:  minWidth,
		MaxWidth:  GAME_BOUNDS_X - minWidth,
		MinHeight: minHeight,
		MaxHeight: GAME_BOUNDS_Y - minHeight,
	}
	SCOREZONE = Zone{
		MinWidth:  minWidth,
		MaxWidth:  ROLLZONE.MaxWidth,
		MinHeight: 0,
		MaxHeight: ROLLZONE.MinHeight - 1,
	}
}
