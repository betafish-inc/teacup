package example

import (
	"context"

	"github.com/apex/log"
	"github.com/betafish-inc/teacup"
)

// The trivial example for a long running worker doesn't actually do anything useful.
type Hello struct {
}

func (h *Hello) Start(ctx context.Context, _ *teacup.Teacup) {
	log.Info("Hello World")
	<-ctx.Done() // Wait for the context to close
	log.Info("Worker is done")
}
