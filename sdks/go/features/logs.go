package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// LogsClient provides methods for interacting with Hatchet's logs API. To write logs
// from inside a task, use the task context's Log method instead.
type LogsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewLogsClient creates a new LogsClient
func NewLogsClient(api *rest.ClientWithResponses, tenantId string) *LogsClient {
	return &LogsClient{api: api, tenantId: uuid.MustParse(tenantId)}
}

// List retrieves the log lines for a given task run.
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
