import { useRuns, RunsFilters } from '@/next/hooks/use-runs';
import { useEffect, useMemo, useState, useCallback } from 'react';
import { DataTable } from './data-table';
import { columns } from './columns';
import {
  Pagination,
  PageSizeSelector,
  PageSelector,
} from '@/next/components/ui/pagination';
import {
  FilterGroup,
  FilterSelect,
  FilterTaskSelect,
  FilterKeyValue,
  ClearFiltersButton,
  FilterWorkerSelect,
} from '@/next/components/ui/filters/filters';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';
import { RowSelectionState, OnChangeFn } from '@tanstack/react-table';
import { MdOutlineReplay, MdOutlineCancel } from 'react-icons/md';
import { Button } from '@/next/components/ui/button';
import { RunsBulkActionDialog } from './bulk-action-dialog';
import { Plus } from 'lucide-react';
import { ROUTES } from '@/next/lib/routes';
import { useNavigate } from 'react-router-dom';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';
import { WorkersProvider } from '@/next/hooks/use-workers';
interface RunsTableProps {
  onRowClick?: (row: V1TaskSummary) => void;
  selectedTaskId?: string;
  onSelectionChange?: (selectedRows: V1TaskSummary[]) => void;
  onTriggerRunClick?: () => void;
  excludedFilters?: (keyof RunsFilters)[];
  showPagination?: boolean;
  allowSelection?: boolean;
  showActions?: boolean;
}

export function RunsTable({
  onRowClick,
  selectedTaskId,
  onSelectionChange,
  onTriggerRunClick,
  excludedFilters = [],
  showPagination = true,
  allowSelection = true,
  showActions = true,
}: RunsTableProps) {
  const {
    data: runs,
    count,
    timeRange: { pause, isPaused },
    isLoading,
    filters: { filters, clearAllFilters },
    hasFilters,
    cancel,
    replay,
  } = useRuns();
  const { tenantId } = useCurrentTenantId();

  const [selectAll, setSelectAll] = useState(false);
  const [showBulkActionDialog, setShowBulkActionDialog] = useState<
    'replay' | 'cancel' | null
  >(null);

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [selectedTasks, setSelectedTasks] = useState<
    Map<string, V1TaskSummary>
  >(new Map());

  useEffect(() => {
    if (Object.keys(rowSelection).length > 0 && !isPaused) {
      pause();
    }
  }, [pause, rowSelection, isPaused]);

  const selectedRuns = useMemo(() => {
    return Array.from(selectedTasks.values());
  }, [selectedTasks]);

  const canCancel = useMemo(() => {
    return selectedRuns.some(
      (t) =>
        t.status === V1TaskStatus.RUNNING || t.status === V1TaskStatus.QUEUED,
    );
  }, [selectedRuns]);

  const canReplay = useMemo(() => {
    return selectedRuns.length > 0;
  }, [selectedRuns]);

  const additionalMetaOpts = useMemo(() => {
    if (!runs || runs.length === 0) {
      return [];
    }

    const allKeys = new Set<string>();
    runs.forEach((run) => {
      if (run.additionalMetadata) {
        Object.keys(run.additionalMetadata).forEach((key) => allKeys.add(key));
      }
    });

    return Array.from(allKeys).map((key) => ({
      label: key,
      value: key,
    }));
  }, [runs]);

  const numSelectedRows = useMemo(() => {
    return Object.keys(rowSelection).length;
  }, [rowSelection]);

  const handleSelectionChange: OnChangeFn<RowSelectionState> = (
    updaterOrValue,
  ) => {
    const newSelection =
      typeof updaterOrValue === 'function'
        ? updaterOrValue(rowSelection)
        : updaterOrValue;

    setRowSelection(newSelection);

    // Update the selected tasks map
    const newSelectedTasks = new Map();
    if (runs) {
      Object.keys(newSelection).forEach((taskId) => {
        const task = runs.find((run) => run.taskExternalId === taskId);
        if (task) {
          newSelectedTasks.set(taskId, task);
        }
      });
    }
    setSelectedTasks(newSelectedTasks);
  };

  const clearSelection = useCallback(() => {
    setSelectAll(false);
    setRowSelection({});
    setSelectedTasks(new Map());
  }, [setSelectAll, setRowSelection, setSelectedTasks]);

  useEffect(() => {
    clearSelection();
  }, [filters, clearSelection]);

  const navigate = useNavigate();

  const handleRowDoubleClick = useCallback(
    (row: V1TaskSummary) => {
      navigate(
        ROUTES.runs.detailWithSheet(tenantId, row.workflowRunExternalId || '', {
          type: 'task-detail',
          props: {
            selectedWorkflowRunId: row.workflowRunExternalId || '',
            selectedTaskId: row.taskExternalId,
          },
        }),
      );
    },
    [navigate, tenantId],
  );

  return (
    <>
      <FilterGroup>
        {!excludedFilters.includes('statuses') && (
          <FilterSelect<RunsFilters, V1TaskStatus[]>
            name="statuses"
            value={filters.statuses}
            placeholder="Status"
            multi
            options={[
              { label: 'Running', value: V1TaskStatus.RUNNING },
              { label: 'Completed', value: V1TaskStatus.COMPLETED },
              { label: 'Failed', value: V1TaskStatus.FAILED },
              { label: 'Cancelled', value: V1TaskStatus.CANCELLED },
              { label: 'Queued', value: V1TaskStatus.QUEUED },
            ]}
          />
        )}
        {!excludedFilters.includes('workflow_ids') && (
          <FilterTaskSelect<RunsFilters>
            name="workflow_ids"
            placeholder="Name"
            multi
          />
        )}
        {!excludedFilters.includes('is_root_task') && (
          <FilterSelect<RunsFilters, boolean>
            name="is_root_task"
            value={filters.is_root_task}
            placeholder="Only Root Tasks"
            options={[
              { label: 'Yes', value: true },
              { label: 'No', value: false },
            ]}
          />
        )}
        {!excludedFilters.includes('workflow_ids') && (
          <FilterTaskSelect<RunsFilters>
            name="workflow_ids"
            placeholder="Task Name"
            multi
          />
        )}
        {!excludedFilters.includes('worker_id') && (
          <WorkersProvider status="ACTIVE">
            <FilterWorkerSelect<RunsFilters>
              name="worker_id"
              placeholder="Worker"
              multi
            />
          </WorkersProvider>
        )}
        {!excludedFilters.includes('additional_metadata') && (
          <FilterKeyValue<RunsFilters>
            name="additional_metadata"
            placeholder="Metadata"
            options={additionalMetaOpts}
          />
        )}
        {/* IMPORTANT: Keep this count in sync with the number of filters above */}
        {excludedFilters.length < 4 && <ClearFiltersButton />}
      </FilterGroup>
      <div className="flex items-center justify-between">
        <div className="flex-1 text-sm text-muted-foreground">
          {numSelectedRows > 0 || selectAll ? (
            <>
              <span className="text-muted-foreground">
                {selectAll
                  ? count.toLocaleString()
                  : numSelectedRows.toLocaleString()}{' '}
                of {count.toLocaleString()} runs selected
              </span>
            </>
          ) : (
            <span className="text-muted-foreground">
              {count.toLocaleString()} runs
            </span>
          )}

          {count > 0 && !selectAll && allowSelection && (
            <Button
              variant="ghost"
              size="sm"
              className="ml-2 h-6 px-2"
              onClick={() => setSelectAll(true)}
            >
              Select All
            </Button>
          )}
          {(numSelectedRows > 0 || selectAll) && (
            <Button
              variant="ghost"
              size="sm"
              className="ml-2 h-6 px-2"
              onClick={clearSelection}
            >
              Clear Selection
            </Button>
          )}
        </div>
        {showActions &&
          (!selectAll ? (
            <div className="flex gap-2">
              <Button
                tooltip={
                  numSelectedRows == 0
                    ? 'No runs selected'
                    : canReplay
                      ? 'Replay the selected runs'
                      : 'Cannot replay the selected runs'
                }
                variant="outline"
                size="sm"
                disabled={!canReplay || replay.isPending}
                onClick={async () =>
                  replay.mutateAsync({ tasks: selectedRuns })
                }
              >
                <MdOutlineReplay className="h-4 w-4" />
                Replay
              </Button>
              <Button
                tooltip={
                  numSelectedRows == 0
                    ? 'No runs selected'
                    : canCancel
                      ? 'Cancel the selected runs'
                      : 'Cannot cancel the selected runs because they are not running or queued'
                }
                variant="outline"
                size="sm"
                disabled={!canCancel || cancel.isPending}
                onClick={async () =>
                  cancel.mutateAsync({ tasks: selectedRuns })
                }
              >
                <MdOutlineCancel className="h-4 w-4" />
                Cancel
              </Button>
            </div>
          ) : (
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={replay.isPending}
                onClick={() => setShowBulkActionDialog('replay')}
              >
                <MdOutlineReplay className="h-4 w-4" />
                Replay All
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={cancel.isPending}
                onClick={() => setShowBulkActionDialog('cancel')}
              >
                <MdOutlineCancel className="h-4 w-4" />
                Cancel All
              </Button>
            </div>
          ))}
      </div>
      <DataTable
        columns={columns(onRowClick, selectAll, allowSelection)}
        data={runs || []}
        onDoubleClick={handleRowDoubleClick}
        emptyState={
          <div className="flex flex-col items-center justify-center gap-4 py-8">
            <p className="text-md">No runs found.</p>
            <p className="text-sm text-muted-foreground">
              Trigger a new run to get started.
            </p>
            <div className="flex flex-col gap-2">
              <Button size="sm" onClick={onTriggerRunClick}>
                <Plus className="h-4 w-4 mr-2" />
                Trigger Run
              </Button>
              {hasFilters && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => clearAllFilters()}
                >
                  Clear Filters
                </Button>
              )}
              <DocsButton
                doc={docs.home.running_tasks}
                titleOverride="Running Tasks"
              />
            </div>
          </div>
        }
        isLoading={isLoading}
        selectedTaskId={selectedTaskId}
        onRowClick={onRowClick}
        onSelectionChange={onSelectionChange}
        rowSelection={rowSelection}
        setRowSelection={handleSelectionChange}
        selectAll={selectAll}
        getSubRows={(row) => row.children || []}
      />
      {showPagination && (
        <Pagination className="justify-between flex flex-row">
          <PageSizeSelector />
          <PageSelector variant="dropdown" />
        </Pagination>
      )}

      <RunsBulkActionDialog
        open={!!showBulkActionDialog}
        onOpenChange={(open) =>
          setShowBulkActionDialog(open ? showBulkActionDialog : null)
        }
        action={showBulkActionDialog || 'replay'}
        isLoading={
          showBulkActionDialog === 'replay'
            ? replay.isPending
            : cancel.isPending
        }
        onConfirm={async () => {
          if (showBulkActionDialog === 'replay') {
            await replay.mutateAsync({ bulk: true });
          } else {
            await cancel.mutateAsync({ bulk: true });
          }
          setShowBulkActionDialog(null);
        }}
      />
    </>
  );
}
