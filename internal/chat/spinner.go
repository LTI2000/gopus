// Package chat provides the main chat loop functionality.
package chat

import (
	"fmt"
	"math"
	"os"
	"strings"

	"gopus/internal/canvas"
)

// ANSI escape codes for terminal control.
const (
	ansiEscape      = "\033["              // CSI (Control Sequence Introducer)
	ansiHideCursor  = ansiEscape + "?25l"  // hide cursor
	ansiShowCursor  = ansiEscape + "?25h"  // show cursor
	ansiResetColor  = ansiEscape + "0m"    // reset all attributes
	ansiClearLine   = ansiEscape + "K"     // clear from cursor to end of line
	ansiTrueColorFg = ansiEscape + "38;2;" // 24-bit foreground color prefix (append r;g;bm)
	ansi256ColorFg  = ansiEscape + "38;5;" // 256-color foreground prefix (append Nm)
	carriageReturn  = "\r"                 // return cursor to start of line
)

// Phase shifts for RGB color cycling, evenly distributed over 2π radians.
// This creates a smooth rainbow effect as the phase advances.
const (
	redPhase   = 0.0                 // 0 degrees
	greenPhase = 2.0 * math.Pi / 3.0 // 120 degrees
	bluePhase  = 4.0 * math.Pi / 3.0 // 240 degrees
)

// trailLength is the number of pixels in the animation trail (including the head).
const trailLength = 4

// circlePixels defines the circular path for the animation.
// Each point is an [x, y] coordinate on a 4x4 braille pixel grid.
// The path traces a clockwise circle starting from the top-center.
//
// Visual representation (0,0 is top-left):
//
//	    x: 0   1   2   3
//	y:0       *   *
//	  1   *           *
//	  2   *           *
//	  3       *   *
var circlePixels = [][2]int{
	{1, 0}, // top center-left
	{2, 0}, // top center-right
	{3, 1}, // right upper
	{3, 2}, // right lower
	{2, 3}, // bottom center-right
	{1, 3}, // bottom center-left
	{0, 2}, // left lower
	{0, 1}, // left upper
}

// CircleSpinner implements the animator.Animation interface with a circular
// braille pattern and smooth rainbow color cycling.
type CircleSpinner struct {
	canvas       *canvas.Canvas // braille character renderer
	phase        float64        // current phase angle for RGB cycling (radians)
	frameIdx     int            // current position in circlePixels
	useTrueColor bool           // true for 24-bit color, false for 256-color fallback
}

// NewCircleSpinner creates a new CircleSpinner.
// It auto-detects terminal color support via the COLORTERM environment variable.
func NewCircleSpinner() *CircleSpinner {
	return &CircleSpinner{
		canvas:       canvas.New(4, 4), // 2 braille chars wide, 1 char tall
		phase:        0,
		frameIdx:     0,
		useTrueColor: supportsTrueColor(),
	}
}

// supportsTrueColor checks if the terminal supports 24-bit true color
// by examining the COLORTERM environment variable.
// Returns true if COLORTERM contains "truecolor" or "24bit".
// Note: macOS Terminal.app does not support true color; iTerm2 and most modern terminals do.
func supportsTrueColor() bool {
	colorterm := os.Getenv("COLORTERM")
	return strings.Contains(colorterm, "truecolor") || strings.Contains(colorterm, "24bit")
}

// Start hides the cursor and renders the initial frame.
// Implements Animation.Start().
func (s *CircleSpinner) Start() {
	fmt.Print(ansiHideCursor)
	s.Render()
}

// Stop clears the animation line, resets colors, and restores the cursor.
// Implements Animation.Stop().
func (s *CircleSpinner) Stop() {
	fmt.Print(carriageReturn + ansiClearLine + ansiResetColor + ansiShowCursor)
}

// Render prints the current frame with color, then advances state for the next frame.
// Implements Animation.Render().
func (s *CircleSpinner) Render() {
	frame := s.renderFrame()
	colorCode := s.getColorCode()
	fmt.Printf("%s%s%s", carriageReturn, colorCode, frame)

	// Advance to next frame position and color
	s.frameIdx = (s.frameIdx + 1) % len(circlePixels)
	s.advanceColor()
}

// FrameCount returns the number of frames in one complete rotation (8 positions).
// Implements Animation.FrameCount().
func (s *CircleSpinner) FrameCount() int {
	return len(circlePixels)
}

// advanceColor increments the color phase for rainbow cycling.
// The phase completes a full cycle every ~3 seconds at 80ms frame intervals.
func (s *CircleSpinner) advanceColor() {
	s.phase += 2.0 * math.Pi / 37.5 // ~37.5 frames per color cycle
	if s.phase >= 2.0*math.Pi {
		s.phase -= 2.0 * math.Pi
	}
}

// getRGB calculates RGB values (0-255) using phase-shifted sine waves.
// Each color component is offset by 120° to create smooth rainbow transitions.
func (s *CircleSpinner) getRGB() (r, g, b int) {
	// sin(x) ∈ [-1,1] → (sin(x)+1)/2 ∈ [0,1] → scaled to [0,255]
	r = int((math.Sin(s.phase+redPhase) + 1.0) / 2.0 * 255.0)
	g = int((math.Sin(s.phase+greenPhase) + 1.0) / 2.0 * 255.0)
	b = int((math.Sin(s.phase+bluePhase) + 1.0) / 2.0 * 255.0)
	return r, g, b
}

// rgbTo256 converts RGB values (0-255) to an ANSI 256-color palette index.
// Uses the 6x6x6 color cube (indices 16-231) with rounded scaling.
func rgbTo256(r, g, b int) int {
	// Scale 0-255 to 0-5 with rounding: (v*5+127)/255
	r6 := (r*5 + 127) / 255
	g6 := (g*5 + 127) / 255
	b6 := (b*5 + 127) / 255
	return 16 + 36*r6 + 6*g6 + b6 // 6x6x6 cube starts at index 16
}

// getColorCode returns the ANSI escape sequence for the current rainbow color.
// Uses 24-bit true color if supported, otherwise falls back to 256-color mode.
func (s *CircleSpinner) getColorCode() string {
	r, g, b := s.getRGB()
	if s.useTrueColor {
		return fmt.Sprintf("%s%d;%d;%dm", ansiTrueColorFg, r, g, b)
	}
	return fmt.Sprintf("%s%dm", ansi256ColorFg, rgbTo256(r, g, b))
}

// renderFrame draws the current animation frame to the canvas and returns it as a string.
// The frame consists of a trail of pixels following the circular path.
func (s *CircleSpinner) renderFrame() string {
	s.canvas.Reset()

	// Draw trail: i=0 is head, i=1..trailLength-1 trail behind
	numPixels := len(circlePixels)
	for i := range trailLength {
		trailIdx := (s.frameIdx - i + numPixels) % numPixels
		pos := circlePixels[trailIdx]
		s.canvas.Set(pos[0], pos[1])
	}

	return s.canvas.String()
}
