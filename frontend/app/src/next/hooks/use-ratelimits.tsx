import api, { RateLimit as ApiRateLimit, RateLimit } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useTenant } from './use-tenant';
import { createContext, useContext, PropsWithChildren } from 'react';
import {
  PaginationProvider,
  usePagination,
  PaginationProviderProps,
} from './utils/use-pagination';
import { useToast } from './utils/use-toast';

interface RateLimitsState {
  data?: RateLimit[];
  isLoading: boolean;
  pagination: ReturnType<typeof usePagination>;
}

const RateLimitsContext = createContext<RateLimitsState | null>(null);

interface RateLimitsProviderProps extends PropsWithChildren {
  initialPagination?: PaginationProviderProps;
}

export function RateLimitsProvider({
  children,
  initialPagination = {
    initialPageSize: 50,
  },
}: RateLimitsProviderProps) {
  return (
    <PaginationProvider {...initialPagination}>
      <RateLimitsProviderContent>{children}</RateLimitsProviderContent>
    </PaginationProvider>
  );
}

function RateLimitsProviderContent({ children }: PropsWithChildren) {
  const pagination = usePagination();
  const { tenant } = useTenant();
  const { toast } = useToast();

  const listRateLimitsQuery = useQuery({
    queryKey: ['rate-limit:list', tenant, pagination],
    queryFn: async () => {
      if (!tenant) {
        const p = {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
        pagination.setNumPages(p.pagination.num_pages);
        return p;
      }

      try {
        const res = await api.rateLimitList(tenant?.metadata.id || '', {
          limit: pagination.pageSize || 10,
          offset: (pagination.currentPage - 1) * pagination.pageSize || 0,
        });

        pagination.setNumPages(res.data.pagination?.num_pages || 1);

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
      } catch (error) {
        toast({
          title: 'Error fetching rate limits',

          variant: 'destructive',
          error,
        });
        return {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
      }
    },
  });

  const value = {
    data: listRateLimitsQuery.data?.rows || [],
    isLoading: listRateLimitsQuery.isLoading,
    pagination,
  };

  return (
    <RateLimitsContext.Provider value={value}>
      {children}
    </RateLimitsContext.Provider>
  );
}

export function useRateLimits() {
  const context = useContext(RateLimitsContext);
  if (!context) {
    throw new Error('useRateLimits must be used within a RateLimitsProvider');
  }
  return context;
}
