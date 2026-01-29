// Package spinner provides a minimal loading animation.
package spinner

import (
	"fmt"
	"sync"
	"time"
)

// Minimal spinner frames
var frames = []string{
	"⠋\n",
	"⠙\n",
	"⠹\n",
	"⠸\n",
	"⠼\n",
	"⠴\n",
	"⠦\n",
	"⠧\n",
	"⠇\n",
	"⠏\n",
}

// Spinner represents an animated loading spinner.
type Spinner struct {
	frames   []string
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// New creates a new spinner.
func New() *Spinner {
	return &Spinner{
		frames:   frames,
		interval: 120 * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.doneCh)
		frameIdx := 0
		lineCount := 0

		for {
			select {
			case <-s.stopCh:
				// Clear the spinner
				s.clearLines(lineCount)
				return
			default:
				// Clear previous frame
				s.clearLines(lineCount)

				// Print current frame
				frame := s.frames[frameIdx]
				fmt.Print(frame)

				// Count lines in frame for clearing
				lineCount = countLines(frame)

				frameIdx = (frameIdx + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
}

// clearLines moves cursor up and clears lines.
func (s *Spinner) clearLines(n int) {
	for i := 0; i < n; i++ {
		// Move cursor up one line and clear it
		fmt.Print("\033[1A\033[2K")
	}
}

// countLines counts the number of newlines in a string.
func countLines(s string) int {
	count := 0
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count
}
