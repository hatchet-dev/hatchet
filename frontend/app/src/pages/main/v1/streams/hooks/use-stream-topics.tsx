import { topicKey } from '../components/stream-topic-columns';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { queries, V1StreamTopic } from '@/lib/api';
import { withPolling } from '@/lib/api/polling';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useDebounce } from 'use-debounce';
import { z } from 'zod';

const streamTopicQuerySchema = z
  .object({
    s: z.string().optional(), // search
  })
  .default({ s: undefined });

export type StreamTopicWithMetadata = V1StreamTopic & {
  metadata: {
    id: string;
  };
};

export const useStreamTopics = ({ key }: { key: string }) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const paramKey = `stream-topics-${key}`;

  const {
    state: { s: search },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(streamTopicQuerySchema, paramKey, { s: topicKey });

  const [debouncedSearch] = useDebounce(search, 300);

  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
      resetPageOnChange: [debouncedSearch],
    });

  const { data, isLoading, error, isRefetching, isPlaceholderData, refetch } =
    useQuery({
      // each fetch counts every retained message of the topics on the page
      ...withPolling(
        queries.stream_topics.list(tenantId, {
          search: debouncedSearch,
          offset,
          limit,
        }),
        refetchInterval,
      ),
      placeholderData: (data) => data,
    });

  const topics = useMemo(
    () =>
      (data?.rows ?? []).map((t) => ({
        ...t,
        metadata: {
          // the null byte can't appear in a namespace, so ids can't collide
          id: `${t.namespace}\u0000${t.topic}`,
        },
      })),
    [data],
  );

  const numPages = useMemo(() => data?.pagination?.num_pages ?? 0, [data]);

  return {
    data: topics,
    retentionHours: data?.retentionHours,
    numPages,
    isLoading,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    isRefetching,
    isPlaceholderData,
    refetch,
    resetFilters,
  };
};
