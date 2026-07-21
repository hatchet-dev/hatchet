import {
  statusKey,
  workflowKey,
  additionalMetadataKey,
  additionalMetadataOperatorKey,
  flattenDAGsKey,
  createdAfterKey,
  finishedBeforeKey,
  isCustomTimeRangeKey,
  timeWindowKey,
  runningFilterKey,
  idempotencyKeyKey,
} from '../components/v1/task-runs-columns';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import {
  V1AdditionalMetadataOperator,
  V1RunningFilter,
  V1TaskStatus,
} from '@/lib/api';
import { useSearchParams } from '@/lib/router-helpers';
import { ColumnFiltersState } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { z } from 'zod';

type TimeWindow = '1h' | '6h' | '1d' | '7d';

const getCreatedAfterFromTimeRange = (timeWindow: TimeWindow): string => {
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

export type AdditionalMetadataProp = {
  key: string;
  value: string;
};

type APIFilters = {
  since: string;
  until?: string;
  statuses?: V1TaskStatus[];
  workflowIds?: string[];
  additionalMetadata?: string[];
  additionalMetadataOperator: V1AdditionalMetadataOperator;
  flattenDAGs: boolean;
  runningFilter?: V1RunningFilter;
  idempotencyKeys?: string[];
};

export type FilterActions = {
  timeWindow: TimeWindow;
  isCustomTimeRange: boolean;
  apiFilters: APIFilters;
  setTimeWindow: (timeWindow: TimeWindow) => void;
  setCustomTimeRange: (range: { start: string; end: string } | null) => void;
  updateCurrentTimeWindow: () => void;
  setStatuses: (statuses: V1TaskStatus[]) => void;
  setAdditionalMetadata: (metadata: AdditionalMetadataProp) => void;
  setIdempotencyKey: (idempotencyKey: string) => void;
  setAdditionalMetadataOperator: (
    operator: V1AdditionalMetadataOperator,
  ) => void;
  setColumnFilters: (filters: ColumnFiltersState) => void;
  resetFilters: () => void;
};

const createApiFilterSchema = (initialValues?: { workflowIds?: string[] }) =>
  z.object({
    tw: z.enum(['1h', '6h', '1d', '7d']).default('1d'), // time window preset
    ctr: z.boolean().default(false), // whether using custom range
    s: z.string().optional(), // since
    u: z.string().optional(), // until
    st: z
      .array(z.nativeEnum(V1TaskStatus))
      .default(() => Object.values(V1TaskStatus)), // statuses
    w: z // workflow ids
      .array(z.string())
      .optional()
      .default(() =>
        initialValues?.workflowIds?.length ? initialValues.workflowIds : [],
      ),
    m: z.array(z.string()).optional(), // additional metadata
    // metadata match operator. The frontend defaults to AND (all pairs must
    // match) even though the API defaults to OR, so the param is always sent.
    mo: z
      .nativeEnum(V1AdditionalMetadataOperator)
      .default(V1AdditionalMetadataOperator.AND),
    f: z.boolean().default(false), // flatten dags
    rf: z.nativeEnum(V1RunningFilter).optional(), // running sub-filter (undefined = ALL)
    i: z.array(z.string()).optional(), // idempotency keys
  });

export const useRunsTableFilters = (
  tableKey: string,
  initialValues?: {
    workflowIds?: string[];
  },
): FilterActions & {
  columnFilters: ColumnFiltersState;
  timeWindow: TimeWindow;
  isCustomTimeRange: boolean;
  apiFilters: APIFilters;
} => {
  const paramKey = tableKey + '-workflow-runs-filters';
  const apiFilterSchema = createApiFilterSchema(initialValues);
  const [, setSearchParams] = useSearchParams();

  const zodFiltersHook = useZodColumnFilters(apiFilterSchema, paramKey, {
    tw: timeWindowKey,
    ctr: isCustomTimeRangeKey,
    u: finishedBeforeKey,
    s: createdAfterKey,
    st: statusKey,
    w: workflowKey,
    m: additionalMetadataKey,
    mo: additionalMetadataOperatorKey,
    f: flattenDAGsKey,
    rf: runningFilterKey,
    i: idempotencyKeyKey,
  });

  const {
    state: zodState,
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = zodFiltersHook;

  const {
    tw: timeWindow,
    ctr: isCustomTimeRange,
    s: rawCreatedAfter,
    u: finishedBefore,
    st: selectedStatuses,
    w: selectedWorkflowIds,
    m: selectedAdditionalMetadata,
    mo: selectedAdditionalMetadataOperator,
    f: selectedFlattenDAGs,
    rf: selectedRunningFilter,
    i: selectedIdempotencyKeys,
  } = zodState;

  const createdAfter = useMemo(() => {
    if (rawCreatedAfter) {
      return rawCreatedAfter;
    }
    return getCreatedAfterFromTimeRange(timeWindow);
  }, [rawCreatedAfter, timeWindow]);

  const setZodState = useCallback(
    (newState: Partial<typeof zodState>) => {
      const updatedState = { ...zodState, ...newState };
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [paramKey]: updatedState,
      }));
    },
    [zodState, setSearchParams, paramKey],
  );

  const setTimeWindow = useCallback(
    (timeWindow: TimeWindow) => {
      setZodState({
        tw: timeWindow,
        ctr: false,
        s: getCreatedAfterFromTimeRange(timeWindow),
        u: undefined,
      });
    },
    [setZodState],
  );

  const updateCurrentTimeWindow = useCallback(() => {
    if (!isCustomTimeRange) {
      setZodState({
        ...zodState,
        s: getCreatedAfterFromTimeRange(timeWindow),
      });
    }
  }, [isCustomTimeRange, timeWindow, setZodState, zodState]);

  const setCustomTimeRange = useCallback(
    (range: { start: string; end: string } | null) => {
      if (range) {
        setZodState({
          ctr: true,
          s: range.start,
          u: range.end,
        });
      } else {
        setZodState({
          ctr: false,
          s: getCreatedAfterFromTimeRange(timeWindow),
          u: undefined,
        });
      }
    },
    [setZodState, timeWindow],
  );

  const setStatuses = useCallback(
    (statuses: V1TaskStatus[]) => {
      const finalStatuses =
        statuses.length > 0 ? statuses : Object.values(V1TaskStatus);

      const newColumnFilters = columnFilters
        .filter((f) => f.id !== statusKey)
        .concat([{ id: statusKey, value: finalStatuses }]);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters],
  );

  const setAdditionalMetadata = useCallback(
    ({ key, value }: { key: string; value: string }) => {
      const existing = selectedAdditionalMetadata || [];
      const filtered = existing.filter((m: string) => m.split(':')[0] !== key);
      const newMetadata = [...filtered, `${key}:${value}`];

      const newColumnFilters = columnFilters
        .filter((f) => f.id !== additionalMetadataKey)
        .concat([{ id: additionalMetadataKey, value: newMetadata }]);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters, selectedAdditionalMetadata],
  );

  const setIdempotencyKey = useCallback(
    (idempotencyKey: string) => {
      const existing = selectedIdempotencyKeys || [];
      const newKeys = existing.includes(idempotencyKey)
        ? existing.filter((k: string) => k !== idempotencyKey)
        : [...existing, idempotencyKey];

      const newColumnFilters = columnFilters
        .filter((f) => f.id !== idempotencyKeyKey)
        .concat(
          newKeys.length > 0 ? [{ id: idempotencyKeyKey, value: newKeys }] : [],
        );

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters, selectedIdempotencyKeys],
  );

  const setAdditionalMetadataOperator = useCallback(
    (operator: V1AdditionalMetadataOperator) => {
      const newColumnFilters = columnFilters
        .filter((f) => f.id !== additionalMetadataOperatorKey)
        .concat([{ id: additionalMetadataOperatorKey, value: operator }]);

      setColumnFilters(newColumnFilters);
    },
    [setColumnFilters, columnFilters],
  );

  const apiFilters = useMemo(
    () => ({
      since: createdAfter || getCreatedAfterFromTimeRange('1d'),
      until: finishedBefore,
      statuses: selectedStatuses,
      workflowIds: selectedWorkflowIds,
      additionalMetadata: selectedAdditionalMetadata,
      additionalMetadataOperator: selectedAdditionalMetadataOperator,
      flattenDAGs: selectedFlattenDAGs || false,
      runningFilter: selectedRunningFilter,
      idempotencyKeys: selectedIdempotencyKeys,
    }),
    [
      createdAfter,
      finishedBefore,
      selectedStatuses,
      selectedWorkflowIds,
      selectedAdditionalMetadata,
      selectedAdditionalMetadataOperator,
      selectedFlattenDAGs,
      selectedRunningFilter,
      selectedIdempotencyKeys,
    ],
  );

  return {
    columnFilters,
    timeWindow,
    isCustomTimeRange,
    apiFilters,
    setTimeWindow,
    setCustomTimeRange,
    updateCurrentTimeWindow,
    setStatuses,
    setAdditionalMetadata,
    setIdempotencyKey,
    setAdditionalMetadataOperator,
    setColumnFilters,
    resetFilters,
  };
};
