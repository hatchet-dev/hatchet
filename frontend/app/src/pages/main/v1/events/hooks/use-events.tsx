import {
  keyKey,
  workflowKey,
  statusKey,
  metadataKey,
  idKey,
  scopeKey,
} from '../components/event-columns';
import {
  FilterOption,
  TimeRangeConfig,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import api, { queries, V1TaskStatus } from '@/lib/api';
import { useSearchParams } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { z } from 'zod';

const eventStatusFilters: FilterOption[] = [
  { value: V1TaskStatus.COMPLETED, label: 'Succeeded' },
  { value: V1TaskStatus.FAILED, label: 'Failed' },
  { value: V1TaskStatus.RUNNING, label: 'Running' },
  { value: V1TaskStatus.QUEUED, label: 'Queued' },
  { value: V1TaskStatus.CANCELLED, label: 'Cancelled' },
];

type UseEventsProps = {
  key: string;
};

type TimeWindow = '1h' | '6h' | '1d' | '7d';

const TIME_KEY = 'events-time';

const timeSchema = z.object({
  tw: z.enum(['1h', '6h', '1d', '7d']).default('1d'),
  since: z.string().optional(),
  until: z.string().optional(),
});

type TimeState = z.infer<typeof timeSchema>;

function getSinceFromTimeWindow(tw: TimeWindow): string {
  const hours: Record<TimeWindow, number> = {
    '1h': 1,
    '6h': 6,
    '1d': 24,
    '7d': 168,
  };
  return new Date(Date.now() - hours[tw] * 60 * 60 * 1000).toISOString();
}

const eventFilterSchema = z
  .object({
    k: z.array(z.string()).default([]), // keys
    w: z.array(z.string()).default([]), // workflow ids
    s: z.array(z.string()).default([]), // scopes
    st: z.array(z.nativeEnum(V1TaskStatus)).default([]), // statuses
    m: z.array(z.string()).default([]), // metadata
    i: z.array(z.string()).default([]), // event ids
  })
  .default({});

export const useEvents = ({ key }: UseEventsProps) => {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { refetchInterval } = useRefetchInterval();
  const [searchParams, setSearchParams] = useSearchParams();

  // Time range state
  const timeState = useMemo<TimeState>(() => {
    const raw = searchParams.get(TIME_KEY);
    try {
      return timeSchema.parse(raw ? JSON.parse(raw) : {});
    } catch {
      return timeSchema.parse({});
    }
  }, [searchParams]);

  const setTimeState = useCallback(
    (update: Partial<TimeState>) => {
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [TIME_KEY]: JSON.stringify({ ...timeState, ...update }),
      }));
    },
    [timeState, setSearchParams],
  );

  const timeWindow = timeState.tw;
  const isCustomTimeRange = !!timeState.since;

  // Stabilize `since` so Date.now() is not called on every render
  const [since, setSince] = useState(
    () => timeState.since ?? getSinceFromTimeWindow(timeState.tw),
  );
  useEffect(() => {
    setSince(timeState.since ?? getSinceFromTimeWindow(timeState.tw));
  }, [timeState.tw, timeState.since, timeState.until]);

  const until = timeState.until;

  const setTimeWindow = useCallback(
    (tw: TimeWindow) => {
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [TIME_KEY]: JSON.stringify({ tw, since: undefined, until: undefined }),
      }));
    },
    [setSearchParams],
  );

  const setCustomTimeRange = useCallback(
    (newSince: string, newUntil: string) => {
      setTimeState({ since: newSince, until: newUntil });
    },
    [setTimeState],
  );

  const clearTimeRange = useCallback(() => {
    setSearchParams((prev) => {
      const next = Object.fromEntries(prev.entries());
      delete next[TIME_KEY];
      return next;
    });
  }, [setSearchParams]);

  const paramKey = `events-${key}`;
  const {
    state: {
      k: selectedKeys,
      w: selectedWorkflowIds,
      s: selectedScopes,
      st: selectedStatuses,
      m: selectedMetadata,
      i: selectedEventIds,
    },
    columnFilters,
    setColumnFilters,
    resetFilters: resetColumnFilters,
  } = useZodColumnFilters(eventFilterSchema, paramKey, {
    k: keyKey,
    w: workflowKey,
    s: scopeKey,
    st: statusKey,
    m: metadataKey,
    i: idKey,
  });

  const resetFilters = useCallback(() => {
    resetColumnFilters();
    clearTimeRange();
  }, [resetColumnFilters, clearTimeRange]);

  const hasActiveColumnFilters = columnFilters.length > 0;
  const hasActiveFilters =
    hasActiveColumnFilters || isCustomTimeRange || timeWindow !== '1d';
  const isDefaultOneDayWindow = !isCustomTimeRange && timeWindow === '1d';

  const timeRangeConfig: TimeRangeConfig = useMemo(
    () => ({
      onTimeWindowChange: (value: string) => {
        if (value !== 'custom') {
          setTimeWindow(value as TimeWindow);
        } else {
          setCustomTimeRange(since, new Date().toISOString());
        }
      },
      onCreatedAfterChange: (date?: string) => {
        if (isCustomTimeRange && until) {
          setCustomTimeRange(date ?? getSinceFromTimeWindow('1d'), until);
        }
      },
      onFinishedBeforeChange: (date?: string) => {
        if (isCustomTimeRange && since) {
          setCustomTimeRange(since, date ?? new Date().toISOString());
        }
      },
      onClearTimeRange: clearTimeRange,
      currentTimeWindow: timeWindow,
      isCustomTimeRange,
      createdAfter: isCustomTimeRange ? since : undefined,
      finishedBefore: until,
    }),
    [
      timeWindow,
      isCustomTimeRange,
      since,
      until,
      setTimeWindow,
      setCustomTimeRange,
      clearTimeRange,
    ],
  );

  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
      resetPageOnChange: [
        selectedKeys,
        selectedWorkflowIds,
        selectedScopes,
        selectedStatuses,
        selectedMetadata,
        selectedEventIds,
        since,
        until,
      ],
    });

  const { data, isLoading, refetch, error, isRefetching } = useQuery({
    queryKey: [
      'v1:events:list',
      tenantId,
      {
        keys: selectedKeys,
        workflows: selectedWorkflowIds,
        offset,
        limit,
        statuses: selectedStatuses,
        additionalMetadata: selectedMetadata,
        eventIds: selectedEventIds,
        scopes: selectedScopes,
        since,
        until,
      },
    ],
    queryFn: async () => {
      const response = await api.v1EventList(tenantId, {
        offset,
        limit,
        keys: selectedKeys,
        since,
        until,
        eventIds: selectedEventIds,
        workflowRunStatuses: selectedStatuses,
        additionalMetadata: selectedMetadata,
        workflowIds: selectedWorkflowIds,
        scopes: selectedScopes,
      });

      return response.data;
    },
    refetchInterval: selectedEventIds?.length ? false : refetchInterval,
    placeholderData: (prev) => prev,
  });

  const events = data?.rows ?? [];
  const numEvents = data?.pagination?.num_pages ?? 1;

  const { data: eventKeys, error: eventKeysError } = useQuery({
    queryKey: ['v1:events:listKeys', tenantId],
    queryFn: async () => {
      const response = await api.v1EventKeyList(tenantId);
      return response.data;
    },
    staleTime: 5 * 60 * 1000,
  });

  const eventKeyFilters = useMemo((): FilterOption[] => {
    return (
      eventKeys?.rows?.map((key) => ({
        value: key,
        label: key,
      })) || []
    );
  }, [eventKeys]);

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  return {
    events,
    numEvents,
    isLoading: isLoading || workflowKeysIsLoading,
    refetch,
    error: error || eventKeysError || workflowKeysError,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    selectedKeys,
    selectedWorkflowIds,
    selectedScopes,
    selectedStatuses,
    selectedMetadata,
    selectedEventIds,
    eventKeyFilters,
    workflowKeyFilters,
    workflowRunStatusFilters: eventStatusFilters,
    isRefetching,
    resetFilters,
    // time range
    timeWindow,
    isCustomTimeRange,
    timeRangeConfig,
    hasActiveFilters,
    isDefaultOneDayWindow,
    setTimeWindow,
  };
};
