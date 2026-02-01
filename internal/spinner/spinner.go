// Package spinner provides a minimal loading animation.
package spinner

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

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

// Spinner represents an animated loading spinner.
type Spinner struct {
	interval     time.Duration
	cancel       context.CancelFunc
	done         chan struct{}
	canvas       *canvas.Canvas
	phase        float64 // current phase angle for RGB cycling (in radians)
	useTrueColor bool    // whether to use 24-bit true color or 256-color mode
}

// supportsTrueColor checks if the terminal supports 24-bit true color.
// macOS Terminal.app does not support true color, but iTerm2 and other
// modern terminals do. We detect this via the COLORTERM environment variable.
func supportsTrueColor() bool {
	colorterm := os.Getenv("COLORTERM")
	// COLORTERM=truecolor or COLORTERM=24bit indicates true color support
	return strings.Contains(colorterm, "truecolor") || strings.Contains(colorterm, "24bit")
}

// New creates a new spinner.
func New() *Spinner {
	return &Spinner{
		interval:     80 * time.Millisecond,
		canvas:       canvas.New(4, 4), // 2 braille chars wide, 1 char tall
		phase:        0,
		useTrueColor: supportsTrueColor(),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	// Already running
	if s.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	go s.run(ctx)
}

// run animates the spinner until the context is cancelled.
func (s *Spinner) run(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Hide cursor before starting animation
	fmt.Print(ansiHideCursor)

	frameIdx := 0
	s.renderFrame(frameIdx)

	for {
		select {
		case <-ctx.Done():
			// Return to start of line, clear the spinner characters, reset color, and show cursor
			fmt.Print(carriageReturn + ansiClearLine + ansiResetColor + ansiShowCursor)
			return
		case <-ticker.C:
			frameIdx = (frameIdx + 1) % len(circlePixels)
			s.updateColor()
			s.renderFrame(frameIdx)
		}
	}
}

// updateColor advances the color for the next frame.
func (s *Spinner) updateColor() {
	// Advance phase by a small increment each frame
	// Complete cycle every ~3 seconds at 80ms interval (~37.5 frames)
	s.phase += 2.0 * math.Pi / 37.5
	if s.phase >= 2.0*math.Pi {
		s.phase -= 2.0 * math.Pi
	}
}

// getRGB calculates the current RGB values using sinusoidal functions
// with evenly distributed phase shifts for each component.
func (s *Spinner) getRGB() (r, g, b int) {
	// Use sin function shifted to range [0, 1] then scaled to [0, 255]
	// sin(x) ranges from -1 to 1, so (sin(x) + 1) / 2 ranges from 0 to 1
	r = int((math.Sin(s.phase+redPhase) + 1.0) / 2.0 * 255.0)
	g = int((math.Sin(s.phase+greenPhase) + 1.0) / 2.0 * 255.0)
	b = int((math.Sin(s.phase+bluePhase) + 1.0) / 2.0 * 255.0)
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
func (s *Spinner) getColorCode() string {
	// Get current RGB color from sinusoidal cycling
	r, g, b := s.getRGB()
	if s.useTrueColor {
		// ANSI 24-bit true color foreground (ESC[38;2;r;g;bm)
		return fmt.Sprintf("%s%d;%d;%dm", ansiTrueColorFg, r, g, b)
	} else {
		// ANSI 256-color foreground (ESC[38;5;Nm)
		// Convert RGB to 6x6x6 cube index
		return fmt.Sprintf("%s%dm", ansi256ColorFg, rgbTo256(r, g, b))
	}
}

// renderFrame renders a single frame of the spinner animation with a trail.
func (s *Spinner) renderFrame(frameIdx int) {
	s.canvas.Reset()

	// Draw the trail (head + trailing dots)
	numPixels := len(circlePixels)
	for i := range trailLength {
		// Calculate position for this trail segment
		// i=0 is the head, i=1,2,3 are trailing behind
		trailIdx := (frameIdx - i + numPixels) % numPixels
		pos := circlePixels[trailIdx]
		s.canvas.Set(pos[0], pos[1])
	}

	// Use carriage return to go back to start of line, print with color
	fmt.Printf("%s%s%s", carriageReturn, s.getColorCode(), s.canvas.String())
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	if s.cancel == nil {
		return
	}

	s.cancel()
	<-s.done
	s.cancel = nil
}
