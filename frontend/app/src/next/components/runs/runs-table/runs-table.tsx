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
import { V1TaskStatus, V1TaskSummary, V1WorkflowType } from '@/lib/api';
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

const defaultExcludedFilters: (keyof RunsFilters)[] = [];

export function RunsTable({
  onRowClick,
  selectedTaskId,
  onSelectionChange,
  onTriggerRunClick,
  excludedFilters = defaultExcludedFilters,
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

  useEffect(() => {
    if (Object.keys(rowSelection).length > 0 && !isPaused) {
      pause();
    }
  }, [pause, rowSelection, isPaused]);

  const selectedTasks = useMemo(() => {
    const directlySelectedRuns = runs.filter(
      (run) => rowSelection[run.metadata.id],
    );

    const flattenedDagChildren = runs
      .filter((run) => run.children?.length)
      .flatMap((dagRun) => {
        const allChildrenSelected =
          dagRun.children?.every((child) => rowSelection[child.metadata.id]) ??
          false;

        return dagRun?.children?.map((child) => ({
          parentDagId: dagRun.metadata.id,
          childTask: child,
          allSiblingsSelected: allChildrenSelected,
        }));
      })
      .filter((record): record is NonNullable<typeof record> =>
        Boolean(record),
      );

    const implicitlySelectedDagIds = flattenedDagChildren
      .filter((child) => child.allSiblingsSelected)
      .map((child) => child.parentDagId);

    const implicitlySelectedDags = runs.filter((run) =>
      implicitlySelectedDagIds.includes(run.metadata.id),
    );

    const allSelectedDags = Array.from(
      new Set([...implicitlySelectedDags, ...directlySelectedRuns]),
    );

    // Find individual child tasks that are selected but whose parent DAG is not fully selected
    // Doing this so we can simplify handling of bulk actions (e.g. if all of the tasks in a DAG
    // are selected, we can just select the DAG instead of each individual task)
    const individuallySelectedChildren = flattenedDagChildren
      .filter(
        (child) =>
          !child.allSiblingsSelected &&
          rowSelection[child.childTask.metadata.id],
      )
      .map((analysis) => analysis.childTask);

    return [...individuallySelectedChildren, ...allSelectedDags];
  }, [rowSelection, runs]);

  const canCancel = useMemo(() => {
    return selectedTasks.some(
      (t) =>
        t.status === V1TaskStatus.RUNNING || t.status === V1TaskStatus.QUEUED,
    );
  }, [selectedTasks]);

  const canReplay = useMemo(() => {
    return selectedTasks.length > 0;
  }, [selectedTasks]);

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
  };

  const clearSelection = useCallback(() => {
    setSelectAll(false);
    setRowSelection({});
  }, [setSelectAll, setRowSelection]);

  useEffect(() => {
    clearSelection();
  }, [filters, clearSelection]);

  const navigate = useNavigate();

  const handleRowDoubleClick = useCallback(
    (row: V1TaskSummary) => {
      navigate(ROUTES.runs.detail(tenantId, row.workflowRunExternalId || ''));
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
            <span className="text-muted-foreground">
              {selectAll
                ? count.toLocaleString()
                : numSelectedRows.toLocaleString()}{' '}
              of {count.toLocaleString()} runs selected
            </span>
          ) : (
            <span className="text-muted-foreground">
              {count.toLocaleString()} runs
            </span>
          )}

          {count > 0 && !selectAll && allowSelection ? (
            <Button
              variant="ghost"
              size="sm"
              className="ml-2 h-6 px-2"
              onClick={() => setSelectAll(true)}
            >
              Select All
            </Button>
          ) : null}
          {numSelectedRows > 0 || selectAll ? (
            <Button
              variant="ghost"
              size="sm"
              className="ml-2 h-6 px-2"
              onClick={clearSelection}
            >
              Clear Selection
            </Button>
          ) : null}
        </div>
        {showActions ? (
          !selectAll ? (
            <div className="flex gap-2">
              <Button
                tooltip={
                  numSelectedRows === 0
                    ? 'No runs selected'
                    : canReplay
                      ? 'Replay the selected runs'
                      : 'Cannot replay the selected runs'
                }
                variant="outline"
                size="sm"
                disabled={!canReplay || replay.isPending}
                onClick={async () =>
                  replay.mutateAsync({ tasks: selectedTasks })
                }
              >
                <MdOutlineReplay className="h-4 w-4" />
                Replay
              </Button>
              <Button
                tooltip={
                  numSelectedRows === 0
                    ? 'No runs selected'
                    : canCancel
                      ? 'Cancel the selected runs'
                      : 'Cannot cancel the selected runs because they are not running or queued'
                }
                variant="outline"
                size="sm"
                disabled={!canCancel || cancel.isPending}
                onClick={async () =>
                  cancel.mutateAsync({ tasks: selectedTasks })
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
          )
        ) : null}
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
              {hasFilters ? (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => clearAllFilters()}
                >
                  Clear Filters
                </Button>
              ) : null}
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
      {showPagination ? (
        <Pagination className="justify-between flex flex-row">
          <PageSizeSelector />
          <PageSelector variant="dropdown" />
        </Pagination>
      ) : null}

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
