package canvas

import (
	"testing"
	"testing/quick"
)

// TestNewDimensionRounding verifies that New() always rounds dimensions
// to valid braille cell boundaries.
func TestNewDimensionRounding(t *testing.T) {
	property := func(width, height uint8) bool {
		// Use uint8 to limit input range (0-255)
		w, h := int(width), int(height)
		if w == 0 || h == 0 {
			return true // Skip zero dimensions
		}

		c := New(w, h)

		// Property 1: Width is always even
		if c.Width()%2 != 0 {
			return false
		}

		// Property 2: Height is always multiple of 4
		if c.Height()%4 != 0 {
			return false
		}

		// Property 3: Dimensions are >= input
		if c.Width() < w || c.Height() < h {
			return false
		}

		// Property 4: Dimensions don't grow more than necessary
		if c.Width() > w+1 || c.Height() > h+3 {
			return false
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestSetGetRoundTrip verifies that Set followed by Get returns true
// for any valid coordinate.
func TestSetGetRoundTrip(t *testing.T) {
	property := func(width, height, x, y uint8) bool {
		w, h := int(width)+2, int(height)+4 // Ensure minimum size
		px, py := int(x), int(y)

		c := New(w, h)

		// Only test in-bounds coordinates
		if px >= c.Width() || py >= c.Height() {
			return true // Skip out-of-bounds
		}

		// Property: Set then Get returns true
		c.Set(px, py)
		return c.Get(px, py)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestToggleInvolution verifies the involution property:
// For any function f, f(f(x)) = x
// Toggle is an involution because toggling twice restores the original state.
func TestToggleInvolution(t *testing.T) {
	property := func(width, height, x, y uint8, initialState bool) bool {
		w, h := int(width)+2, int(height)+4
		px, py := int(x), int(y)

		c := New(w, h)

		if px >= c.Width() || py >= c.Height() {
			return true
		}

		// Set initial state
		if initialState {
			c.Set(px, py)
		}
		original := c.Get(px, py)

		// Toggle twice
		c.Toggle(px, py)
		c.Toggle(px, py)

		// Property: State is restored
		return c.Get(px, py) == original
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestBrailleCharacterRange verifies that rendered characters are valid braille.
func TestBrailleCharacterRange(t *testing.T) {
	const brailleMin = '\u2800'
	const brailleMax = '\u28FF'

	property := func(pixels [8]bool) bool {
		c := New(2, 4) // Single braille cell

		// Set pixels based on input
		positions := [][2]int{
			{0, 0}, {0, 1}, {0, 2}, {0, 3},
			{1, 0}, {1, 1}, {1, 2}, {1, 3},
		}
		for i, on := range pixels {
			if on {
				c.Set(positions[i][0], positions[i][1])
			}
		}

		// Get the rendered character
		str := c.String()
		if len(str) == 0 {
			return false
		}

		char := []rune(str)[0]

		// Property: Character is in valid braille range
		return char >= brailleMin && char <= brailleMax
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestFillIdempotence verifies that calling Fill multiple times
// has the same effect as calling it once.
func TestFillIdempotence(t *testing.T) {
	property := func(width, height uint8) bool {
		w, h := int(width)+2, int(height)+4

		c1 := New(w, h)
		c2 := New(w, h)

		// Apply Fill once to c1
		c1.Fill()

		// Apply Fill twice to c2
		c2.Fill()
		c2.Fill()

		// Property: Both canvases should be identical
		for y := 0; y < c1.Height(); y++ {
			for x := 0; x < c1.Width(); x++ {
				if c1.Get(x, y) != c2.Get(x, y) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestResetIdempotence verifies that calling Reset multiple times
// has the same effect as calling it once.
func TestResetIdempotence(t *testing.T) {
	property := func(width, height uint8) bool {
		w, h := int(width)+2, int(height)+4

		c1 := New(w, h)
		c2 := New(w, h)

		// Set some pixels first
		c1.Fill()
		c2.Fill()

		// Apply Reset once to c1
		c1.Reset()

		// Apply Reset twice to c2
		c2.Reset()
		c2.Reset()

		// Property: Both canvases should be identical (all false)
		for y := 0; y < c1.Height(); y++ {
			for x := 0; x < c1.Width(); x++ {
				if c1.Get(x, y) != c2.Get(x, y) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestClearAfterSet verifies that Clear turns off a pixel that was set.
func TestClearAfterSet(t *testing.T) {
	property := func(width, height, x, y uint8) bool {
		w, h := int(width)+2, int(height)+4
		px, py := int(x), int(y)

		c := New(w, h)

		if px >= c.Width() || py >= c.Height() {
			return true
		}

		// Set then Clear
		c.Set(px, py)
		c.Clear(px, py)

		// Property: Pixel should be off
		return !c.Get(px, py)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestOutOfBoundsGetReturnsFalse verifies that Get returns false for
// out-of-bounds coordinates.
func TestOutOfBoundsGetReturnsFalse(t *testing.T) {
	property := func(width, height uint8, x, y int16) bool {
		w, h := int(width)+2, int(height)+4
		px, py := int(x), int(y)

		c := New(w, h)

		// Only test out-of-bounds coordinates
		if px >= 0 && px < c.Width() && py >= 0 && py < c.Height() {
			return true // Skip in-bounds
		}

		// Property: Out-of-bounds Get returns false
		return !c.Get(px, py)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestCharDimensionsConsistency verifies that CharWidth and CharHeight
// are consistent with Width and Height.
func TestCharDimensionsConsistency(t *testing.T) {
	property := func(width, height uint8) bool {
		w, h := int(width)+2, int(height)+4

		c := New(w, h)

		// Property: CharWidth * 2 == Width
		if c.CharWidth()*2 != c.Width() {
			return false
		}

		// Property: CharHeight * 4 == Height
		if c.CharHeight()*4 != c.Height() {
			return false
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
