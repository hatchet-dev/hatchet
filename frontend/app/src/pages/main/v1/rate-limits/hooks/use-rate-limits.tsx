import { keyKey } from '../components/rate-limit-columns';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import {
  queries,
  RateLimit,
  RateLimitOrderByDirection,
  RateLimitOrderByField,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useDebounce } from 'use-debounce';
import { z } from 'zod';

const rateLimitQuerySchema = z
  .object({
    s: z.string().optional(), // search
  })
  .default({ s: undefined });

export type RateLimitWithMetadata = RateLimit & {
  metadata: {
    id: string;
  };
};

export const useRateLimits = ({ key }: { key: string }) => {
  const { tenantId } = useCurrentTenantId();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });
  const { refetchInterval } = useRefetchInterval();

  const paramKey = `rate-limits-${key}`;

  const {
    state: { s: search },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(rateLimitQuerySchema, paramKey, { s: keyKey });

  const [debouncedSearch] = useDebounce(search, 300);

  const { data, isLoading, error, isRefetching, refetch } = useQuery({
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

  const rateLimits = useMemo(
    () =>
      (data?.rows ?? []).map((rl) => ({
        ...rl,
        metadata: {
          id: rl.key,
        },
      })),
    [data],
  );

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
    isRefetching,
    refetch,
    resetFilters,
  };
};
