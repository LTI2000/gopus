// Package spinner provides a minimal loading animation.
package spinner

import (
	"fmt"
	"math"
	"os"
	"strings"

	"gopus/internal/canvas"
)

// ANSI escape codes for cursor visibility, color reset, and line clearing
const (
	ansiEscape      = "\033["
	ansiHideCursor  = ansiEscape + "?25l"
	ansiShowCursor  = ansiEscape + "?25h"
	ansiResetColor  = ansiEscape + "0m"
	ansiClearLine   = ansiEscape + "K"
	ansiTrueColorFg = ansiEscape + "38;2;" // followed by r;g;bm
	ansi256ColorFg  = ansiEscape + "38;5;" // followed by Nm
	carriageReturn  = "\r"
)

// Animation defines the interface for spinner animation behavior.
// Implementations control all visual aspects including color, frame rendering,
// terminal state management, and output. The Spinner only handles timing and lifecycle.
type Animation interface {
	// Start is called when the animation begins.
	// It should handle any setup (e.g., hiding cursor) and print the initial frame.
	Start()

	// Stop is called when the animation ends.
	// It should handle any cleanup (e.g., clearing line, showing cursor).
	Stop()

	// Render advances the animation state and prints the current frame,
	// including any color codes and positioning.
	Render()

	// FrameCount returns the total number of frames in the animation cycle.
	FrameCount() int
}

// Phase shifts for RGB components (evenly distributed over 2π)
const (
	redPhase   = 0.0                 // 0 degrees
	greenPhase = 2.0 * math.Pi / 3.0 // 120 degrees
	bluePhase  = 4.0 * math.Pi / 3.0 // 240 degrees
)

// trailLength is the number of dots in the trail (including the head).
const trailLength = 4

// circlePixels defines a circular path on a 4x4 braille display.
// The path approximates a circle using the available pixel positions.
//
// Grid layout (0,0 is top-left):
//
//	  0   1   2   3
//	0     *   *
//	1 *           *
//	2 *           *
//	3     *   *
//
// Path goes clockwise starting from top-center.
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

// CircleAnimation implements the Animation interface with a circular
// braille pattern and rainbow color cycling.
type CircleAnimation struct {
	canvas       *canvas.Canvas
	phase        float64 // current phase angle for RGB cycling (in radians)
	frameIdx     int     // current frame index in the animation cycle
	useTrueColor bool    // whether to use 24-bit true color or 256-color mode
}

// NewCircleAnimation creates a new CircleAnimation with default settings.
func NewCircleAnimation() *CircleAnimation {
	return &CircleAnimation{
		canvas:       canvas.New(4, 4), // 2 braille chars wide, 1 char tall
		phase:        0,
		frameIdx:     0,
		useTrueColor: supportsTrueColor(),
	}
}

// supportsTrueColor checks if the terminal supports 24-bit true color.
// macOS Terminal.app does not support true color, but iTerm2 and other
// modern terminals do. We detect this via the COLORTERM environment variable.
func supportsTrueColor() bool {
	colorterm := os.Getenv("COLORTERM")
	// COLORTERM=truecolor or COLORTERM=24bit indicates true color support
	return strings.Contains(colorterm, "truecolor") || strings.Contains(colorterm, "24bit")
}

// Start hides the cursor and prints the initial frame.
func (a *CircleAnimation) Start() {
	fmt.Print(ansiHideCursor)
	a.Render()
}

// Stop clears the line and shows the cursor.
func (a *CircleAnimation) Stop() {
	fmt.Print(carriageReturn + ansiClearLine + ansiResetColor + ansiShowCursor)
}

// Render advances the animation state and prints the current frame.
func (a *CircleAnimation) Render() {
	// Render the current frame
	frame := a.renderFrame()
	colorCode := a.getColorCode()
	fmt.Printf("%s%s%s", carriageReturn, colorCode, frame)

	// Advance state for next frame
	a.frameIdx = (a.frameIdx + 1) % len(circlePixels)
	a.advanceColor()
}

// FrameCount returns the total number of frames in the animation cycle.
func (a *CircleAnimation) FrameCount() int {
	return len(circlePixels)
}

// advanceColor advances the color for the next frame.
func (a *CircleAnimation) advanceColor() {
	// Advance phase by a small increment each frame
	// Complete cycle every ~3 seconds at 80ms interval (~37.5 frames)
	a.phase += 2.0 * math.Pi / 37.5
	if a.phase >= 2.0*math.Pi {
		a.phase -= 2.0 * math.Pi
	}
}

// getRGB calculates the current RGB values using sinusoidal functions
// with evenly distributed phase shifts for each component.
func (a *CircleAnimation) getRGB() (r, g, b int) {
	// Use sin function shifted to range [0, 1] then scaled to [0, 255]
	// sin(x) ranges from -1 to 1, so (sin(x) + 1) / 2 ranges from 0 to 1
	r = int((math.Sin(a.phase+redPhase) + 1.0) / 2.0 * 255.0)
	g = int((math.Sin(a.phase+greenPhase) + 1.0) / 2.0 * 255.0)
	b = int((math.Sin(a.phase+bluePhase) + 1.0) / 2.0 * 255.0)
	return r, g, b
}

// rgbTo256 converts RGB values (0-255) to an ANSI 256-color palette index
// using the 6x6x6 color cube (indices 16-231).
// The cube uses values 0-5 for each component, so we scale from 0-255.
func rgbTo256(r, g, b int) int {
	// Convert 0-255 range to 0-5 range for the 6x6x6 cube
	// Using integer division: (value * 5 + 127) / 255 gives better rounding
	r6 := (r*5 + 127) / 255
	g6 := (g*5 + 127) / 255
	b6 := (b*5 + 127) / 255
	// 6x6x6 cube starts at index 16
	return 16 + 36*r6 + 6*g6 + b6
}

// getColorCode returns the ANSI escape sequence for the current color.
func (a *CircleAnimation) getColorCode() string {
	// Get current RGB color from sinusoidal cycling
	r, g, b := a.getRGB()
	if a.useTrueColor {
		// ANSI 24-bit true color foreground (ESC[38;2;r;g;bm)
		return fmt.Sprintf("%s%d;%d;%dm", ansiTrueColorFg, r, g, b)
	}
	// ANSI 256-color foreground (ESC[38;5;Nm)
	// Convert RGB to 6x6x6 cube index
	return fmt.Sprintf("%s%dm", ansi256ColorFg, rgbTo256(r, g, b))
}

// renderFrame renders a single frame of the spinner animation with a trail.
func (a *CircleAnimation) renderFrame() string {
	a.canvas.Reset()

	// Draw the trail (head + trailing dots)
	numPixels := len(circlePixels)
	for i := range trailLength {
		// Calculate position for this trail segment
		// i=0 is the head, i=1,2,3 are trailing behind
		trailIdx := (a.frameIdx - i + numPixels) % numPixels
		pos := circlePixels[trailIdx]
		a.canvas.Set(pos[0], pos[1])
	}

	return a.canvas.String()
}
