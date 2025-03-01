import { V1TaskStatus } from '@/lib/api';
import { ColumnFilter, ColumnFiltersState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

type FilterParams = {
  defaultTimeRange: string | undefined;
  createdAfter: string | undefined;
  finishedBefore: string | undefined;
  isCustomTimeRange: boolean | undefined;
  status: V1TaskStatus | undefined;
  additionalMetadata: string[] | undefined;
  columnFilters: ColumnFiltersState;
  workflowId: string | undefined;
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

export const useColumnFilters = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const parseFiltersFromParams = (): FilterParams => {
    const defaultTimeRange = searchParams.get('defaultTimeRange') || undefined;
    const createdAfter = searchParams.get('createdAfter') || undefined;
    const finishedBefore = searchParams.get('finishedBefore') || undefined;
    const isCustomTimeRange =
      searchParams.get('isCustomTimeRange') === 'true' || undefined;

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

    return {
      defaultTimeRange,
      createdAfter,
      finishedBefore,
      isCustomTimeRange,
      status,
      additionalMetadata,
      columnFilters,
      workflowId,
    };
  };

  const filters = useMemo(() => parseFiltersFromParams(), [searchParams]);

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
        const newVal = updaterOrValue(filters.columnFilters);

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
    [],
  );

  const setDefaultTimeRange = useCallback(
    (timeRange: string) => {
      setFilterValues([{ key: 'defaultTimeRange', value: timeRange }]);
    },
    [setFilterValues],
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
    (key: string, value: string) => {
      const existing = filters.additionalMetadata || [];
      const newMetadata = existing.filter((m) => m.split(':')[0] !== key);

      newMetadata.push(`${key}:${value}`);

      setFilterValues([{ key: 'additionalMetadata', value: newMetadata }]);
    },
    [filters.additionalMetadata, setFilterValues],
  );

  return {
    filters,
    setCustomTimeRange,
    setDefaultTimeRange,
    setCreatedAfter,
    setFinishedBefore,
    setFilterValues,
    setStatus,
    setWorkflowId,
    setColumnFilters,
    setAdditionalMetadata,
  };
};
