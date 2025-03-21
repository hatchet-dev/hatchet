package workflowruns

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunCreate(ctx echo.Context, request gen.V1WorkflowRunCreateRequestObject) (gen.V1WorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// make sure input can be marshalled and unmarshalled to input type
	inputBytes, err := json.Marshal(request.Body.Input)

	if err != nil {
		return gen.V1WorkflowRunCreate400JSONResponse(
			apierrors.NewAPIErrors("Invalid input"),
		), nil
	}

	var additionalMetadataBytes []byte

	if request.Body.AdditionalMetadata != nil {

		additionalMetadataBytes, err = json.Marshal(request.Body.AdditionalMetadata)

		if err != nil {
			return gen.V1WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("Invalid additional metadata"),
			), nil
		}
	}

	grpcReq := &contracts.TriggerWorkflowRunRequest{
		WorkflowName:       request.Body.WorkflowName,
		Input:              inputBytes,
		AdditionalMetadata: additionalMetadataBytes,
	}

	resp, err := t.proxyTrigger.Do(
		ctx.Request().Context(),
		tenant,
		grpcReq,
	)

	if err != nil {
		if e, ok := status.FromError(err); ok {
			switch e.Code() { // nolint: gocritic
			case codes.InvalidArgument:
				return gen.V1WorkflowRunCreate400JSONResponse(
					apierrors.NewAPIErrors(e.Message()),
				), nil
			}
		}

		return nil, err
	}

	// loop for workflow to be created in the OLAP database
	var rawWorkflowRun *v1.V1WorkflowRunPopulator
	retries := 0

	for retries < 10 {
		rawWorkflowRun, err = t.config.V1.OLAP().ReadWorkflowRun(
			ctx.Request().Context(),
			sqlchelpers.UUIDFromStr(resp.ExternalId),
		)

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		if err != nil && errors.Is(err, pgx.ErrNoRows) {
			retries++
			time.Sleep(1 * time.Second)
			continue
		}

		break
	}

	if rawWorkflowRun == nil || rawWorkflowRun.WorkflowRun == nil {
		return nil, fmt.Errorf("rawWorkflowRun not populated, we are likely seeing high latency in creating tasks")
	}

	if sqlchelpers.UUIDToStr(rawWorkflowRun.WorkflowRun.TenantID) != tenantId {
		return nil, fmt.Errorf("tenantId mismatch in the triggered workflow run")
	}

	details, err := t.getWorkflowRunDetails(
		ctx.Request().Context(),
		tenantId,
		rawWorkflowRun,
	)

	if err != nil {
		return nil, err
	}

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunCreate200JSONResponse(
		*details,
	), nil
}
