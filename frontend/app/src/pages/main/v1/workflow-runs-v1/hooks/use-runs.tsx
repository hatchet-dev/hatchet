import { queries, V1TaskSummary, V1TaskStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';
import { RowSelectionState, PaginationState } from '@tanstack/react-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

type UseRunsProps = {
  rowSelection: RowSelectionState;
  pagination: PaginationState;
  createdAfter?: string;
  finishedBefore?: string;
  statuses?: V1TaskStatus[];
  additionalMetadata?: string[];
  workerId: string | undefined;
  workflowIds?: string[];
  parentTaskExternalId: string | undefined;
  triggeringEventExternalId?: string | undefined;
  onlyTasks: boolean;
  disablePagination?: boolean;
};

export const useRuns = ({
  rowSelection,
  pagination,
  createdAfter,
  finishedBefore,
  statuses,
  additionalMetadata,
  workerId,
  workflowIds,
  parentTaskExternalId,
  triggeringEventExternalId,
  onlyTasks,
  disablePagination = false,
}: UseRunsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { currentInterval } = useRefetchInterval();
  const offset = pagination.pageIndex * pagination.pageSize;

  const [initialRenderTime] = useState(
    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );
  const since = useMemo(
    () =>
      // hack - when a parentTaskExternalId is provided, we want to show all child tasks regardless
      // of how long ago they were run
      parentTaskExternalId
        ? new Date(Date.now() - 31 * 24 * 60 * 60 * 1000).toISOString()
        : createdAfter || initialRenderTime,
    [createdAfter, initialRenderTime, parentTaskExternalId],
  );

  const listTasksQuery = useQuery({
    ...queries.v1WorkflowRuns.list(tenantId, {
      offset: disablePagination ? 0 : offset,
      limit: disablePagination ? 500 : pagination.pageSize,
      statuses: statuses && statuses.length > 0 ? statuses : undefined,
      workflow_ids: workflowIds && workflowIds.length > 0 ? workflowIds : [],
      parent_task_external_id: parentTaskExternalId,
      since,
      until: finishedBefore,
      additional_metadata: additionalMetadata,
      worker_id: workerId,
      only_tasks: onlyTasks,
      triggering_event_external_id: triggeringEventExternalId,
    }),
    placeholderData: (prev) => prev,
    refetchInterval:
      Object.keys(rowSelection).length > 0 ? false : currentInterval,
  });

  const tasks = listTasksQuery.data;
  const tableRows = useMemo(() => {
    return tasks?.rows || [];
  }, [tasks]);

  const selectedRuns = useMemo(() => {
    return Object.entries(rowSelection)
      .filter(([, selected]) => !!selected)
      .map(([id]) => {
        const findRow = (rows: V1TaskSummary[]): V1TaskSummary | undefined => {
          for (const row of rows) {
            if (row.metadata.id === id) {
              return row;
            }
            if (row.children) {
              const childRow = findRow(row.children);
              if (childRow) {
                return childRow;
              }
            }
          }
          return undefined;
        };

        return findRow(tableRows);
      })
      .filter((row) => row !== undefined) as V1TaskSummary[];
  }, [rowSelection, tableRows]);

  const getRowId = useCallback((row: V1TaskSummary) => {
    return row.metadata.id;
  }, []);

  return {
    numPages: tasks?.pagination.num_pages || 0,
    tableRows,
    selectedRuns,
    refetch: listTasksQuery.refetch,
    isLoading: listTasksQuery.isLoading,
    isError: listTasksQuery.isError,
    isFetching: listTasksQuery.isFetching,
    getRowId,
    isRefetching: listTasksQuery.isRefetching,
  };
};
