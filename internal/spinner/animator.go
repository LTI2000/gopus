// Package spinner provides a minimal loading animation with customizable visuals.
// The package separates animation timing (Animator) from visual rendering (Animation interface),
// allowing different spinner styles to be plugged in.
package spinner

import (
	"context"
	"time"
)

// Animation defines the interface for spinner visual behavior.
// Implementations control all visual aspects including color, frame rendering,
// terminal state management, and output. The Animator only handles timing and lifecycle.
type Animation interface {
	// Start is called when the animation begins.
	// It should handle any setup (e.g., hiding cursor) and render the initial frame.
	Start()

	// Stop is called when the animation ends.
	// It should handle any cleanup (e.g., clearing line, showing cursor, resetting colors).
	Stop()

	// Render advances the animation state and prints the current frame,
	// including any color codes and positioning.
	Render()

	// FrameCount returns the total number of frames in one complete animation cycle.
	FrameCount() int
}

// Animator manages the animation loop and timing for a spinner.
// It delegates all visual rendering and terminal output to an Animation implementation,
// handling only the goroutine lifecycle and frame timing.
type Animator struct {
	interval  time.Duration      // time between frames
	cancel    context.CancelFunc // cancels the animation goroutine
	done      chan struct{}      // signals animation goroutine has exited
	animation Animation          // the visual implementation
}

// NewAnimator creates a new Animator with the given Animation implementation.
// The default frame interval is 80ms (~12.5 FPS).
func NewAnimator(animation Animation) *Animator {
	return &Animator{
		interval:  80 * time.Millisecond,
		animation: animation,
	}
}

// Start begins the animation in a background goroutine.
// If the animation is already running, this is a no-op.
func (a *Animator) Start() {
	if a.cancel != nil {
		return // already running
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	go a.run(ctx)
}

// run is the animation loop goroutine. It calls Animation.Start() once,
// then calls Animation.Render() on each tick until the context is cancelled,
// at which point it calls Animation.Stop() and exits.
func (a *Animator) run(ctx context.Context) {
	defer close(a.done)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	a.animation.Start()

	for {
		select {
		case <-ctx.Done():
			a.animation.Stop()
			return
		case <-ticker.C:
			a.animation.Render()
		}
	}
}

// Stop stops the animation and waits for the goroutine to exit.
// If the animation is not running, this is a no-op.
func (a *Animator) Stop() {
	if a.cancel == nil {
		return // not running
	}

	a.cancel()
	<-a.done
	a.cancel = nil
}
