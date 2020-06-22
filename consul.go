package teacup

import (
	"context"

	consul "github.com/hashicorp/consul/api"
)

// Consul returns a consul client ready to use. The first access to Consul will dial the Consul server
// and the provided context is used to control things like timeouts.
func (t *Teacup) Consul(ctx context.Context) (*consul.Client, error) {
	if t.consulClient != nil {
		return t.consulClient, nil
	}
	config := consul.DefaultConfig()
	config.Address = t.ServiceAddr(ctx, "consul", 8500)
	return consul.NewClient(config)
}
