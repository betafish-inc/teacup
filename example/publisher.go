package example

import (
	"context"

	"github.com/betafish-inc/teacup"
)

// Pub sends a message to a subject.
type Pub struct {
}

func (p *Pub) Start(ctx context.Context, t *teacup.Teacup) {
	defer t.Stop()
	_ = t.Queue().Producer(ctx).Publish(ctx, "example", []byte("hello"))
 }
