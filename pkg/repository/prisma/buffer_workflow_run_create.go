package prisma

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func newCreateWorkflowRunBuffer(shared *sharedRepository, conf buffer.ConfigFileBuffer) (*buffer.TenantBufferManager[*repository.CreateWorkflowRunOpts, dbsqlc.WorkflowRun], error) {
	workflowRunBufOpts := buffer.TenantBufManagerOpts[*repository.CreateWorkflowRunOpts, dbsqlc.WorkflowRun]{
		Name:       "workflow_run_buffer",
		OutputFunc: shared.bulkCreateWorkflowRuns,
		SizeFunc:   sizeOfWorkflowRunData,
		L:          shared.l,
		V:          shared.v,
		Config:     conf,
	}

	manager, err := buffer.NewTenantBufManager(workflowRunBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func sizeOfWorkflowRunData(data *repository.CreateWorkflowRunOpts) int {
	size := 0

	size += len(data.InputData)
	size += len(data.AdditionalMetadata)
	size += len(*data.DisplayName)
	return size
}

func (w *sharedRepository) bulkCreateWorkflowRuns(ctx context.Context, opts []*repository.CreateWorkflowRunOpts) ([]*dbsqlc.WorkflowRun, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("no workflow runs to create")
	}

	w.l.Debug().Msgf("bulk creating %d workflow runs", len(opts))

	return createNewWorkflowRuns(ctx, w.pool, w.queries, w.l, opts)
}
