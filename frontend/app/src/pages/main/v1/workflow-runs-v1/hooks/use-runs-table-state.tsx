import { useCallback, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
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
  parentTaskExternalId?: string;

  // Table state / visibility
  columnFilters: ColumnFiltersState;
  rowSelection: RowSelectionState;
  columnVisibility: VisibilityState;

  // UI state
  viewQueueMetrics: boolean;
  triggerWorkflow: boolean;
  taskRunDetailSheet: TaskRunDetailSheetState;
}

export interface RunsTableState extends BaseRunsTableState {
  hasFiltersApplied: boolean;
  hasRowsSelected: boolean;
  selectedAdditionalMetaRunId?: string;
}

const DEFAULT_STATE: RunsTableState = {
  pagination: { pageIndex: 0, pageSize: 50 },
  columnFilters: [],
  rowSelection: {},
  columnVisibility: {},
  viewQueueMetrics: false,
  triggerWorkflow: false,
  taskRunDetailSheet: { isOpen: false },
  hasFiltersApplied: false,
  hasRowsSelected: false,
};

// Mapping keys to abbreviations for URL storage
// so we don't accidentally overflow browser URL length limits
// It's not bulletproof, but it should help a lot
const KEY_MAP = {
  // Pagination
  pagination: 'p',
  pageIndex: 'i',
  pageSize: 's',

  // Column filters
  parentTaskExternalId: 'pt',

  // Table state
  rowSelection: 'rs',
  columnVisibility: 'cv',

  // UI state
  viewQueueMetrics: 'vq',
  triggerWorkflow: 'tw',
  taskRunDetailSheet: 'td',

  // Nested properties
  isOpen: 'o',
  taskRunId: 'tr',
  id: 'id',
  value: 'v',
} as const;

const REVERSE_KEY_MAP = Object.fromEntries(
  Object.entries(KEY_MAP).map(([key, value]) => [value, key]),
) as Record<string, string>;

interface ColumnFilterWithId {
  id: string;
  value: unknown;
  [key: string]: unknown;
}

type CompressibleValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | Array<CompressibleValue>
  | { [key: string]: CompressibleValue };

const parseColumnFilters = (obj: CompressibleValue[]) => {
  return obj.map((filter) => {
    if (
      filter &&
      typeof filter === 'object' &&
      !Array.isArray(filter) &&
      'id' in filter
    ) {
      const columnFilter = filter as ColumnFilterWithId;
      const compressedFilter = { ...columnFilter };
      const compressedId =
        KEY_MAP[columnFilter.id as keyof typeof KEY_MAP] || columnFilter.id;
      compressedFilter.id = compressedId;

      return compressKeys(
        compressedFilter as CompressibleValue,
        'columnFilter',
      );
    }
    return compressKeys(filter);
  });
};

function compressKeys(
  obj: CompressibleValue,
  parentKey?: string,
): CompressibleValue {
  if (obj === null || obj === undefined) {
    return obj;
  }
  if (typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    if (parentKey === 'columnFilters') {
      return parseColumnFilters(obj);
    }

    return obj.map((item) => compressKeys(item));
  }

  const compressed: Record<string, CompressibleValue> = {};
  for (const [key, value] of Object.entries(obj)) {
    const compressedKey = KEY_MAP[key as keyof typeof KEY_MAP] || key;
    compressed[compressedKey] = compressKeys(value, key);
  }
  return compressed;
}

const decompressColumnFilters = (obj: CompressibleValue[]) => {
  return obj.map((filter) => {
    if (
      filter &&
      typeof filter === 'object' &&
      !Array.isArray(filter) &&
      'id' in filter
    ) {
      const columnFilter = filter as ColumnFilterWithId;
      const decompressedFilter = { ...columnFilter };
      const decompressedId =
        REVERSE_KEY_MAP[columnFilter.id] || columnFilter.id;
      decompressedFilter.id = decompressedId;
      return decompressKeys(
        decompressedFilter as CompressibleValue,
        'columnFilter',
      );
    }
    return decompressKeys(filter);
  });
};

function decompressKeys(
  obj: CompressibleValue,
  parentKey?: string,
): CompressibleValue {
  if (obj === null || obj === undefined) {
    return obj;
  }
  if (typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    if (parentKey === 'cf') {
      return decompressColumnFilters(obj);
    }
    return obj.map((item) => decompressKeys(item));
  }

  const decompressed: Record<string, CompressibleValue> = {};
  for (const [key, value] of Object.entries(obj)) {
    const decompressedKey = REVERSE_KEY_MAP[key] || key;
    decompressed[decompressedKey] = decompressKeys(value, key);
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
      return merged;
    }

    try {
      const compressedState = JSON.parse(stateParam);
      const parsedState = decompressKeys(
        compressedState,
      ) as unknown as RunsTableState;
      const merged = {
        ...DEFAULT_STATE,
        ...parsedState,
        ...initialStateRef.current,
        columnFilters: parsedState.columnFilters || [],
        columnVisibility: {
          ...parsedState.columnVisibility,
          ...initialStateRef.current?.columnVisibility,
        },
      };

      return merged;
    } catch (error) {
      console.warn('Failed to parse table state from URL:', error);
      const merged = { ...DEFAULT_STATE, ...initialStateRef.current };
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
            const merged = {
              ...DEFAULT_STATE,
              ...initialStateRef.current,
              columnVisibility: {
                ...DEFAULT_STATE.columnVisibility,
                ...initialStateRef.current?.columnVisibility,
              },
            };
            currentStateFromURL = merged;
          } else {
            try {
              const compressedState = JSON.parse(stateParam);
              const parsedState = decompressKeys(
                compressedState,
              ) as unknown as RunsTableState;
              const merged = {
                ...DEFAULT_STATE,
                ...parsedState,
                ...initialStateRef.current,
                columnFilters: parsedState.columnFilters || [],
                columnVisibility: {
                  ...parsedState.columnVisibility,
                  ...initialStateRef.current?.columnVisibility,
                },
              };
              currentStateFromURL = merged;
            } catch (error) {
              const merged = {
                ...DEFAULT_STATE,
                ...initialStateRef.current,
                columnVisibility: {
                  ...DEFAULT_STATE.columnVisibility,
                  ...initialStateRef.current?.columnVisibility,
                },
              };
              currentStateFromURL = merged;
            }
          }

          const newState = { ...currentStateFromURL, ...updates };

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
            parentTaskExternalId: newState.parentTaskExternalId,
            columnFilters: newState.columnFilters,
            rowSelection: newState.rowSelection,
            columnVisibility: newState.columnVisibility,
            viewQueueMetrics: newState.viewQueueMetrics,
            triggerWorkflow: newState.triggerWorkflow,
            taskRunDetailSheet: newState.taskRunDetailSheet,
          };

          const compressedState = compressKeys(
            stateToSerialize as unknown as CompressibleValue,
          );
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
        Pick<RunsTableState, 'parentTaskExternalId' | 'columnFilters'>
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
          'viewQueueMetrics' | 'triggerWorkflow' | 'taskRunDetailSheet'
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

  const derivedState = useMemo(() => {
    return {
      ...currentState,
      hasRowsSelected: Object.values(currentState.rowSelection).some(
        (selected) => !!selected,
      ),
      hasFiltersApplied: !!currentState.parentTaskExternalId,
      hasOpenUI: !!(
        currentState.taskRunDetailSheet.isOpen ||
        currentState.viewQueueMetrics ||
        currentState.triggerWorkflow ||
        currentState.selectedAdditionalMetaRunId
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
  };
};
