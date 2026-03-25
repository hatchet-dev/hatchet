import { parseLogQuery } from "@/components/v1/cloud/logging/log-search/parser";
import { LOG_LEVEL_TO_API } from "@/components/v1/cloud/logging/log-search/types";
import { LogLine } from "@/components/v1/cloud/logging/log-search/use-logs";
import { useRefetchInterval } from "@/contexts/refetch-interval-context";
import { useCurrentTenantId } from "@/hooks/use-tenant";
import {
  V1LogLine,
  V1LogLineOrderByDirection,
  V1LogsPointMetric,
} from "@/lib/api";
import api from "@/lib/api/api";
import { useSearchParams } from "@/lib/router-helpers";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { z } from "zod";

const LOGS_PER_PAGE = 100;
const FILTER_KEY = "tenant-logs-filters";

export type TimeWindow = "1h" | "6h" | "1d" | "7d";

const filterSchema = z.object({
  q: z.string().optional(),
  tw: z.enum(["1h", "6h", "1d", "7d"]).default("1d"),
  // custom time range set by chart drag; overrides tw when present
  since: z.string().optional(),
  until: z.string().optional(),
});

type FilterState = z.infer<typeof filterSchema>;

function getSinceFromTimeWindow(tw: TimeWindow): string {
  const hours = { "1h": 1, "6h": 6, "1d": 24, "7d": 168 }[tw];
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function logKey(log: LogLine): string {
  return `${log.timestamp ?? ""}-${log.line ?? ""}`;
}

function mapToLogLines(rows: V1LogLine[]): LogLine[] {
  return rows.map((row) => ({
    timestamp: row.createdAt,
    line: row.message,
    level: row.level,
    attempt: row.attempt,
    taskExternalId: row.taskExternalId,
    taskDisplayName: row.taskDisplayName,
  }));
}

export function useTenantLogs() {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const [searchParams, setSearchParams] = useSearchParams();

  // URL-stored filter state
  const filters = useMemo<FilterState>(() => {
    const raw = searchParams.get(FILTER_KEY);
    try {
      return filterSchema.parse(raw ? JSON.parse(raw) : {});
    } catch {
      return filterSchema.parse({});
    }
  }, [searchParams]);

  const setFilters = useCallback(
    (update: Partial<FilterState>) => {
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [FILTER_KEY]: JSON.stringify({ ...filters, ...update }),
      }));
    },
    [filters, setSearchParams],
  );

  const parsedQuery = useMemo(
    () => parseLogQuery(filters.q ?? ""),
    [filters.q],
  );

  // Stable since: computed once per filter change, not on every render
  const [since, setSince] = useState(
    () => filters.since ?? getSinceFromTimeWindow(filters.tw),
  );

  useEffect(() => {
    setSince(filters.since ?? getSinceFromTimeWindow(filters.tw));
  }, [
    filters.tw,
    filters.since,
    filters.until,
    parsedQuery.level,
    parsedQuery.search,
  ]);

  const logsQuery = useInfiniteQuery({
    queryKey: [
      "tenant-logs",
      tenantId,
      since,
      filters.until,
      parsedQuery.level,
      parsedQuery.search,
    ],
    queryFn: async ({ pageParam }: { pageParam: string | undefined }) => {
      const response = await api.v1TenantLogLineList(tenantId, {
        limit: LOGS_PER_PAGE,
        since,
        ...(pageParam
          ? { until: pageParam }
          : filters.until
            ? { until: filters.until }
            : {}),
        ...(parsedQuery.level && {
          levels: [LOG_LEVEL_TO_API[parsedQuery.level]],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
        order_by_direction: V1LogLineOrderByDirection.DESC,
      });
      return response.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      const rows = lastPage.rows ?? [];
      if (rows.length < LOGS_PER_PAGE) {
        return undefined;
      }
      return rows[rows.length - 1].createdAt;
    },
    refetchInterval,
    enabled: !!tenantId,
  });

  const logs = useMemo<LogLine[]>(() => {
    const seen = new Set<string>();
    const result: LogLine[] = [];
    for (const page of logsQuery.data?.pages ?? []) {
      for (const log of mapToLogLines(page.rows ?? [])) {
        const k = logKey(log);
        if (!seen.has(k)) {
          seen.add(k);
          result.push(log);
        }
      }
    }
    return result;
  }, [logsQuery.data]);

  const fetchOlderLogs = useCallback(() => {
    if (logsQuery.hasNextPage && !logsQuery.isFetchingNextPage) {
      logsQuery.fetchNextPage();
    }
  }, [logsQuery]);

  const metricsQuery = useQuery({
    queryKey: [
      "tenant-log-metrics",
      tenantId,
      since,
      filters.until,
      parsedQuery.level,
      parsedQuery.search,
    ],
    queryFn: async () => {
      const response = await api.v1TenantLogLineGetPointMetrics(tenantId, {
        since,
        ...(filters.until && { until: filters.until }),
        ...(parsedQuery.level && {
          levels: [LOG_LEVEL_TO_API[parsedQuery.level]],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
      });
      return response.data;
    },
    refetchInterval,
    enabled: !!tenantId,
  });

  const setCustomTimeRange = useCallback(
    (newSince: string, newUntil: string) => {
      setFilters({ since: newSince, until: newUntil });
    },
    [setFilters],
  );

  const setTimeWindow = useCallback(
    (tw: TimeWindow) => {
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [FILTER_KEY]: JSON.stringify({
          ...filters,
          tw,
          since: undefined,
          until: undefined,
        }),
      }));
    },
    [filters, setSearchParams],
  );

  const clearTimeRange = useCallback(() => {
    setSearchParams((prev) => ({
      ...Object.fromEntries(prev.entries()),
      [FILTER_KEY]: JSON.stringify({
        ...filters,
        since: undefined,
        until: undefined,
      }),
    }));
  }, [filters, setSearchParams]);

  const setCustomSince = useCallback(
    (newSince: string | undefined) => setFilters({ since: newSince }),
    [setFilters],
  );

  const setCustomUntil = useCallback(
    (newUntil: string | undefined) => setFilters({ until: newUntil }),
    [setFilters],
  );

  const isCustomTimeRange = !!filters.since;

  return {
    logs,
    isLoading: logsQuery.isLoading,
    isRefetching: logsQuery.isRefetching,
    isFetchingMore: logsQuery.isFetchingNextPage,
    refetch: logsQuery.refetch,
    fetchOlderLogs,
    queryString: filters.q ?? "",
    setQueryString: (q: string) => setFilters({ q }),
    parsedQuery,
    // chart
    chartSince: since,
    chartUntil: filters.until,
    chartMetrics: (metricsQuery.data?.results ?? []) as V1LogsPointMetric[],
    setCustomTimeRange,
    // time window selector
    timeWindow: filters.tw,
    isCustomTimeRange,
    customSince: filters.since,
    customUntil: filters.until,
    setTimeWindow,
    clearTimeRange,
    setCustomSince,
    setCustomUntil,
  };
}
