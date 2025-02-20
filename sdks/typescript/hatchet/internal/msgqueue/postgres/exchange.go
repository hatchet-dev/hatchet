package postgres

import (
	"context"
)

func (p *PostgresMessageQueue) addTenantExchangeMessage(ctx context.Context, tenantId string, msgBytes []byte) error {
	// determine if the exchange message is greater than 8kb
	if len(msgBytes) > 8000 {
		// if the message is greater than 8kb, store the message in the database
		return p.repo.AddMessage(ctx, tenantId, msgBytes)
	}

	// if the message is less than 8kb, publish the message to the channel
	return p.repo.Notify(ctx, tenantId, string(msgBytes))
}
