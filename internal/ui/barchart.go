package ui

import "strings"

// BrailleChart renders a braille line chart.
// Returns plain text (no ANSI) — exactly `height` lines, each `width` runes.
// values are data points (0..maxVal), drawn left to right.
// If len(values) > width*2, only the last width*2 values are shown.
func BrailleChart(values []float64, maxVal float64, width, height int) string {
	if maxVal <= 0 {
		maxVal = 1
	}
	if height <= 0 {
		height = 1
	}
	if width <= 0 {
		width = 1
	}

	xRes := width * 2
	yRes := height * 4

	// Trim values to fit
	if len(values) > xRes {
		values = values[len(values)-xRes:]
	}

	// Initialize grid with empty braille
	grid := make([][]rune, height)
	for r := 0; r < height; r++ {
		grid[r] = make([]rune, width)
		for c := 0; c < width; c++ {
			grid[r][c] = 0x2800
		}
	}

	// Convert value to y-dot position (0 = bottom, yRes-1 = top)
	toY := func(v float64) int {
		if v < 0 {
			v = 0
		}
		if v > maxVal {
			v = maxVal
		}
		y := int(v / maxVal * float64(yRes-1))
		if y >= yRes {
			y = yRes - 1
		}
		return y
	}

	// Set a dot at (x, y) where y=0 is bottom
	setDot := func(x, y int) {
		if x < 0 || x >= xRes || y < 0 || y >= yRes {
			return
		}
		charCol := x / 2
		invertedY := yRes - 1 - y
		charRow := invertedY / 4
		dotCol := x % 2
		dotRow := invertedY % 4
		if charRow < 0 || charRow >= height || charCol < 0 || charCol >= width {
			return
		}
		grid[charRow][charCol] |= brailleBit(dotCol, dotRow)
	}

	// Draw points and interpolate lines
	for i, v := range values {
		y := toY(v)
		setDot(i, y)

		// Interpolate between this point and previous using Bresenham's line
		if i > 0 {
			drawBresenhamLine(i-1, toY(values[i-1]), i, y, setDot)
		}
	}

	// Convert grid to string
	rows := make([]string, height)
	for r := 0; r < height; r++ {
		rows[r] = string(grid[r])
	}
	return strings.Join(rows, "\n")
}

func brailleBit(col, row int) rune {
	dotMap := [2][4]rune{
		{0x01, 0x02, 0x04, 0x40}, // left column: dots 1,2,3,7
		{0x08, 0x10, 0x20, 0x80}, // right column: dots 4,5,6,8
	}
	if col < 0 || col > 1 || row < 0 || row > 3 {
		return 0
	}
	return dotMap[col][row]
}

// drawBresenhamLine draws a line between two points using Bresenham's algorithm.
func drawBresenhamLine(x0, y0, x1, y1 int, setDot func(x, y int)) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	for {
		setDot(x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}
