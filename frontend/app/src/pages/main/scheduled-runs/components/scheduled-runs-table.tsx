import { DataTable } from '@/components/molecules/data-table/data-table';
import { useState } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { ScheduledWorkflows } from '@/lib/api';
import {
  ToolbarFilters,
  ToolbarType,
} from '@/components/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/ui/button';
import { columns } from './scheduled-runs-columns';
import { DeleteScheduledRun } from './delete-scheduled-runs';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { DocsButton } from '@/components/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';
import { useScheduledRuns } from '../hooks/use-scheduled-runs';
import {
  ScheduledRunColumn,
  workflowKey,
  statusKey,
  metadataKey,
} from './scheduled-runs-columns';
import { workflowRunStatusFilters } from '../../workflow-runs/hooks/use-toolbar-filters';

interface ScheduledWorkflowRunsTableProps {
  createdAfter?: string;
  createdBefore?: string;
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  initColumnVisibility?: VisibilityState;
  filterVisibility?: { [key: string]: boolean };
  showMetrics?: boolean;
}

export function ScheduledRunsTable({
  workflowId,
  initColumnVisibility = {
    createdAt: false,
  },
  filterVisibility = {},
  parentWorkflowRunId,
  parentStepRunId,
}: ScheduledWorkflowRunsTableProps) {
  const { tenantId } = useCurrentTenantId();
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [selectedAdditionalMetaJobId, setSelectedAdditionalMetaJobId] =
    useState<string | null>(null);

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);

  const {
    scheduledRuns,
    numPages,
    isLoading,
    refetch,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    workflowKeyFilters,
    isRefetching,
    resetFilters,
  } = useScheduledRuns({
    key: 'table',
    workflowId,
    parentWorkflowRunId,
    parentStepRunId,
  });

  const filters: ToolbarFilters = [
    {
      columnId: workflowKey,
      title: ScheduledRunColumn.workflow,
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: statusKey,
      title: ScheduledRunColumn.status,
      options: workflowRunStatusFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: metadataKey,
      title: ScheduledRunColumn.metadata,
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  const actions = [
    <Button
      key="schedule-run"
      onClick={() => setTriggerWorkflow(true)}
      className="h-8 border px-3"
    >
      Schedule Run
    </Button>,
  ];

  const [showScheduledRunRevoke, setShowScheduledRunRevoke] = useState<
    ScheduledWorkflows | undefined
  >(undefined);

  return (
    <>
      <DeleteScheduledRun
        scheduledRun={showScheduledRunRevoke}
        setShowScheduledRunRevoke={setShowScheduledRunRevoke}
        onSuccess={() => {
          refetch();
          setShowScheduledRunRevoke(undefined);
        }}
      />
      <TriggerWorkflowForm
        defaultTimingOption="schedule"
        defaultWorkflow={undefined}
        show={triggerWorkflow}
        onClose={() => setTriggerWorkflow(false)}
      />

      <DataTable
        emptyState={
          <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
            <p className="text-lg font-semibold">No runs found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['scheduled-runs']}
                size="full"
                variant="outline"
                label="Learn about scheduled runs"
              />
            </div>
          </div>
        }
        error={error}
        isLoading={isLoading}
        columns={columns({
          tenantId,
          onDeleteClick: (row) => {
            setShowScheduledRunRevoke(row);
          },
          selectedAdditionalMetaJobId,
          handleSetSelectedAdditionalMetaJobId: setSelectedAdditionalMetaJobId,
        })}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={scheduledRuns}
        filters={filters}
        rightActions={actions}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={numPages}
        showColumnToggle={true}
        columnKeyToName={ScheduledRunColumn}
        refetchProps={{
          isRefetching,
          onRefetch: refetch,
        }}
        onResetFilters={resetFilters}
        showSelectedRows={false}
      />
    </>
  );
}
