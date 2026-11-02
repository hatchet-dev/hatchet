import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries, V1ConcurrencyScope, V1ConcurrencyStrategy } from '@/lib/api';
import { withPolling } from '@/lib/api/polling';
import { useQuery } from '@tanstack/react-query';

export const KEYS_PAGE_SIZE = 50;
export const STRATEGIES_PAGE_SIZE = 20;

export function useConcurrencyStrategies(tenantId: string) {
  const { refetchInterval } = useRefetchInterval();

  return useQuery({
    ...withPolling(queries.v1Concurrency.strategies(tenantId), refetchInterval),
    placeholderData: (prev) => prev,
  });
}

export function useConcurrencyKeys({
  tenantId,
  strategy,
  search,
  page,
}: {
  tenantId: string;
  strategy: V1ConcurrencyStrategy | undefined;
  search: string;
  page: number;
}) {
  const { refetchInterval } = useRefetchInterval();

  return useQuery({
    ...withPolling(
      queries.v1Concurrency.keys(
        tenantId,
        strategy?.scope ?? V1ConcurrencyScope.V1ConcurrencyScopeTask,
        strategy?.id ?? 0,
        {
          search: search || undefined,
          limit: KEYS_PAGE_SIZE,
          offset: page * KEYS_PAGE_SIZE,
        },
      ),
      refetchInterval,
    ),
    enabled: !!strategy,
    placeholderData: (prev) => prev,
  });
}
