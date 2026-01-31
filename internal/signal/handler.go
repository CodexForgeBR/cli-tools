// Package signal provides signal handling for graceful shutdown of the ralph-loop CLI.
//
// The SetupSignalHandler function registers handlers for SIGINT and SIGTERM,
// allowing the application to respond to interruptions by calling cleanup callbacks
// and canceling the provided context.
package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler registers SIGINT and SIGTERM handlers.
// When a signal is received, it calls the onInterrupt callback (if non-nil),
// then cancels the context.
//
// This function starts a goroutine that listens for signals. The goroutine
// terminates when either a signal is received or the context is canceled.
//
// Parameters:
//   - ctx: The context to monitor for cancellation
//   - cancel: The cancel function to call when a signal is received
//   - onInterrupt: Optional callback to execute before canceling context
//
// Example usage:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	signal.SetupSignalHandler(ctx, cancel, func() {
//	    fmt.Println("Received interrupt signal, cleaning up...")
//	})
func SetupSignalHandler(ctx context.Context, cancel context.CancelFunc, onInterrupt func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			if onInterrupt != nil {
				onInterrupt()
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}()
}
