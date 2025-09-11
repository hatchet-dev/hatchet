package workflowruns

import (
	"sort"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetTimings(ctx echo.Context, request gen.V1WorkflowRunGetTimingsRequestObject) (gen.V1WorkflowRunGetTimingsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	rawWorkflowRun := ctx.Get("v1-workflow-run").(*v1.V1WorkflowRunPopulator)

	workflowRun := rawWorkflowRun.WorkflowRun

	var depth int32 = 0

	if request.Params.Depth != nil {
		depth = int32(*request.Params.Depth)
	}

	if depth > 10 {
		return gen.V1WorkflowRunGetTimings400JSONResponse(
			apierrors.NewAPIErrors("depth must be less than or equal to 10"),
		), nil
	}

	taskTimings, idsToDepths, err := t.config.V1.OLAP().GetTaskTimings(
		ctx.Request().Context(),
		tenantId,
		workflowRun.ExternalID,
		depth,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskTimings(taskTimings, idsToDepths)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunGetTimings200JSONResponse(
		gen.V1TaskTimingList{
			Rows: DFSOrder(result),
		},
	), nil
}

func DFSOrder(tasks []gen.V1TaskTiming) []gen.V1TaskTiming {
	// build set of task IDs
	idSet := make(map[openapi_types.UUID]struct{}, len(tasks))
	for _, t := range tasks {
		idSet[t.TaskExternalId] = struct{}{}
	}

	// group children and collect roots
	children := make(map[openapi_types.UUID][]gen.V1TaskTiming)
	var roots []gen.V1TaskTiming
	for _, t := range tasks {
		if t.ParentTaskExternalId != nil {
			if _, ok := idSet[*t.ParentTaskExternalId]; ok {
				children[*t.ParentTaskExternalId] = append(children[*t.ParentTaskExternalId], t)
				continue
			}
		}
		roots = append(roots, t)
	}

	// sort roots by queuedAt, then taskId
	sort.SliceStable(roots, func(i, j int) bool {
		qi, qj := roots[i].QueuedAt, roots[j].QueuedAt
		if qi != nil && qj != nil {
			if !qi.Equal(*qj) {
				return qi.Before(*qj)
			}
		} else if qi != nil {
			return true
		} else if qj != nil {
			return false
		}
		return roots[i].TaskId < roots[j].TaskId
	})

	// sort each child list the same way
	for parent, childs := range children {
		sort.SliceStable(childs, func(i, j int) bool {
			qi, qj := childs[i].QueuedAt, childs[j].QueuedAt
			if qi != nil && qj != nil {
				if !qi.Equal(*qj) {
					return qi.Before(*qj)
				}
			} else if qi != nil {
				return true
			} else if qj != nil {
				return false
			}
			return childs[i].TaskId < childs[j].TaskId
		})
		children[parent] = childs
	}

	// depth-first traversal
	var result []gen.V1TaskTiming
	var visit func(gen.V1TaskTiming)
	visit = func(t gen.V1TaskTiming) {
		result = append(result, t)
		for _, child := range children[t.TaskExternalId] {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}

	return result
}
