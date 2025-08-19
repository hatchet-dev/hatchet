import { useMemo, useCallback } from 'react';
import { V1TaskStatus } from '@/lib/api';
import { ColumnFiltersState } from '@tanstack/react-table';
import {
  RunsTableState,
  TimeWindow,
  getCreatedAfterFromTimeRange,
  getWorkflowIdsFromFilters,
  getStatusesFromFilters,
  getAdditionalMetadataFromFilters,
  getFlattenDAGsFromFilters,
} from './use-runs-table-state';
import {
  statusKey,
  workflowKey,
  additionalMetadataKey,
} from '../components/v1/task-runs-columns';

export type AdditionalMetadataProp = {
  key: string;
  value: string;
};

export type FilterActions = {
  setTimeWindow: (timeWindow: TimeWindow) => void;
  setCustomTimeRange: (range: { start: string; end: string } | null) => void;
  setStatuses: (statuses: V1TaskStatus[]) => void;
  setWorkflowIds: (workflowIds: string[]) => void;
  setAdditionalMetadata: (metadata: AdditionalMetadataProp) => void;
  setAllAdditionalMetadata: (kvPairs: AdditionalMetadataProp[]) => void;
  setParentTaskExternalId: (id: string | undefined) => void;
  setColumnFilters: (filters: ColumnFiltersState) => void;
  clearAllFilters: () => void;
  clearParentFilter: () => void;
};

export type APIFilters = {
  since?: string;
  until?: string;
  statuses?: V1TaskStatus[];
  workflowIds?: string[];
  additionalMetadata?: string[];
  flattenDAGs: boolean;
};

export const useRunsTableFilters = (
  state: RunsTableState,
  updateFilters: (filters: Partial<RunsTableState>) => void,
): FilterActions & { apiFilters: APIFilters } => {
  const apiFilters = useMemo((): APIFilters => {
    const statuses = getStatusesFromFilters(state.columnFilters);
    const workflowIds = getWorkflowIdsFromFilters(state.columnFilters);
    const additionalMetadata = getAdditionalMetadataFromFilters(
      state.columnFilters,
    );
    const flattenDAGs = getFlattenDAGsFromFilters(state.columnFilters);

    return {
      since: state.createdAfter,
      until: state.finishedBefore,
      statuses: statuses.length > 0 ? statuses : undefined,
      workflowIds: workflowIds.length > 0 ? workflowIds : undefined,
      additionalMetadata,
      flattenDAGs,
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

  const setStatuses = useCallback(
    (statuses: V1TaskStatus[]) => {
      const newColumnFilters =
        statuses.length > 0
          ? state.columnFilters
              .filter((f) => f.id !== statusKey)
              .concat([{ id: statusKey, value: statuses }])
          : state.columnFilters.filter((f) => f.id !== statusKey);

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
  );

  const setWorkflowIds = useCallback(
    (workflowIds: string[]) => {
      const newColumnFilters =
        workflowIds.length > 0
          ? state.columnFilters
              .filter((f) => f.id !== workflowKey)
              .concat([{ id: workflowKey, value: workflowIds }])
          : state.columnFilters.filter((f) => f.id !== workflowKey);

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
        .filter((f) => f.id !== additionalMetadataKey)
        .concat([{ id: additionalMetadataKey, value: newMetadata }]);

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
              .filter((f) => f.id !== additionalMetadataKey)
              .concat([{ id: additionalMetadataKey, value: newMetadata }])
          : state.columnFilters.filter((f) => f.id !== additionalMetadataKey);

      updateFilters({
        columnFilters: newColumnFilters,
      });
    },
    [updateFilters, state.columnFilters],
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

  const setParentTaskExternalId = useCallback(
    (parentTaskExternalId: string | undefined) => {
      updateFilters({ parentTaskExternalId });
    },
    [updateFilters],
  );

  return {
    apiFilters,
    setTimeWindow,
    setCustomTimeRange,
    setStatuses,
    setWorkflowIds,
    setAdditionalMetadata,
    setAllAdditionalMetadata,
    setParentTaskExternalId,
    setColumnFilters,
    clearAllFilters,
    clearParentFilter,
  };
};
