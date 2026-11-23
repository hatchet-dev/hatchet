//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

type failingSecretEncryption struct{ encryption.EncryptionService }

func (failingSecretEncryption) DecryptString(string, string) (string, error) {
	return "", errors.New("private KMS resource marker")
}

func TestSigningSecretFailureIsLoggedAndRedacted(t *testing.T) {
	var logs bytes.Buffer
	l := zerolog.New(&logs)
	repo := memrepo.New()
	tenantID := uuid.New()
	row := newEndpointRow(endpointSpec{tenantId: tenantID, name: "endpoint", enabled: true, actions: []string{"svc:a"}})
	repo.AddEndpoint(row)
	cache := newRoutingCache(tenantID, repo.Endpoints(), failingSecretEncryption{}, &l)
	require.NoError(t, cache.Load(context.Background()))
	_, cfg, err := cache.Route("svc:a")
	require.NoError(t, err)
	require.EqualError(t, cfg.secretErr, "could not decrypt signing secret")
	require.Contains(t, logs.String(), "private KMS resource marker")
	require.Contains(t, logs.String(), row.ID.String())

	out := deliverAction(context.Background(), nil, nil, cfg, nil)
	require.Equal(t, "could not decrypt signing secret", out.errMsg)
	_, healthErr := pollHealthcheck(context.Background(), nil, nil, cfg, catalogLimits{})
	require.EqualError(t, healthErr, "could not decrypt signing secret")
}
