import { V1TaskStatus } from '@/lib/api';
import {
  ColumnFilter,
  ColumnFiltersState,
  Updater,
} from '@tanstack/react-table';
import { useCallback, useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export type TimeWindow = '1h' | '6h' | '1d' | '7d';

type FilterParams = {
  createdAfter: string;
  finishedBefore: string | undefined;
  isCustomTimeRange: boolean | undefined;
  status: V1TaskStatus | undefined;
  additionalMetadata: string[] | undefined;
  columnFilters: ColumnFiltersState;
  workflowId: string | undefined;
  timeWindow: TimeWindow | undefined;
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

  const timeWindowFilter = (searchParams.get('timeWindow') ||
    '1d') as TimeWindow;

  const isCustomTimeRange =
    searchParams.get('isCustomTimeRange') === 'true' || false;

  const { createdAfter, finishedBefore } = useMemo(() => {
    return parseTimeRange({
      isCustom: isCustomTimeRange,
      timeWindow: timeWindowFilter,
      createdAfter: searchParams.get('createdAfter'),
      finishedBefore: searchParams.get('finishedBefore'),
    });
  }, [isCustomTimeRange, timeWindowFilter, searchParams]);

  const status = searchParams.get('status') as V1TaskStatus | undefined;
  const additionalMetadataRaw = (searchParams.get('additionalMetadata') ||
    undefined) as string | undefined;

  const additionalMetadata = additionalMetadataRaw
    ? additionalMetadataRaw.split(',')
    : undefined;

  const workflowId = searchParams.get('workflowId') || undefined;

  const statusColumnFilter = status
    ? { id: 'status', value: status }
    : undefined;
  const additionalMetadataColumnFilter = additionalMetadata
    ? { id: 'additionalMetadata', value: additionalMetadata }
    : undefined;
  const workflowIdColumnFilter = workflowId
    ? { id: 'workflowId', value: workflowId }
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

        return newParams;
      });
    },
    [setSearchParams],
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
      setFilterValues([{ key: 'status', value: status }]);
    },
    [setFilterValues],
  );

  const setWorkflowId = useCallback(
    (workflowId: string | undefined) => {
      setFilterValues([{ key: 'workflowId', value: workflowId }]);
    },
    [setFilterValues],
  );

  const setAdditionalMetadata = useCallback(
    ({ key, value }: { key: string; value: string }) => {
      const existing = additionalMetadata || [];
      const newMetadata = existing.filter((m) => m.split(':')[0] !== key);

      newMetadata.push(`${key}:${value}`);

      setFilterValues([{ key: 'additionalMetadata', value: newMetadata }]);
    },
    [additionalMetadata, setFilterValues],
  );

  return {
    filters: {
      createdAfter,
      finishedBefore,
      columnFilters,
      additionalMetadata,
      status,
      workflowId,
      isCustomTimeRange,
      timeWindow: timeWindowFilter,
    },
    setCustomTimeRange,
    setCreatedAfter,
    setFinishedBefore,
    setFilterValues,
    setStatus,
    setWorkflowId,
    setColumnFilters,
    setAdditionalMetadata,
  };
};
