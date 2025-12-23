import { TriggerWorkflowForm } from '../workflows/$workflow/components/trigger-workflow-form';
import { BulkDeleteScheduledRuns } from './components/bulk-delete-scheduled-runs';
import { BulkRescheduleScheduledRuns } from './components/bulk-reschedule-scheduled-runs';
import { DeleteScheduledRun } from './components/delete-scheduled-runs';
import { columns } from './components/scheduled-runs-columns';
import {
  ScheduledRunColumn,
  workflowKey,
  metadataKey,
} from './components/scheduled-runs-columns';
import { useScheduledRuns } from './hooks/use-scheduled-runs';
import { DocsButton } from '@/components/v1/docs/docs-button';
import {
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  ScheduledWorkflows,
  ScheduledWorkflowsBulkDeleteFilter,
} from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { RowSelectionState, VisibilityState } from '@tanstack/react-table';
import { Command } from 'lucide-react';
import { useState } from 'react';

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

export default function ScheduledRunsTable({
  workflowId,
  initColumnVisibility = {
    createdAt: false,
  },
  filterVisibility = {},
  parentWorkflowRunId,
  parentStepRunId,
}: ScheduledWorkflowRunsTableProps) {
  const { tenantId } = useCurrentTenantId();
  const { open } = useSidePanel();
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [selectedAdditionalMetaJobId, setSelectedAdditionalMetaJobId] =
    useState<string | null>(null);

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

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
    selectedWorkflowIds,
    selectedStatuses,
    selectedMetadata,
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
      columnId: metadataKey,
      title: ScheduledRunColumn.metadata,
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  const actions = [
    <Button
      key="schedule-run"
      onClick={() => setTriggerWorkflow(true)}
      variant="cta"
    >
      Schedule Run
    </Button>,
  ];

  const [showScheduledRunRevoke, setShowScheduledRunRevoke] = useState<
    ScheduledWorkflows | undefined
  >(undefined);

  const selectedIds = Object.entries(rowSelection)
    .filter(([, selected]) => !!selected)
    .map(([id]) => id);

  const formatCount = (n: number) => new Intl.NumberFormat().format(n);

  const [cancelParams, setCancelParams] = useState<{
    scheduledRunIds: string[];
    filter?: ScheduledWorkflowsBulkDeleteFilter;
  } | null>(null);
  const [rescheduleParams, setRescheduleParams] = useState<{
    scheduledRunIds: string[];
    filter?: ScheduledWorkflowsBulkDeleteFilter;
  } | null>(null);
  const [isActionsOpen, setIsActionsOpen] = useState(false);

  const effectiveWorkflowId = workflowId || selectedWorkflowIds[0];
  const hasActiveFilters =
    !!effectiveWorkflowId ||
    selectedStatuses.length > 0 ||
    selectedMetadata.length > 0 ||
    !!parentWorkflowRunId ||
    !!parentStepRunId;

  const actionFilter: ScheduledWorkflowsBulkDeleteFilter = hasActiveFilters
    ? {
        workflowId: effectiveWorkflowId,
        additionalMetadata:
          selectedMetadata.length > 0 ? selectedMetadata : undefined,
        parentWorkflowRunId,
        parentStepRunId,
      }
    : {};

  const deleteLabel =
    selectedIds.length > 0
      ? `Delete selected (${formatCount(selectedIds.length)})`
      : hasActiveFilters
        ? 'Delete filtered'
        : 'Delete all';

  const rescheduleLabel =
    selectedIds.length > 0
      ? `Reschedule selected (${formatCount(selectedIds.length)})`
      : hasActiveFilters
        ? 'Reschedule filtered'
        : 'Reschedule all';

  const leftActions = [
    <DropdownMenu
      key="scheduled-run-actions"
      open={isActionsOpen}
      onOpenChange={setIsActionsOpen}
    >
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" type="button">
          <Command className="cq-xl:hidden size-4" />
          <span className="cq-xl:inline hidden text-sm">Actions</span>
          <ChevronDownIcon className="cq-xl:inline ml-2 hidden size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="z-40">
        <DropdownMenuItem
          className="h-8 w-full cursor-pointer justify-start rounded-sm px-2 py-1.5 font-normal"
          onSelect={() => {
            setIsActionsOpen(false);
            if (selectedIds.length > 0) {
              setCancelParams({ scheduledRunIds: selectedIds });
            } else {
              setCancelParams({ scheduledRunIds: [], filter: actionFilter });
            }
          }}
        >
          {deleteLabel}
        </DropdownMenuItem>
        <DropdownMenuItem
          className="h-8 w-full cursor-pointer justify-start rounded-sm px-2 py-1.5 font-normal"
          onSelect={() => {
            setIsActionsOpen(false);
            if (selectedIds.length > 0) {
              setRescheduleParams({ scheduledRunIds: selectedIds });
            } else {
              setRescheduleParams({
                scheduledRunIds: [],
                filter: actionFilter,
              });
            }
          }}
        >
          {rescheduleLabel}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>,
  ];

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

      <BulkDeleteScheduledRuns
        open={!!cancelParams}
        scheduledRunIds={cancelParams?.scheduledRunIds ?? []}
        filter={cancelParams?.filter}
        onOpenChange={(open) => {
          if (!open) {
            setCancelParams(null);
          }
        }}
        onSuccess={() => {
          refetch();
          setRowSelection({});
          setCancelParams(null);
        }}
      />

      <BulkRescheduleScheduledRuns
        open={!!rescheduleParams}
        scheduledRunIds={rescheduleParams?.scheduledRunIds ?? []}
        filter={rescheduleParams?.filter}
        initialRuns={scheduledRuns}
        onOpenChange={(open) => {
          if (!open) {
            setRescheduleParams(null);
          }
        }}
        onSuccess={() => {
          refetch();
          setRowSelection({});
          setRescheduleParams(null);
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
          <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
            <p className="text-lg font-semibold">No runs found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['scheduled-runs']}
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
          onRescheduleClick: (row) => {
            setRescheduleParams({ scheduledRunIds: [row.metadata.id] });
          },
          selectedAdditionalMetaJobId,
          handleSetSelectedAdditionalMetaJobId: setSelectedAdditionalMetaJobId,
          onRowClick: (row) => {
            open({
              type: 'scheduled-run-details',
              content: {
                scheduledRun: row,
              },
            });
          },
        })}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={scheduledRuns}
        getRowId={(row) => row.metadata.id}
        filters={filters}
        leftActions={leftActions}
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
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        showSelectedRows={true}
      />
    </>
  );
}
