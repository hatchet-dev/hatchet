import { parseLogQuery } from './parser';
import { ParsedLogQuery } from './types';
import {
  V1TaskSummary,
  V1LogLineList,
  V1TaskStatus,
  V1LogLineLevel,
} from '@/lib/api';
import api from '@/lib/api/api';
import { V1LogLineListQuery } from '@/lib/api/queries';
import {
  useInfiniteQuery,
  InfiniteData,
  useQueryClient,
} from '@tanstack/react-query';
import {
  useMemo,
  useCallback,
  useRef,
  useEffect,
  useState,
  createContext,
  useContext,
  type ReactNode,
} from 'react';

const LOGS_PER_PAGE = 100;

export interface LogLine {
  timestamp?: string;
  line?: string;
  instance?: string;
  level?: string;
  metadata?: Record<string, unknown>;
}

export interface UseLogsOptions {
  taskRun: V1TaskSummary | undefined;
  resetTrigger?: number;
}

export interface UseLogsReturn {
  logs: LogLine[];
  isLoading: boolean;
  isFetchingMore: boolean;
  queryString: string;
  setQueryString: (value: string) => void;
  parsedQuery: ParsedLogQuery;
  fetchOlderLogs: () => void;
}

export function useLogs({
  taskRun,
  resetTrigger,
}: UseLogsOptions): UseLogsReturn {
  const queryClient = useQueryClient();
  const lastPageTimestampRef = useRef<string | undefined>(undefined);

  const [queryString, setQueryString] = useState('');
  const parsedQuery = useMemo(() => parseLogQuery(queryString), [queryString]);

  const isTaskRunning = taskRun?.status === V1TaskStatus.RUNNING;

  useEffect(() => {
    if (taskRun?.metadata.id) {
      queryClient.resetQueries({
        queryKey: ['v1Tasks', 'getLogs', taskRun.metadata.id],
        exact: false,
      });
      lastPageTimestampRef.current = undefined;
    }
  }, [
    resetTrigger,
    queryClient,
    taskRun?.metadata.id,
    parsedQuery.level,
    parsedQuery.search,
  ]);

  const getLogsQuery = useInfiniteQuery<
    V1LogLineList,
    Error,
    InfiniteData<V1LogLineList>,
    (string | undefined)[],
    { since: string | undefined; until: string | undefined }
  >({
    queryKey: [
      'v1Tasks',
      'getLogs',
      taskRun?.metadata.id,
      parsedQuery.level,
      parsedQuery.search,
    ],
    queryFn: async ({ pageParam }) => {
      const params: V1LogLineListQuery = {
        limit: LOGS_PER_PAGE,
        ...(pageParam && { since: pageParam.since, until: pageParam.until }),
        ...(parsedQuery.level && {
          levels: [parsedQuery.level.toUpperCase() as V1LogLineLevel],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
      };

      const response = await api.v1LogLineList(
        taskRun?.metadata.id || '',
        params,
      );
      const rows = response.data.rows;

      // API returns logs in descending order (newest first, oldest last)
      // Track the newest timestamp for running task polling (first row is newest)
      const newestTimestamp = rows?.[0]?.createdAt;
      if (isTaskRunning && newestTimestamp) {
        lastPageTimestampRef.current = newestTimestamp;
      }

      return response.data;
    },
    initialPageParam: { since: undefined, until: undefined },
    enabled: !!taskRun,
    maxPages: 0, // Keep all pages in memory
    refetchInterval: false,
    staleTime: Infinity,
    getNextPageParam: (lastPage) => {
      const rows = lastPage?.rows;
      // API returns descending order: first row is newest, last row is oldest
      if (!isTaskRunning && rows && rows.length === LOGS_PER_PAGE) {
        // Fetch older logs: use the last (oldest) log's timestamp as 'until'
        const oldestLog = rows[rows.length - 1];
        return { since: undefined, until: oldestLog?.createdAt };
      } else if (isTaskRunning) {
        // For running tasks, fetch newer logs using 'since' with newest timestamp
        return { since: lastPageTimestampRef.current, until: undefined };
      }
      return undefined;
    },
  });

  // Poll for new logs when task is running
  useEffect(() => {
    if (!isTaskRunning) {
      return;
    }

    const interval = setInterval(() => {
      if (!getLogsQuery.isFetchingNextPage) {
        getLogsQuery.fetchNextPage();
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isTaskRunning, getLogsQuery]);

  const logs = useMemo((): LogLine[] => {
    if (!getLogsQuery.data?.pages) {
      return [];
    }

    return getLogsQuery.data.pages.flatMap(
      (page) =>
        page?.rows?.map((row) => ({
          timestamp: row.createdAt,
          line: row.message,
          instance: taskRun?.displayName,
          level: row.level,
          metadata: row.metadata as Record<string, unknown> | undefined,
        })) || [],
    );
  }, [getLogsQuery.data?.pages, taskRun?.displayName]);

  const fetchOlderLogs = useCallback(() => {
    if (
      !getLogsQuery.isFetchingNextPage &&
      !isTaskRunning &&
      getLogsQuery.hasNextPage
    ) {
      getLogsQuery.fetchNextPage();
    }
  }, [getLogsQuery, isTaskRunning]);

  return {
    logs,
    isLoading: getLogsQuery.isLoading,
    isFetchingMore: getLogsQuery.isFetchingNextPage,
    queryString,
    setQueryString,
    parsedQuery,
    fetchOlderLogs,
  };
}

const LogsContext = createContext<UseLogsReturn | null>(null);

export function LogsProvider({
  taskRun,
  resetTrigger,
  children,
}: UseLogsOptions & { children: ReactNode }) {
  const logsState = useLogs({ taskRun, resetTrigger });

  return (
    <LogsContext.Provider value={logsState}>{children}</LogsContext.Provider>
  );
}

export function useLogsContext(): UseLogsReturn {
  const context = useContext(LogsContext);
  if (!context) {
    throw new Error('useLogsContext must be used within a LogsProvider');
  }
  return context;
}
