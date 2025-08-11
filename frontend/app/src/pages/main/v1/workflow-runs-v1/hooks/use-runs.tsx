import { queries, V1TaskSummary, V1TaskStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';
import { RowSelectionState, PaginationState } from '@tanstack/react-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';

type UseRunsProps = {
  rowSelection: RowSelectionState;
  pagination: PaginationState;
  createdAfter?: string;
  finishedBefore?: string;
  status?: V1TaskStatus;
  additionalMetadata?: string[];
  workerId: string | undefined;
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  triggeringEventExternalId?: string | undefined;
  disablePagination?: boolean;
  pauseRefetch?: boolean;
};

export const useRuns = ({
  rowSelection,
  pagination,
  createdAfter,
  finishedBefore,
  status,
  additionalMetadata,
  workerId,
  workflow,
  parentTaskExternalId,
  triggeringEventExternalId,
  disablePagination = false,
  pauseRefetch = false,
}: UseRunsProps) => {
  const { tenantId } = useCurrentTenantId();
  const offset = pagination.pageIndex * pagination.pageSize;

  const [initialRenderTime] = useState(
    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );

  const listTasksQuery = useQuery({
    ...queries.v1WorkflowRuns.list(tenantId, {
      offset: disablePagination ? 0 : offset,
      limit: disablePagination ? 500 : pagination.pageSize,
      statuses: status ? [status] : undefined,
      workflow_ids: workflow ? [workflow] : [],
      parent_task_external_id: parentTaskExternalId,
      since: createdAfter || initialRenderTime,
      until: finishedBefore,
      additional_metadata: additionalMetadata,
      worker_id: workerId,
      only_tasks: !!workerId,
      triggering_event_external_id: triggeringEventExternalId,
    }),
    placeholderData: (prev) => prev,
    refetchInterval:
      Object.keys(rowSelection).length > 0 || pauseRefetch ? false : 5000,
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
  };
};
