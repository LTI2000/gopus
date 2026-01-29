// Package signal provides utilities for handling OS signals in a graceful manner.
package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// SetUpHandler sets up signal handling for graceful shutdown.
// It listens for SIGINT and SIGTERM signals and calls the provided action
// function with a cancellable context. When a signal is received, the context
// is cancelled and the program exits gracefully.
func SetUpHandler(action func(context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nGoodbye!")
		cancel()
		os.Exit(0)
	}()

	action(ctx)
}
