import { useCallback, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { V1TaskStatus } from '@/lib/api';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  VisibilityState,
} from '@tanstack/react-table';

export type TimeWindow = '1h' | '6h' | '1d' | '7d';

type TaskRunDetailSheetState =
  | {
      isOpen: true;
      taskRunId: string;
    }
  | {
      isOpen: false;
      taskRunId?: never;
    };

export interface BaseRunsTableState {
  // Pagination
  pagination: PaginationState;

  // Filters
  timeWindow: TimeWindow;
  isCustomTimeRange: boolean;
  createdAfter?: string;
  finishedBefore?: string;
  parentTaskExternalId?: string;

  // Table state / visibility
  columnFilters: ColumnFiltersState;
  rowSelection: RowSelectionState;
  columnVisibility: VisibilityState;

  // UI state
  selectedAdditionalMetaRunId?: string;
  viewQueueMetrics: boolean;
  triggerWorkflow: boolean;
  taskRunDetailSheet: TaskRunDetailSheetState;
}

export interface RunsTableState extends BaseRunsTableState {
  hasFiltersApplied: boolean;
  hasRowsSelected: boolean;
  hasOpenUI: boolean;
}

const DEFAULT_STATE: RunsTableState = {
  pagination: { pageIndex: 0, pageSize: 50 },
  timeWindow: '1d',
  isCustomTimeRange: false,
  columnFilters: [],
  rowSelection: {},
  columnVisibility: {},
  viewQueueMetrics: false,
  triggerWorkflow: false,
  taskRunDetailSheet: { isOpen: false },
  hasFiltersApplied: false,
  hasRowsSelected: false,
  hasOpenUI: false,
};

// Mapping keys to abbreviations for URL storage
// so we don't accidentally overflow browser URL length limits
// It's not bulletproof, but it should help a lot
const KEY_MAP = {
  // Pagination
  pagination: 'p',
  pageIndex: 'i',
  pageSize: 's',

  // Time filters
  timeWindow: 't',
  isCustomTimeRange: 'c',
  createdAfter: 'ca',
  finishedBefore: 'fb',
  parentTaskExternalId: 'pt',

  // Table state
  columnFilters: 'cf',
  rowSelection: 'rs',
  columnVisibility: 'cv',

  // UI state
  selectedAdditionalMetaRunId: 'sm',
  viewQueueMetrics: 'vq',
  triggerWorkflow: 'tw',
  stepDetailSheet: 'sd',

  // Nested properties
  isOpen: 'o',
  taskRunId: 'tr',
  id: 'id',
  value: 'v',
} as const;

const REVERSE_KEY_MAP = Object.fromEntries(
  Object.entries(KEY_MAP).map(([key, value]) => [value, key]),
) as Record<string, string>;

function compressKeys(obj: any): any {
  if (obj === null || obj === undefined) {
    return obj;
  }
  if (typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(compressKeys);
  }

  const compressed: any = {};
  for (const [key, value] of Object.entries(obj)) {
    const compressedKey = KEY_MAP[key as keyof typeof KEY_MAP] || key;
    compressed[compressedKey] = compressKeys(value);
  }
  return compressed;
}

function decompressKeys(obj: any): any {
  if (obj === null || obj === undefined) {
    return obj;
  }
  if (typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(decompressKeys);
  }

  const decompressed: any = {};
  for (const [key, value] of Object.entries(obj)) {
    const decompressedKey = REVERSE_KEY_MAP[key] || key;
    decompressed[decompressedKey] = decompressKeys(value);
  }
  return decompressed;
}

export const getCreatedAfterFromTimeRange = (
  timeWindow: TimeWindow,
): string => {
  switch (timeWindow) {
    case '1h':
      return new Date(Date.now() - 60 * 60 * 1000).toISOString();
    case '6h':
      return new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString();
    case '1d':
      return new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    case '7d':
      return new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
    default: {
      const exhaustiveCheck: never = timeWindow;
      throw new Error(`Unhandled time range: ${exhaustiveCheck}`);
    }
  }
};

export const getWorkflowIdFromFilters = (
  columnFilters: ColumnFiltersState,
): string | undefined => {
  const filter = columnFilters.find((f) => f.id === 'Workflow');
  if (!filter) {
    return undefined;
  }
  const value = filter.value;
  return Array.isArray(value) ? (value[0] as string) : (value as string);
};

export const getWorkflowIdsFromFilters = (
  columnFilters: ColumnFiltersState,
): string[] => {
  const filter = columnFilters.find((f) => f.id === 'Workflow');
  if (!filter) {
    return [];
  }
  const value = filter.value;
  return Array.isArray(value) ? (value as string[]) : [value as string];
};

export const getStatusFromFilters = (
  columnFilters: ColumnFiltersState,
): V1TaskStatus | undefined => {
  const filter = columnFilters.find((f) => f.id === 'status');
  if (!filter) {
    return undefined;
  }
  const value = filter.value;
  return Array.isArray(value)
    ? (value[0] as V1TaskStatus)
    : (value as V1TaskStatus);
};

export const getStatusesFromFilters = (
  columnFilters: ColumnFiltersState,
): V1TaskStatus[] => {
  const filter = columnFilters.find((f) => f.id === 'status');
  if (!filter) {
    return [];
  }
  const value = filter.value;
  return Array.isArray(value) ? (value as V1TaskStatus[]) : [value as V1TaskStatus];
};

export const getAdditionalMetadataFromFilters = (
  columnFilters: ColumnFiltersState,
): string[] | undefined => {
  const filter = columnFilters.find((f) => f.id === 'additionalMetadata');
  if (!filter) {
    return undefined;
  }
  const value = filter.value;
  return Array.isArray(value) ? (value as string[]) : [value as string];
};

export const useRunsTableState = (
  tableKey: string,
  initialState?: Partial<RunsTableState>,
) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const paramKey = `table_${tableKey}`;

  const initialStateRef = useRef(initialState);
  initialStateRef.current = initialState;

  const currentState = useMemo((): RunsTableState => {
    const stateParam = searchParams.get(paramKey);

    if (!stateParam) {
      const merged = { ...DEFAULT_STATE, ...initialStateRef.current };
      if (!merged.isCustomTimeRange) {
        merged.createdAfter = getCreatedAfterFromTimeRange(merged.timeWindow);
      }
      return merged;
    }

    try {
      const compressedState = JSON.parse(stateParam);
      const parsedState = decompressKeys(compressedState) as RunsTableState;
      const merged = {
        ...DEFAULT_STATE,
        ...parsedState,
        ...initialStateRef.current,
      };

      if (!merged.isCustomTimeRange) {
        merged.createdAfter = getCreatedAfterFromTimeRange(merged.timeWindow);
      }

      return merged;
    } catch (error) {
      console.warn('Failed to parse table state from URL:', error);
      const merged = { ...DEFAULT_STATE, ...initialStateRef.current };
      if (!merged.isCustomTimeRange) {
        merged.createdAfter = getCreatedAfterFromTimeRange(merged.timeWindow);
      }
      return merged;
    }
  }, [searchParams, paramKey]);

  const updateState = useCallback(
    (updates: Partial<RunsTableState>) => {
      setSearchParams(
        (prev) => {
          const newParams = new URLSearchParams(prev);
          const stateParam = newParams.get(paramKey);

          let currentStateFromURL: RunsTableState;
          if (!stateParam) {
            const merged = { ...DEFAULT_STATE, ...initialStateRef.current };
            if (!merged.isCustomTimeRange) {
              merged.createdAfter = getCreatedAfterFromTimeRange(
                merged.timeWindow,
              );
            }
            currentStateFromURL = merged;
          } else {
            try {
              const compressedState = JSON.parse(stateParam);
              const parsedState = decompressKeys(
                compressedState,
              ) as RunsTableState;
              const merged = {
                ...DEFAULT_STATE,
                ...parsedState,
                ...initialStateRef.current,
              };
              if (!merged.isCustomTimeRange) {
                merged.createdAfter = getCreatedAfterFromTimeRange(
                  merged.timeWindow,
                );
              }
              currentStateFromURL = merged;
            } catch (error) {
              const merged = { ...DEFAULT_STATE, ...initialStateRef.current };
              if (!merged.isCustomTimeRange) {
                merged.createdAfter = getCreatedAfterFromTimeRange(
                  merged.timeWindow,
                );
              }
              currentStateFromURL = merged;
            }
          }

          const newState = { ...currentStateFromURL, ...updates };

          if (updates.timeWindow && !newState.isCustomTimeRange) {
            newState.createdAfter = getCreatedAfterFromTimeRange(
              newState.timeWindow,
            );
            newState.finishedBefore = undefined;
          }

          if (
            Object.keys(updates).some(
              (key) =>
                key !== 'pagination' &&
                key !== 'rowSelection' &&
                key !== 'columnVisibility',
            )
          ) {
            newState.pagination = { ...newState.pagination, pageIndex: 0 };
          }

          const stateToSerialize: BaseRunsTableState = {
            pagination: newState.pagination,
            timeWindow: newState.timeWindow,
            isCustomTimeRange: newState.isCustomTimeRange,
            createdAfter: newState.createdAfter,
            finishedBefore: newState.finishedBefore,
            parentTaskExternalId: newState.parentTaskExternalId,
            columnFilters: newState.columnFilters,
            rowSelection: newState.rowSelection,
            columnVisibility: newState.columnVisibility,
            selectedAdditionalMetaRunId: newState.selectedAdditionalMetaRunId,
            viewQueueMetrics: newState.viewQueueMetrics,
            triggerWorkflow: newState.triggerWorkflow,
            taskRunDetailSheet: newState.taskRunDetailSheet,
          };

          const compressedState = compressKeys(stateToSerialize);
          const minifiedState = JSON.stringify(compressedState);

          newParams.set(paramKey, minifiedState);
          return newParams;
        },
        { replace: true },
      );
    },
    [paramKey, setSearchParams],
  );

  const updatePagination = useCallback(
    (pagination: PaginationState) => {
      updateState({ pagination });
    },
    [updateState],
  );

  const updateFilters = useCallback(
    (
      filters: Partial<
        Pick<
          RunsTableState,
          | 'timeWindow'
          | 'isCustomTimeRange'
          | 'createdAfter'
          | 'finishedBefore'
          | 'parentTaskExternalId'
          | 'columnFilters'
        >
      >,
    ) => {
      updateState(filters);
    },
    [updateState],
  );

  const updateUIState = useCallback(
    (
      uiState: Partial<
        Pick<
          RunsTableState,
          | 'selectedAdditionalMetaRunId'
          | 'viewQueueMetrics'
          | 'triggerWorkflow'
          | 'taskRunDetailSheet'
        >
      >,
    ) => {
      updateState(uiState);
    },
    [updateState],
  );

  const updateTableState = useCallback(
    (
      tableState: Partial<
        Pick<RunsTableState, 'rowSelection' | 'columnVisibility'>
      >,
    ) => {
      updateState(tableState);
    },
    [updateState],
  );

  const resetState = useCallback(() => {
    setSearchParams(
      (prev) => {
        const newParams = new URLSearchParams(prev);
        newParams.delete(paramKey);
        return newParams;
      },
      { replace: true },
    );
  }, [paramKey, setSearchParams]);

  const derivedState = useMemo(() => {
    const status = getStatusFromFilters(currentState.columnFilters);
    const additionalMetadata = getAdditionalMetadataFromFilters(
      currentState.columnFilters,
    );
    const workflowId = getWorkflowIdFromFilters(currentState.columnFilters);

    return {
      ...currentState,
      hasRowsSelected: Object.values(currentState.rowSelection).some(
        (selected) => !!selected,
      ),
      hasFiltersApplied: !!(
        status ||
        additionalMetadata?.length ||
        workflowId ||
        currentState.parentTaskExternalId
      ),
      hasOpenUI: !!(
        currentState.selectedAdditionalMetaRunId ||
        currentState.taskRunDetailSheet.isOpen ||
        currentState.viewQueueMetrics ||
        currentState.triggerWorkflow
      ),
    };
  }, [currentState]);

  return {
    state: derivedState,
    updateState,
    updatePagination,
    updateFilters,
    updateUIState,
    updateTableState,
    resetState,
  };
};
