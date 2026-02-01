// Package spinner provides a minimal loading animation.
package spinner

import (
	"context"
	"time"
)

// Spinner represents an animated loading spinner.
// It manages the animation loop and timing while delegating
// all visual rendering and output to the Animation implementation.
type Spinner struct {
	interval  time.Duration
	cancel    context.CancelFunc
	done      chan struct{}
	animation Animation
}

// New creates a new spinner with the default CircleAnimation.
func New() *Spinner {
	return NewWithAnimation(NewCircleAnimation())
}

// NewWithAnimation creates a new spinner with a custom animation.
func NewWithAnimation(animation Animation) *Spinner {
	return &Spinner{
		interval:  80 * time.Millisecond,
		animation: animation,
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

	// Let the animation handle startup and initial frame
	s.animation.Start()

	for {
		select {
		case <-ctx.Done():
			// Let the animation handle cleanup
			s.animation.Stop()
			return
		case <-ticker.C:
			s.animation.Render()
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
