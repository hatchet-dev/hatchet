import { V1TaskStatus } from '@/lib/api';
import {
  ColumnFilter,
  ColumnFiltersState,
  Updater,
} from '@tanstack/react-table';
import { useCallback, useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { TaskRunColumn } from '../components/v1/task-runs-columns';
import { usePagination } from './pagination';

export type TimeWindow = '1h' | '6h' | '1d' | '7d';

type FilterParams = {
  createdAfter: string;
  finishedBefore: string | undefined;
  isCustomTimeRange: boolean | undefined;
  status: V1TaskStatus | undefined;
  additionalMetadata: string[] | undefined;
  columnFilters: ColumnFiltersState;
  Workflow: string | undefined;
  timeWindow: TimeWindow | undefined;
  parentTaskExternalId: string | undefined;
};
type FilterKey = keyof FilterParams;

type TimeRange = {
  start: string;
  end: string;
};

type KVPair = {
  key: FilterKey;
  value: any;
};

type ColumnFilterKey = 'status' | 'additionalMetadata';

export const getCreatedAfterFromTimeRange = (timeWindow: TimeWindow) => {
  switch (timeWindow) {
    case '1h':
      return new Date(Date.now() - 60 * 60 * 1000).toISOString();
    case '6h':
      return new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString();
    case '1d':
      return new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    case '7d':
      return new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = timeWindow;
      throw new Error(`Unhandled time range: ${exhaustiveCheck}`);
  }
};

const queryParamNames = {
  createdAfter: 'createdAfter',
  finishedBefore: 'finishedBefore',
  isCustomTimeRange: 'isCustomTimeRange',
  status: TaskRunColumn.status,
  additionalMetadata: TaskRunColumn.additionalMetadata,
  workflow: TaskRunColumn.workflow,
  parentTaskExternalId: TaskRunColumn.parentTaskExternalId,
  timeWindow: 'timeWindow',
};

const parseTimeRange = ({
  isCustom,
  timeWindow,
  createdAfter,
  finishedBefore,
}: {
  isCustom: boolean;
  timeWindow: TimeWindow;
  createdAfter: string | null;
  finishedBefore: string | null;
}) => {
  if (isCustom && createdAfter && finishedBefore) {
    return {
      createdAfter,
      finishedBefore,
    };
  } else {
    return {
      createdAfter: getCreatedAfterFromTimeRange(timeWindow),
      finishedBefore: undefined,
    };
  }
};

export const useColumnFilters = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { pageIndexParamName } = usePagination();

  const timeWindowFilter = (searchParams.get(queryParamNames.timeWindow) ||
    '1d') as TimeWindow;

  const isCustomTimeRange =
    searchParams.get(queryParamNames.isCustomTimeRange) === 'true' || false;

  const { createdAfter, finishedBefore } = useMemo(() => {
    return parseTimeRange({
      isCustom: isCustomTimeRange,
      timeWindow: timeWindowFilter,
      createdAfter: searchParams.get(queryParamNames.createdAfter),
      finishedBefore: searchParams.get(queryParamNames.finishedBefore),
    });
  }, [isCustomTimeRange, timeWindowFilter, searchParams]);

  const status = searchParams.get(queryParamNames.status) as
    | V1TaskStatus
    | undefined;
  const additionalMetadataRaw = (searchParams.get(
    queryParamNames.additionalMetadata,
  ) || undefined) as string | undefined;

  const additionalMetadata = additionalMetadataRaw
    ? additionalMetadataRaw.split(',')
    : undefined;

  const workflowId = searchParams.get(queryParamNames.workflow) || undefined;

  const parentTaskExternalId =
    searchParams.get(queryParamNames.parentTaskExternalId) || undefined;

  const statusColumnFilter = status
    ? { id: TaskRunColumn.status, value: status }
    : undefined;
  const additionalMetadataColumnFilter = additionalMetadata
    ? { id: TaskRunColumn.additionalMetadata, value: additionalMetadata }
    : undefined;
  const workflowIdColumnFilter = workflowId
    ? { id: TaskRunColumn.workflow, value: workflowId }
    : undefined;

  const columnFilters: ColumnFiltersState = [
    statusColumnFilter,
    additionalMetadataColumnFilter,
    workflowIdColumnFilter,
  ].filter((f) => f !== undefined) as ColumnFiltersState;

  const setFilterValues = useCallback(
    (items: KVPair[]) => {
      setSearchParams((prev) => {
        const newParams = new URLSearchParams(prev);

        items.forEach(({ key, value }) => {
          if (value === undefined) {
            newParams.delete(key);
          } else {
            newParams.set(key, value);
          }
        });

        newParams.forEach((v, k) => {
          if (!v) {
            newParams.delete(k);
          }
        });

        newParams.set(pageIndexParamName, '0');

        return newParams;
      });
    },
    [setSearchParams, pageIndexParamName],
  );

  // create a timer which updates the defaultTimeWindowStartAt date every minute
  useEffect(() => {
    const interval = setInterval(() => {
      if (isCustomTimeRange) {
        return;
      }

      setFilterValues([
        {
          key: 'createdAfter',
          value:
            getCreatedAfterFromTimeRange(timeWindowFilter) ||
            new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
        },
      ]);
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [isCustomTimeRange, timeWindowFilter, setFilterValues]);

  const parseColumnFilter = (f: ColumnFilter): KVPair => {
    switch (f.id) {
      case 'status':
        return { key: 'status', value: f.value };
      case 'additionalMetadata':
        return { key: 'additionalMetadata', value: f.value };
      default:
        return { key: f.id as ColumnFilterKey, value: f.value };
    }
  };

  const setColumnFilters = useCallback(
    (updaterOrValue: Updater<ColumnFiltersState>) => {
      if (typeof updaterOrValue === 'function') {
        const newVal = updaterOrValue(columnFilters);

        const newFilters = newVal.map(parseColumnFilter);

        setFilterValues(newFilters);
      } else {
        const newFilters = updaterOrValue.map((f) => ({
          key: f.id as ColumnFilterKey,
          value: f.value,
        }));

        setFilterValues(newFilters);
      }
    },
    [columnFilters, setFilterValues],
  );

  const setCustomTimeRange = useCallback(
    (timeRange: TimeRange | undefined) => {
      setFilterValues([
        { key: 'isCustomTimeRange', value: timeRange ? 'true' : undefined },
        { key: 'createdAfter', value: timeRange ? timeRange.start : undefined },
        { key: 'finishedBefore', value: timeRange ? timeRange.end : undefined },
      ]);
    },
    [setFilterValues],
  );

  const setParentTaskExternalId = useCallback(
    (parentTaskExternalId: string | undefined) => {
      setFilterValues([
        {
          key: TaskRunColumn.parentTaskExternalId,
          value: parentTaskExternalId,
        },
      ]);
    },
    [setFilterValues],
  );

  const setCreatedAfter = useCallback(
    (createdAfter: string | undefined) => {
      setFilterValues([{ key: 'createdAfter', value: createdAfter }]);
    },
    [setFilterValues],
  );

  const setFinishedBefore = useCallback(
    (finishedBefore: string | undefined) => {
      setFilterValues([{ key: 'finishedBefore', value: finishedBefore }]);
    },
    [setFilterValues],
  );

  const setStatus = useCallback(
    (status: V1TaskStatus | undefined) => {
      setFilterValues([{ key: TaskRunColumn.status, value: status }]);
    },
    [setFilterValues],
  );

  const setWorkflowId = useCallback(
    (workflowId: string | undefined) => {
      setFilterValues([{ key: TaskRunColumn.workflow, value: workflowId }]);
    },
    [setFilterValues],
  );

  const setAdditionalMetadata = useCallback(
    ({ key, value }: { key: string; value: string }) => {
      const existing = additionalMetadata || [];
      const newMetadata = existing.filter((m) => m.split(':')[0] !== key);

      newMetadata.push(`${key}:${value}`);

      setFilterValues([
        { key: TaskRunColumn.additionalMetadata, value: newMetadata },
      ]);
    },
    [additionalMetadata, setFilterValues],
  );

  const setAllAdditionalMetadata = useCallback(
    ({ kvPairs }: { kvPairs: { key: string; value: string }[] }) => {
      const newMeta = kvPairs.map(({ key, value }) => `${key}:${value}`);

      setFilterValues([
        { key: TaskRunColumn.additionalMetadata, value: newMeta },
      ]);
    },
    [setFilterValues],
  );

  const clearColumnFilters = useCallback(() => {
    setFilterValues([
      { key: TaskRunColumn.workflow, value: undefined },
      { key: TaskRunColumn.additionalMetadata, value: undefined },
      { key: TaskRunColumn.status, value: undefined },
      { key: TaskRunColumn.parentTaskExternalId, value: undefined },
    ]);
  }, [setFilterValues]);

  const clearParentTaskExternalId = useCallback(() => {
    setFilterValues([
      {
        key: TaskRunColumn.parentTaskExternalId,
        value: undefined,
      },
    ]);
  }, [setFilterValues]);

  return {
    queryParamNames,
    filters: {
      createdAfter,
      finishedBefore,
      columnFilters,
      additionalMetadata,
      status,
      workflowId,
      isCustomTimeRange,
      timeWindow: timeWindowFilter,
      parentTaskExternalId,
    },
    setCustomTimeRange,
    setCreatedAfter,
    setFinishedBefore,
    setFilterValues,
    setStatus,
    setWorkflowId,
    setColumnFilters,
    setAdditionalMetadata,
    clearColumnFilters,
    clearParentTaskExternalId,
    setAllAdditionalMetadata,
    setParentTaskExternalId,
  };
};
