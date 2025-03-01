import { V1TaskStatus } from '@/lib/api';
import { Updater } from '@tanstack/react-query';
import { ColumnFilter, ColumnFiltersState } from '@tanstack/react-table';
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
    const additionalMetadata = (searchParams.get('additionalMetadata') ||
      undefined) as string[] | undefined;

    const statusColumnFilter = status
      ? { id: 'status', value: status }
      : undefined;
    const additionalMetadataColumnFilter = additionalMetadata
      ? { id: 'additionalMetadata', value: additionalMetadata }
      : undefined;

    const columnFilters: ColumnFiltersState = [
      statusColumnFilter,
      additionalMetadataColumnFilter,
    ].filter((f) => f !== undefined) as ColumnFiltersState;

    return {
      defaultTimeRange,
      createdAfter,
      finishedBefore,
      isCustomTimeRange,
      status,
      additionalMetadata,
      columnFilters,
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
        const existing = filters.additionalMetadata || [];
        console.log(f.value);

        return { key: 'additionalMetadata', value: [...existing, f.value] };
      default:
        return { key: f.id as ColumnFilterKey, value: f.value };
    }
  };

  const setColumnFilters = useCallback(
    (updaterOrValue: Updater<ColumnFiltersState, any[]>) => {
      if (typeof updaterOrValue === 'function') {
        const newVal = updaterOrValue(filters.columnFilters);

        console.log(newVal);
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

  // Implementations
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

  const setStatuses = useCallback(
    (status: V1TaskStatus | undefined) => {
      setFilterValues([{ key: 'status', value: status }]);
    },
    [setFilterValues],
  );

  const setAdditionalMetadata = useCallback(
    (key: string, value: string) => {
      const existing = filters.additionalMetadata || [];
      const newMetadata = existing.filter((m) => m.split(':')[0] !== key);

      console.log(existing, newMetadata);
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
    setStatuses,
    setColumnFilters,
    setAdditionalMetadata,
  };
};
