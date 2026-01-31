// Package spinner provides a minimal loading animation.
package spinner

import (
	"context"
	"fmt"
	"time"

	"gopus/internal/canvas"
)

// edgePixels defines the clockwise path around the edges of a 4x4 braille display.
// Each pair is (x, y) coordinates.
var edgePixels = [][2]int{
	{0, 0}, {1, 0}, {2, 0}, {3, 0}, // top edge (left to right)
	{3, 1}, {3, 2}, {3, 3}, // right edge (top to bottom)
	{2, 3}, {1, 3}, {0, 3}, // bottom edge (right to left)
	{0, 2}, {0, 1}, // left edge (bottom to top)
}

// Spinner represents an animated loading spinner.
type Spinner struct {
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
	canvas   *canvas.Canvas
}

// New creates a new spinner.
func New() *Spinner {
	return &Spinner{
		interval: 80 * time.Millisecond,
		canvas:   canvas.New(4, 4), // 2 braille chars wide, 1 char tall
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

	frameIdx := 0
	s.renderFrame(frameIdx)

	for {
		select {
		case <-ctx.Done():
			// Clear the spinner characters
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			frameIdx = (frameIdx + 1) % len(edgePixels)
			s.renderFrame(frameIdx)
		}
	}
}

// renderFrame renders a single frame of the spinner animation.
func (s *Spinner) renderFrame(frameIdx int) {
	s.canvas.Reset()
	pos := edgePixels[frameIdx]
	s.canvas.Set(pos[0], pos[1])
	fmt.Printf("\r%s", s.canvas.String())
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
