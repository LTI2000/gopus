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

// ANSI escape codes for cursor visibility and color reset
const (
	hideCursor = "\033[?25l"
	showCursor = "\033[?25h"
	resetColor = "\033[0m"
)

// Phase shifts for RGB components (evenly distributed over 2π)
const (
	redPhase   = 0.0                 // 0 degrees
	greenPhase = 2.0 * math.Pi / 3.0 // 120 degrees
	bluePhase  = 4.0 * math.Pi / 3.0 // 240 degrees
)

// color256Cycle defines a sequence of 256-color palette indices for smooth
// color cycling. These are selected from the 6x6x6 color cube (indices 16-231)
// to create a rainbow-like progression.
var color256Cycle = []int{
	196, 202, 208, 214, 220, 226, // red -> orange -> yellow
	190, 154, 118, 82, 46, // yellow -> green
	47, 48, 49, 50, 51, // green -> cyan
	45, 39, 33, 27, 21, // cyan -> blue
	57, 93, 129, 165, 201, // blue -> magenta
	200, 199, 198, 197, // magenta -> red
}

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
	colorIdx     int     // current index in color256Cycle for 256-color mode
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
		colorIdx:     0,
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

	// Hide cursor before starting animation
	fmt.Print(hideCursor)

	go s.run(ctx)
}

// run animates the spinner until the context is cancelled.
func (s *Spinner) run(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIdx := 0
	s.renderFrame(frameIdx)

	for {
		select {
		case <-ctx.Done():
			// Clear the spinner characters, reset color, and show cursor
			fmt.Print("\r\033[K" + resetColor + showCursor)
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
	if s.useTrueColor {
		// Advance phase by a small increment each frame
		// Complete cycle every ~3 seconds at 80ms interval (~37.5 frames)
		s.phase += 2.0 * math.Pi / 37.5
		if s.phase >= 2.0*math.Pi {
			s.phase -= 2.0 * math.Pi
		}
	} else {
		// Advance through the 256-color cycle
		s.colorIdx = (s.colorIdx + 1) % len(color256Cycle)
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

// getColorCode returns the ANSI escape sequence for the current color.
func (s *Spinner) getColorCode() string {
	if s.useTrueColor {
		// Get current RGB color from sinusoidal cycling
		r, g, b := s.getRGB()
		// ANSI 24-bit true color foreground (ESC[38;2;r;g;bm)
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}
	// ANSI 256-color foreground (ESC[38;5;Nm)
	return fmt.Sprintf("\033[38;5;%dm", color256Cycle[s.colorIdx])
}

// renderFrame renders a single frame of the spinner animation with a trail.
func (s *Spinner) renderFrame(frameIdx int) {
	s.canvas.Reset()

	// Draw the trail (head + trailing dots)
	numPixels := len(circlePixels)
	for i := 0; i < trailLength; i++ {
		// Calculate position for this trail segment
		// i=0 is the head, i=1,2,3 are trailing behind
		trailIdx := (frameIdx - i + numPixels) % numPixels
		pos := circlePixels[trailIdx]
		s.canvas.Set(pos[0], pos[1])
	}

	// Print with appropriate color escape sequence
	fmt.Printf("\r%s%s", s.getColorCode(), s.canvas.String())
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
