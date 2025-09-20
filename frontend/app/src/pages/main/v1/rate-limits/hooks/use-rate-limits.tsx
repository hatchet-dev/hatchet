import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  queries,
  RateLimit,
  RateLimitOrderByDirection,
  RateLimitOrderByField,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { ColumnFiltersState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { keyKey } from '../components/rate-limit-columns';
import { useDebounce } from 'use-debounce';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

type RateLimitQueryShape = {
  s: string | undefined; // search
};

export type RateLimitWithMetadata = RateLimit & {
  metadata: {
    id: string;
  };
};

const parseRateLimitParam = (
  useSearchParams: URLSearchParams,
  key: string,
): RateLimitQueryShape => {
  const rawFilterParamValue = useSearchParams.get(key);

  if (!rawFilterParamValue) {
    return {
      s: undefined,
    };
  }

  const parsedFilterState = JSON.parse(rawFilterParamValue);

  if (!parsedFilterState || typeof parsedFilterState !== 'object') {
    return {
      s: undefined,
    };
  }

  const { s }: RateLimitQueryShape = parsedFilterState;

  return {
    s,
  };
};

export const useRateLimits = ({ key }: { key: string }) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenantId } = useCurrentTenantId();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });
  const { refetchInterval } = useRefetchInterval();

  const paramKey = `rate-limits-${key}`;

  const search = useMemo(() => {
    const { s } = parseRateLimitParam(searchParams, paramKey);
    return s;
  }, [searchParams, paramKey]);

  const [debouncedSearch] = useDebounce(search, 300);

  const columnFilters = useMemo<ColumnFiltersState>(() => {
    const { s } = parseRateLimitParam(searchParams, paramKey);
    const filters: ColumnFiltersState = [];

    if (s) {
      filters.push({ id: keyKey, value: s });
    }

    return filters;
  }, [searchParams, paramKey]);

  const setColumnFilters = useCallback(
    (updater: Updater<ColumnFiltersState>) => {
      setSearchParams((prev) => {
        const currentColumnFilters = columnFilters;
        const newColumnFilters =
          typeof updater === 'function'
            ? updater(currentColumnFilters)
            : updater;

        const searchFilter = newColumnFilters.find((f) => f.id === keyKey);

        const filterState: RateLimitQueryShape = {
          s: searchFilter ? String(searchFilter.value) : undefined,
        };

        return {
          ...Object.fromEntries(prev.entries()),
          [paramKey]: JSON.stringify(filterState),
        };
      });
    },
    [paramKey, setSearchParams, columnFilters],
  );

  const { data, isLoading, error } = useQuery({
    ...queries.rate_limits.list(tenantId, {
      search: debouncedSearch,
      orderByField: RateLimitOrderByField.Key,
      orderByDirection: RateLimitOrderByDirection.Asc,
      offset,
      limit,
    }),
    refetchInterval,
    placeholderData: (data) => data,
  });

  const rateLimits = useMemo(() => data?.rows ?? [], [data]).map((rl) => ({
    ...rl,
    metadata: {
      id: rl.key,
    },
  }));

  const numPages = useMemo(() => data?.pagination?.num_pages ?? 0, [data]);

  return {
    data: rateLimits,
    numPages,
    isLoading,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
  };
};
