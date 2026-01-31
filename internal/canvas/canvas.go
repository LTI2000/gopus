// Package canvas provides a braille-based drawing canvas for terminal graphics.
package canvas

import "strings"

// brailleBase is the Unicode code point for an empty braille character.
const brailleBase = '\u2800'

// pixelToBit maps (x, y) within a 2x4 braille cell to the bit value.
// x: 0-1 (column), y: 0-3 (row)
//
// Braille dot layout:
//
//	┌───┬───┐
//	│ 1 │ 4 │  Row 0
//	├───┼───┤
//	│ 2 │ 5 │  Row 1
//	├───┼───┤
//	│ 3 │ 6 │  Row 2
//	├───┼───┤
//	│ 7 │ 8 │  Row 3
//	└───┴───┘
//	Col 0  Col 1
var pixelToBit = [2][4]rune{
	{0x01, 0x02, 0x04, 0x40}, // column 0: dots 1, 2, 3, 7
	{0x08, 0x10, 0x20, 0x80}, // column 1: dots 4, 5, 6, 8
}

// Canvas represents a drawable braille canvas.
// Coordinates are in pixels, where each braille character represents a 2x4 pixel area.
type Canvas struct {
	width  int      // width in pixels
	height int      // height in pixels
	pixels [][]bool // pixel data [y][x]
}

// New creates a new canvas with the given pixel dimensions.
// Width and height are automatically rounded up to the nearest braille cell boundary
// (width to multiple of 2, height to multiple of 4).
func New(width, height int) *Canvas {
	// Round up to braille cell boundaries
	if width%2 != 0 {
		width++
	}
	if height%4 != 0 {
		height += 4 - (height % 4)
	}

	pixels := make([][]bool, height)
	for y := range pixels {
		pixels[y] = make([]bool, width)
	}

	return &Canvas{
		width:  width,
		height: height,
		pixels: pixels,
	}
}

// Width returns the canvas width in pixels.
func (c *Canvas) Width() int {
	return c.width
}

// Height returns the canvas height in pixels.
func (c *Canvas) Height() int {
	return c.height
}

// CharWidth returns the canvas width in braille characters.
func (c *Canvas) CharWidth() int {
	return c.width / 2
}

// CharHeight returns the canvas height in braille characters (rows).
func (c *Canvas) CharHeight() int {
	return c.height / 4
}

// inBounds checks if the given pixel coordinates are within the canvas.
func (c *Canvas) inBounds(x, y int) bool {
	return x >= 0 && x < c.width && y >= 0 && y < c.height
}

// Set turns on the pixel at (x, y).
func (c *Canvas) Set(x, y int) {
	if c.inBounds(x, y) {
		c.pixels[y][x] = true
	}
}

// Clear turns off the pixel at (x, y).
func (c *Canvas) Clear(x, y int) {
	if c.inBounds(x, y) {
		c.pixels[y][x] = false
	}
}

// Toggle flips the pixel state at (x, y).
func (c *Canvas) Toggle(x, y int) {
	if c.inBounds(x, y) {
		c.pixels[y][x] = !c.pixels[y][x]
	}
}

// Get returns the pixel state at (x, y).
// Returns false for out-of-bounds coordinates.
func (c *Canvas) Get(x, y int) bool {
	if !c.inBounds(x, y) {
		return false
	}
	return c.pixels[y][x]
}

// Fill turns on all pixels.
func (c *Canvas) Fill() {
	for y := range c.pixels {
		for x := range c.pixels[y] {
			c.pixels[y][x] = true
		}
	}
}

// Reset turns off all pixels.
func (c *Canvas) Reset() {
	for y := range c.pixels {
		for x := range c.pixels[y] {
			c.pixels[y][x] = false
		}
	}
}

// charAt returns the braille character for the cell at character position (cx, cy).
func (c *Canvas) charAt(cx, cy int) rune {
	// Convert character position to pixel position
	px := cx * 2
	py := cy * 4

	var char rune = brailleBase

	// Check each pixel in the 2x4 braille cell
	for dx := 0; dx < 2; dx++ {
		for dy := 0; dy < 4; dy++ {
			if c.Get(px+dx, py+dy) {
				char += pixelToBit[dx][dy]
			}
		}
	}

	return char
}

// Row renders a single row of braille characters at the given character row index.
func (c *Canvas) Row(cy int) string {
	if cy < 0 || cy >= c.CharHeight() {
		return ""
	}

	var sb strings.Builder
	sb.Grow(c.CharWidth())

	for cx := 0; cx < c.CharWidth(); cx++ {
		sb.WriteRune(c.charAt(cx, cy))
	}

	return sb.String()
}

// String renders the entire canvas as a multi-line braille string.
func (c *Canvas) String() string {
	var sb strings.Builder
	charHeight := c.CharHeight()

	for cy := 0; cy < charHeight; cy++ {
		if cy > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(c.Row(cy))
	}

	return sb.String()
}
