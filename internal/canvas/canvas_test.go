package canvas

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		width, height  int
		wantW, wantH   int
		wantCW, wantCH int
	}{
		{"exact 2x4", 2, 4, 2, 4, 1, 1},
		{"exact 4x8", 4, 8, 4, 8, 2, 2},
		{"round up width", 3, 4, 4, 4, 2, 1},
		{"round up height", 2, 5, 2, 8, 1, 2},
		{"round up both", 3, 5, 4, 8, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.width, tt.height)
			if c.Width() != tt.wantW {
				t.Errorf("Width() = %d, want %d", c.Width(), tt.wantW)
			}
			if c.Height() != tt.wantH {
				t.Errorf("Height() = %d, want %d", c.Height(), tt.wantH)
			}
			if c.CharWidth() != tt.wantCW {
				t.Errorf("CharWidth() = %d, want %d", c.CharWidth(), tt.wantCW)
			}
			if c.CharHeight() != tt.wantCH {
				t.Errorf("CharHeight() = %d, want %d", c.CharHeight(), tt.wantCH)
			}
		})
	}
}

func TestSetGetClear(t *testing.T) {
	c := New(4, 8)

	// Initially all pixels should be off
	if c.Get(0, 0) {
		t.Error("pixel should be off initially")
	}

	// Set a pixel
	c.Set(1, 2)
	if !c.Get(1, 2) {
		t.Error("pixel should be on after Set")
	}

	// Clear the pixel
	c.Clear(1, 2)
	if c.Get(1, 2) {
		t.Error("pixel should be off after Clear")
	}

	// Out of bounds should not panic
	c.Set(-1, 0)
	c.Set(100, 0)
	c.Clear(-1, 0)
	c.Get(-1, 0)
}

func TestToggle(t *testing.T) {
	c := New(2, 4)

	if c.Get(0, 0) {
		t.Error("pixel should be off initially")
	}

	c.Toggle(0, 0)
	if !c.Get(0, 0) {
		t.Error("pixel should be on after first Toggle")
	}

	c.Toggle(0, 0)
	if c.Get(0, 0) {
		t.Error("pixel should be off after second Toggle")
	}
}

func TestFillReset(t *testing.T) {
	c := New(4, 8)

	c.Fill()
	for y := 0; y < c.Height(); y++ {
		for x := 0; x < c.Width(); x++ {
			if !c.Get(x, y) {
				t.Errorf("pixel (%d,%d) should be on after Fill", x, y)
			}
		}
	}

	c.Reset()
	for y := 0; y < c.Height(); y++ {
		for x := 0; x < c.Width(); x++ {
			if c.Get(x, y) {
				t.Errorf("pixel (%d,%d) should be off after Reset", x, y)
			}
		}
	}
}

func TestSinglePixelBraille(t *testing.T) {
	// Test each pixel position in a single braille character
	tests := []struct {
		x, y int
		want rune
	}{
		{0, 0, '\u2801'}, // dot 1
		{0, 1, '\u2802'}, // dot 2
		{0, 2, '\u2804'}, // dot 3
		{0, 3, '\u2840'}, // dot 7
		{1, 0, '\u2808'}, // dot 4
		{1, 1, '\u2810'}, // dot 5
		{1, 2, '\u2820'}, // dot 6
		{1, 3, '\u2880'}, // dot 8
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			c := New(2, 4)
			c.Set(tt.x, tt.y)
			got := []rune(c.String())[0]
			if got != tt.want {
				t.Errorf("Set(%d,%d): got %U, want %U", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestFullBrailleChar(t *testing.T) {
	c := New(2, 4)
	c.Fill()

	got := c.String()
	want := "⣿" // all 8 dots on = U+28FF

	if got != want {
		t.Errorf("Full braille char: got %q (%U), want %q (%U)",
			got, []rune(got)[0], want, []rune(want)[0])
	}
}

func TestEmptyBrailleChar(t *testing.T) {
	c := New(2, 4)

	got := c.String()
	want := "⠀" // empty braille = U+2800

	if got != want {
		t.Errorf("Empty braille char: got %q (%U), want %q (%U)",
			got, []rune(got)[0], want, []rune(want)[0])
	}
}

func TestMultiCharCanvas(t *testing.T) {
	c := New(4, 8) // 2x2 braille characters

	// Set one pixel in each quadrant
	c.Set(0, 0) // top-left char, dot 1
	c.Set(3, 3) // top-right char, dot 8
	c.Set(0, 4) // bottom-left char, dot 1
	c.Set(3, 7) // bottom-right char, dot 8

	got := c.String()
	want := "⠁⢀\n⠁⢀"

	if got != want {
		t.Errorf("Multi-char canvas:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRow(t *testing.T) {
	c := New(4, 8)
	c.Set(0, 0) // top-left
	c.Set(3, 7) // bottom-right

	row0 := c.Row(0)
	if row0 != "⠁⠀" {
		t.Errorf("Row(0) = %q, want %q", row0, "⠁⠀")
	}

	row1 := c.Row(1)
	if row1 != "⠀⢀" {
		t.Errorf("Row(1) = %q, want %q", row1, "⠀⢀")
	}

	// Out of bounds
	if c.Row(-1) != "" {
		t.Error("Row(-1) should return empty string")
	}
	if c.Row(100) != "" {
		t.Error("Row(100) should return empty string")
	}
}
