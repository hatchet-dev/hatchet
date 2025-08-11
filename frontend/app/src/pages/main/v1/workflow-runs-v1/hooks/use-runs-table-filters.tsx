import { useMemo, useCallback } from 'react';
import { V1TaskStatus } from '@/lib/api';
import { ColumnFiltersState } from '@tanstack/react-table';
import {
  RunsTableState,
  TimeWindow,
  getCreatedAfterFromTimeRange,
  getWorkflowIdFromFilters,
  getStatusFromFilters,
  getAdditionalMetadataFromFilters,
} from './use-runs-table-state';
import { TaskRunColumn } from '../components/v1/task-runs-columns';

export type AdditionalMetadataProp = {
  key: string;
  value: string;
};

type FilterActions = {
  setTimeWindow: (timeWindow: TimeWindow) => void;
  setCustomTimeRange: (range: { start: string; end: string } | null) => void;
  setStatus: (status: V1TaskStatus | undefined) => void;
  setWorkflowId: (workflowId: string | undefined) => void;
  setAdditionalMetadata: (metadata: AdditionalMetadataProp) => void;
  setAllAdditionalMetadata: (kvPairs: AdditionalMetadataProp[]) => void;
  setParentTaskExternalId: (id: string | undefined) => void;
  setColumnFilters: (filters: ColumnFiltersState) => void;
  clearAllFilters: () => void;
  clearParentFilter: () => void;
};

type APIFilters = {
  since?: string;
  until?: string;
  statuses?: V1TaskStatus[];
  workflowIds?: string[];
  additionalMetadata?: string[];
};

export const useRunsTableFilters = (
  state: RunsTableState,
  updateFilters: (filters: Partial<RunsTableState>) => void,
): FilterActions & { apiFilters: APIFilters } => {
  const apiFilters = useMemo((): APIFilters => {
    const status = getStatusFromFilters(state.columnFilters);
    const workflowId = getWorkflowIdFromFilters(state.columnFilters);
    const additionalMetadata = getAdditionalMetadataFromFilters(
      state.columnFilters,
    );

    return {
      since: state.createdAfter,
      until: state.finishedBefore,
      statuses: status ? [status] : undefined,
      workflowIds: workflowId ? [workflowId] : undefined,
      additionalMetadata,
    };
  }, [state.createdAfter, state.finishedBefore, state.columnFilters]);

  const setTimeWindow = useCallback(
    (timeWindow: TimeWindow) => {
      updateFilters({
        timeWindow,
        isCustomTimeRange: false,
        createdAfter: getCreatedAfterFromTimeRange(timeWindow),
        finishedBefore: undefined,
      });
    },
    [updateFilters],
  );

  const setCustomTimeRange = useCallback(
    (range: { start: string; end: string } | null) => {
      if (range) {
        updateFilters({
          isCustomTimeRange: true,
          createdAfter: range.start,
          finishedBefore: range.end,
        });
      } else {
        updateFilters({
          isCustomTimeRange: false,
          createdAfter: getCreatedAfterFromTimeRange(state.timeWindow),
          finishedBefore: undefined,
        });
      }
    },
    [updateFilters, state.timeWindow],
  );

  const setStatus = useCallback(
    (status: V1TaskStatus | undefined) => {
      const newColumnFilters = status
        ? state.columnFilters
            .filter((f) => f.id !== TaskRunColumn.status)
            .concat([{ id: TaskRunColumn.status, value: status }])
        : state.columnFilters.filter((f) => f.id !== TaskRunColumn.status);

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
  );

  const setWorkflowId = useCallback(
    (workflowId: string | undefined) => {
      const newColumnFilters = workflowId
        ? state.columnFilters
            .filter((f) => f.id !== TaskRunColumn.workflow)
            .concat([{ id: TaskRunColumn.workflow, value: workflowId }])
        : state.columnFilters.filter((f) => f.id !== TaskRunColumn.workflow);

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
  );

  const setAdditionalMetadata = useCallback(
    ({ key, value }: { key: string; value: string }) => {
      const existing =
        getAdditionalMetadataFromFilters(state.columnFilters) || [];
      const filtered = existing.filter((m: string) => m.split(':')[0] !== key);
      const newMetadata = [...filtered, `${key}:${value}`];

      const newColumnFilters = state.columnFilters
        .filter((f) => f.id !== TaskRunColumn.additionalMetadata)
        .concat([{ id: TaskRunColumn.additionalMetadata, value: newMetadata }]);

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
  );

  const setAllAdditionalMetadata = useCallback(
    (kvPairs: { key: string; value: string }[]) => {
      const newMetadata = kvPairs.map(({ key, value }) => `${key}:${value}`);

      const newColumnFilters =
        newMetadata.length > 0
          ? state.columnFilters
              .filter((f) => f.id !== TaskRunColumn.additionalMetadata)
              .concat([
                { id: TaskRunColumn.additionalMetadata, value: newMetadata },
              ])
          : state.columnFilters.filter(
              (f) => f.id !== TaskRunColumn.additionalMetadata,
            );

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
  );

  const setParentTaskExternalId = useCallback(
    (parentTaskExternalId: string | undefined) => {
      updateFilters({ parentTaskExternalId });
    },
    [updateFilters],
  );

  const setColumnFilters = useCallback(
    (columnFilters: ColumnFiltersState) => {
      updateFilters({
        columnFilters,
      });
    },
    [updateFilters],
  );

  const clearAllFilters = useCallback(() => {
    updateFilters({
      parentTaskExternalId: undefined,
      columnFilters: [],
    });
  }, [updateFilters]);

  const clearParentFilter = useCallback(() => {
    updateFilters({ parentTaskExternalId: undefined });
  }, [updateFilters]);

  return {
    apiFilters,
    setTimeWindow,
    setCustomTimeRange,
    setStatus,
    setWorkflowId,
    setAdditionalMetadata,
    setAllAdditionalMetadata,
    setParentTaskExternalId,
    setColumnFilters,
    clearAllFilters,
    clearParentFilter,
  };
};
