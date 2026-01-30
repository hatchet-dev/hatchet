package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

type LogsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

func NewLogsClient(api *rest.ClientWithResponses, tenantId uuid.UUID) *LogsClient {
	return &LogsClient{api: api, tenantId: tenantId}
}

func (l *LogsClient) List(ctx context.Context, taskRunId uuid.UUID, opts *rest.V1LogLineListParams) (*rest.V1LogLineList, error) {
	resp, err := l.api.V1LogLineListWithResponse(ctx, taskRunId, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list logs")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
