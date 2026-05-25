package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SetupContext returns a context that is canceled when SIGINT or SIGTERM is received.
func SetupContext(parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)
	
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		// First signal cancels the context to allow graceful shutdown
		cancel()
		
		// If a second signal is received, force exit
		<-c
		os.Exit(1)
	}()
	
	return ctx
}
