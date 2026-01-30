// Package spinner provides a minimal loading animation.
package spinner

import (
	"context"
	"fmt"
	"time"
)

// Spinner frames using braille characters
var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner represents an animated loading spinner.
type Spinner struct {
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// New creates a new spinner.
func New() *Spinner {
	return &Spinner{
		interval: 80 * time.Millisecond,
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
	fmt.Print(frames[frameIdx])

	for {
		select {
		case <-ctx.Done():
			// Clear the spinner character
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			frameIdx = (frameIdx + 1) % len(frames)
			fmt.Printf("\r%s", frames[frameIdx])
		}
	}
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
