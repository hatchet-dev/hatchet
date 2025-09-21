import { useCallback } from 'react';
import { V1TaskStatus } from '@/lib/api';
import { ColumnFiltersState } from '@tanstack/react-table';
import {
  RunsTableState,
  TimeWindow,
  getCreatedAfterFromTimeRange,
  getAdditionalMetadataFromFilters,
} from './use-runs-table-state';
import {
  statusKey,
  workflowKey,
  additionalMetadataKey,
  flattenDAGsKey,
  createdAfterKey,
  finishedBeforeKey,
} from '../components/v1/task-runs-columns';
import { z } from 'zod';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';

export type AdditionalMetadataProp = {
  key: string;
  value: string;
};

export type APIFilters = {
  since: string;
  until?: string;
  statuses?: V1TaskStatus[];
  workflowIds?: string[];
  additionalMetadata?: string[];
  flattenDAGs: boolean;
};

export type FilterActions = {
  setTimeWindow: (timeWindow: TimeWindow) => void;
  setCustomTimeRange: (range: { start: string; end: string } | null) => void;
  setStatuses: (statuses: V1TaskStatus[]) => void;
  setWorkflowIds: (workflowIds: string[]) => void;
  setAdditionalMetadata: (metadata: AdditionalMetadataProp) => void;
  setParentTaskExternalId: (id: string | undefined) => void;
  setColumnFilters: (filters: ColumnFiltersState) => void;
  clearAllFilters: () => void;
  clearParentFilter: () => void;
  resetFilters: () => void;
};

const apiFilterSchema = z.object({
  s: z.string().default(() => getCreatedAfterFromTimeRange('1d')), // since
  u: z.string().optional(), // until
  st: z.array(z.nativeEnum(V1TaskStatus)).optional(), // statuses
  w: z.array(z.string()).optional(), // workflow ids
  m: z.array(z.string()).optional(), // additional metadata
  f: z.boolean().default(false), // flatten dags
});

export const useRunsTableFilters = (
  state: RunsTableState,
  updateFilters: (filters: Partial<RunsTableState>) => void,
): FilterActions & {
  columnFilters: ColumnFiltersState;
  apiFilters: APIFilters;
} => {
  const paramKey = 'workflow-runs-filters';

  const {
    state: {
      s: createdAfter,
      u: finishedBefore,
      st: selectedStatuses,
      w: selectedWorkflowIds,
      m: selectedAdditionalMetadata,
      f: selectedFlattenDAGs,
    },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(apiFilterSchema, paramKey, {
    u: finishedBeforeKey,
    s: createdAfterKey,
    st: statusKey,
    w: workflowKey,
    m: additionalMetadataKey,
    f: flattenDAGsKey,
  });

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
          ? columnFilters
              .filter((f) => f.id !== statusKey)
              .concat([{ id: statusKey, value: statuses }])
          : columnFilters.filter((f) => f.id !== statusKey);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters],
  );

  const setWorkflowIds = useCallback(
    (workflowIds: string[]) => {
      const newColumnFilters =
        workflowIds.length > 0
          ? columnFilters
              .filter((f) => f.id !== workflowKey)
              .concat([{ id: workflowKey, value: workflowIds }])
          : columnFilters.filter((f) => f.id !== workflowKey);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters],
  );

  const setAdditionalMetadata = useCallback(
    ({ key, value }: { key: string; value: string }) => {
      const existing = getAdditionalMetadataFromFilters(columnFilters) || [];
      const filtered = existing.filter((m: string) => m.split(':')[0] !== key);
      const newMetadata = [...filtered, `${key}:${value}`];

      const newColumnFilters = columnFilters
        .filter((f) => f.id !== additionalMetadataKey)
        .concat([{ id: additionalMetadataKey, value: newMetadata }]);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters],
  );

  const clearAllFilters = useCallback(() => {
    setColumnFilters([]);
    updateFilters({
      parentTaskExternalId: undefined,
    });
  }, [setColumnFilters, updateFilters]);

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
    columnFilters,
    apiFilters: {
      since: createdAfter,
      until: finishedBefore,
      statuses: selectedStatuses,
      workflowIds: selectedWorkflowIds,
      additionalMetadata: selectedAdditionalMetadata,
      flattenDAGs: selectedFlattenDAGs || false,
    },
    setTimeWindow,
    setCustomTimeRange,
    setStatuses,
    setWorkflowIds,
    setAdditionalMetadata,
    setParentTaskExternalId,
    setColumnFilters,
    clearAllFilters,
    clearParentFilter,
    resetFilters,
  };
};
