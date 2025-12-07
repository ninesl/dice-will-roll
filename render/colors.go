package render

type ColorType int

const (
	Red ColorType = iota
	Orange
	Yellow
	Green
	Blue
	Indigo
	Purple
	MAX_RAINBOW
)

var (
	RainbowColors = [MAX_RAINBOW]Vec3{
		KageColor(150, 0, 0),    // red
		KageColor(175, 127, 25), // orange
		KageColor(160, 160, 0),  // yellow
		KageColor(0, 150, 50),   // green
		KageColor(50, 50, 200),  // blue
		KageColor(75, 0, 130),   // indigo
		KageColor(125, 50, 183), // purple
	}

	//TODO: make color helpers/shades
	Grey  = KageColor(128, 128, 128)
	Brown = KageColor(139, 69, 19)

	// White shades for rock rendering (brightest to darkest)
	WhiteBright = KageColor(255, 255, 255) // Pure white for highlights
	WhiteLight  = KageColor(230, 230, 230) // Light white for outer surfaces
	WhiteMid    = KageColor(200, 200, 200) // Medium white for inner areas
	WhiteDark   = KageColor(170, 170, 170) // Darker white for crater/depth
)
