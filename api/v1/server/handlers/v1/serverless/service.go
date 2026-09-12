// Package serverlessv1 serves the management API of the serverless operator: the endpoints a
// tenant registers and the tenant's sharding settings. It writes configuration only; the
// operator owns the status columns and the leases.
package serverlessv1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
)

const (
	// minSigningSecretLength matches the minLength of signingSecret in the OpenAPI spec. The
	// spec is not enforced by middleware, so the handlers check it themselves.
	minSigningSecretLength = 32

	pgUniqueViolation = "23505"
)

type V1ServerlessService struct {
	config *server.ServerConfig
}

func NewV1ServerlessService(config *server.ServerConfig) *V1ServerlessService {
	return &V1ServerlessService{
		config: config,
	}
}

// validateEndpointUrl applies the SSRF policy to a user-supplied URL at registration time.
// This is a UX check only: the operator's dial-time check remains the enforcement point.
func validateEndpointUrl(field, rawUrl string) error {
	if err := safeclient.ValidateEndpoint(rawUrl); err != nil {
		return fmt.Errorf("invalid %s: %s", field, err)
	}

	return nil
}

func validateSigningSecret(secret string) error {
	if len(secret) < minSigningSecretLength {
		return fmt.Errorf("signingSecret must be at least %d characters", minSigningSecretLength)
	}

	return nil
}

// marshalLabels turns the request's labels object into the JSON the repository stores. A nil
// map means the field was omitted and returns nil so the repository keeps its default.
func marshalLabels(labels *map[string]interface{}) ([]byte, error) {
	if labels == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(*labels)

	if err != nil {
		return nil, fmt.Errorf("labels must be a JSON object: %w", err)
	}

	return encoded, nil
}

// badRequestMessage returns the client-facing message for repository errors the caller can
// fix: field validation failures and a duplicate endpoint name. It returns false for anything
// else so those surface as internal errors.
func badRequestMessage(err error) (string, bool) {
	var detailed *hatcheterrors.DetailedError

	if errors.As(err, &detailed) && detailed.Code == 400 {
		return detailed.Description, true
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return "a serverless endpoint with this name already exists", true
	}

	return "", false
}
