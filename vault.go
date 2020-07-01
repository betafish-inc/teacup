package teacup

import (
	"context"

	vault "github.com/hashicorp/vault/api"
)

// Vault returns a vault client ready to use. The first access to Vault will dial the Vault server
// and the provided context is used to control things like timeouts.
func (t *Teacup) Vault(ctx context.Context) (*vault.Client, error) {
	if t.vaultClient != nil {
		return t.vaultClient, nil
	}
	config := vault.DefaultConfig()
	config.Address = t.ServiceAddr(ctx, "vault", 8200)
	// TODO login using AppRole https://learn.hashicorp.com/vault/identity-access-management/approle
	return vault.NewClient(config)
}
