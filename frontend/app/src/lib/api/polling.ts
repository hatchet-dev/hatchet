import { defaultQueryRetry } from '@/lib/query-retry';
import {
  hashKey,
  Query,
  QueryCacheNotifyEvent,
  QueryFunction,
  QueryKey,
} from '@tanstack/react-query';

const lastFetchDurationMsByQueryHash = new Map<string, number>();

export const forgetFetchDurationOnQueryRemoval = (
  event: QueryCacheNotifyEvent,
) => {
  if (event.type === 'removed') {
    lastFetchDurationMsByQueryHash.delete(event.query.queryHash);
  }
};

export const nextPollIntervalMs = (
  configuredIntervalMs: number | false,
  lastFetchDurationMs: number | undefined,
): number | false => {
  if (configuredIntervalMs === false) {
    return false;
  }

  if (lastFetchDurationMs === undefined) {
    return configuredIntervalMs;
  }

  return Math.max(configuredIntervalMs, 2 * lastFetchDurationMs);
};

type PollableQueryOptions<TData, TQueryKey extends QueryKey> = {
  queryKey: TQueryKey;
  queryFn: QueryFunction<TData, TQueryKey>;
};

/**
 * Polling settings for queries that can be slow on the API side:
 * - the poll interval stretches to at least twice the previous fetch's
 *   duration, so a 30s query is never re-issued back to back
 * - no retries while polling: the next poll is the retry, and retrying on
 *   top of polling multiplies load on an API that is already struggling
 * - no window-focus refetch, since the interval already resumes on focus
 * - data is fresh for one interval, so remounts within it reuse the cache
 */
export const withPolling = <TData, TQueryKey extends QueryKey>(
  options: PollableQueryOptions<TData, TQueryKey>,
  refetchIntervalMs: number | false,
) => ({
  ...options,
  queryFn: async (context: Parameters<QueryFunction<TData, TQueryKey>>[0]) => {
    const startedAt = performance.now();

    try {
      return await options.queryFn(context);
    } finally {
      lastFetchDurationMsByQueryHash.set(
        hashKey(context.queryKey),
        performance.now() - startedAt,
      );
    }
  },
  refetchInterval: (query: Query<TData, Error, TData, TQueryKey>) =>
    nextPollIntervalMs(
      refetchIntervalMs,
      lastFetchDurationMsByQueryHash.get(query.queryHash),
    ),
  retry: refetchIntervalMs === false ? defaultQueryRetry : false,
  refetchOnWindowFocus: false,
  staleTime: refetchIntervalMs || 0,
});
