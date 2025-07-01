import { queries, V1TaskSummary } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useColumnFilters } from './column-filters';
import { usePagination } from './pagination';
import { useCallback, useMemo, useState } from 'react';
import { RowSelectionState } from '@tanstack/react-table';
import { useCurrentTenantId } from '@/hooks/use-tenant';

type UseTaskRunProps = {
  rowSelection: RowSelectionState;
  workerId: string | undefined;
  workflow: string | undefined;
  parentTaskExternalId: string | undefined;
  triggeringEventExternalId?: string | undefined;
  disablePagination?: boolean;
};

export const useTaskRuns = ({
  rowSelection,
  workerId,
  workflow,
  parentTaskExternalId,
  triggeringEventExternalId,
  disablePagination = false,
}: UseTaskRunProps) => {
  const cf = useColumnFilters();
  const { pagination, offset } = usePagination();
  const { tenantId } = useCurrentTenantId();

  const [initialRenderTime] = useState(
    new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
  );

  const listTasksQuery = useQuery({
    ...queries.v1WorkflowRuns.list(tenantId, {
      offset: disablePagination ? 0 : offset,
      limit: disablePagination ? 500 : pagination.pageSize,
      statuses: cf.filters.status ? [cf.filters.status] : undefined,
      workflow_ids: workflow ? [workflow] : [],
      parent_task_external_id: parentTaskExternalId,
      since: cf.filters.createdAfter || initialRenderTime,
      until: cf.filters.finishedBefore,
      additional_metadata: cf.filters.additionalMetadata,
      worker_id: workerId,
      only_tasks: !!workerId,
      triggering_event_external_id: triggeringEventExternalId,
    }),
    placeholderData: (prev) => prev,
    refetchInterval: () => {
      if (Object.keys(rowSelection).length > 0) {
        return false;
      }

      return 5000;
    },
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
