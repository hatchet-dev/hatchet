import api, { RateLimit as ApiRateLimit, RateLimit } from '@/next/lib/api';
import { useQuery } from '@tanstack/react-query';
import useTenant from './use-tenant';
import { PaginationManager, PaginationManagerNoOp } from './use-pagination';

interface RateLimitsState {
  data?: RateLimit[];
  isLoading: boolean;
}

interface UseRateLimitsOptions {
  refetchInterval?: number;
  paginationManager?: PaginationManager;
}

export default function useRateLimits({
  refetchInterval,
  paginationManager: pagination = PaginationManagerNoOp,
}: UseRateLimitsOptions = {}): RateLimitsState {
  const { tenant } = useTenant();

  const listRateLimitsQuery = useQuery({
    queryKey: ['rate-limit:list', tenant, pagination],
    queryFn: async () => {
      if (!tenant) {
        const p = {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
        pagination?.setNumPages(p.pagination.num_pages);
        return p;
      }

      const res = await api.rateLimitList(tenant?.metadata.id || '', {
        limit: pagination?.pageSize || 10,
        offset: (pagination?.currentPage - 1) * pagination?.pageSize || 0,
      });

      pagination?.setNumPages(res.data.pagination?.num_pages || 1);

      // Transform API response to match our local RateLimit type
      const transformedRows = (res.data.rows || []).map(
        (row: ApiRateLimit) => ({
          key: row.key || '',
          tenantId: row.tenantId || '',
          limitValue: row.limitValue || 0,
          value: row.value || 0,
          window: row.window || '',
          lastRefill: row.lastRefill || new Date().toISOString(),
        }),
      );

      return {
        ...res.data,
        rows: transformedRows,
      };
    },
    refetchInterval,
  });

  return {
    data: listRateLimitsQuery.data?.rows || [],
    isLoading: listRateLimitsQuery.isLoading,
  };
}
