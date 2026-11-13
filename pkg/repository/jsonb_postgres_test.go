//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateJSONBAgreesWithPostgresOnSurrogateEscapes(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	for _, c := range surrogateEscapeCases {
		var stored string
		postgresErr := pool.QueryRow(context.Background(), `SELECT ($1::text)::jsonb::text`, c.json).Scan(&stored)

		require.Equal(t, c.unpaired, postgresErr != nil, "postgres on %s: %v", c.name, postgresErr)
		require.Equal(t, c.unpaired, ValidateJSONB([]byte(c.json), "field") != nil, "ValidateJSONB on %s", c.name)
	}
}
