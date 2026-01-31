import { parseLogQuery } from './parser';
import { ParsedLogQuery } from './types';
import {
  V1TaskSummary,
  V1LogLineList,
  V1LogLine,
  V1TaskStatus,
  V1LogLineLevel,
  V1LogLineOrderByDirection,
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

const getLogLineKey = (log: V1LogLine): string => {
  return `${log.createdAt}-${log.message}`;
};

export interface LogLine {
  timestamp?: string;
  line?: string;
  instance?: string;
  level?: string;
  metadata?: Record<string, unknown>;
  attempt?: number;
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
  setPollingEnabled: (enabled: boolean) => void;
  taskStatus: V1TaskStatus | undefined;
  availableAttempts: number[];
}

export function useLogs({
  taskRun,
  resetTrigger,
}: UseLogsOptions): UseLogsReturn {
  const queryClient = useQueryClient();
  const lastPageTimestampRef = useRef<string | undefined>(undefined);
  const isPollingRef = useRef(false);
  const timeoutIdRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [queryString, setQueryString] = useState('');
  const [isPollingEnabled, setPollingEnabled] = useState(true);
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
    parsedQuery.attempt,
  ]);

  const getLogsQuery = useInfiniteQuery<
    V1LogLineList,
    Error,
    InfiniteData<V1LogLineList>,
    (string | number | undefined)[],
    { since: string | undefined; until: string | undefined }
  >({
    queryKey: [
      'v1Tasks',
      'getLogs',
      taskRun?.metadata.id,
      parsedQuery.level,
      parsedQuery.search,
      parsedQuery.attempt,
    ],
    queryFn: async ({ pageParam }) => {
      const params: V1LogLineListQuery = {
        limit: LOGS_PER_PAGE,
        ...(pageParam && { since: pageParam.since, until: pageParam.until }),
        ...(parsedQuery.level && {
          levels: [parsedQuery.level.toUpperCase() as V1LogLineLevel],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
        ...(parsedQuery.attempt && { attempt: parsedQuery.attempt }),
        order_by_direction: V1LogLineOrderByDirection.DESC,
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
      if (rows && rows.length === LOGS_PER_PAGE) {
        const oldestLog = rows[rows.length - 1];
        return { since: undefined, until: oldestLog?.createdAt };
      }
      return undefined;
    },
  });

  // Poll for new logs when task is running and polling is enabled
  useEffect(() => {
    if (!isTaskRunning || !isPollingEnabled || !taskRun?.metadata.id) {
      return;
    }

    // Clear any existing timeout and reset polling state when starting a new polling session
    if (timeoutIdRef.current) {
      clearTimeout(timeoutIdRef.current);
    }
    isPollingRef.current = false;
    timeoutIdRef.current = null;

    const pollForNewLogs = async () => {
      // Prevent overlapping requests
      if (isPollingRef.current) {
        return;
      }

      isPollingRef.current = true;

      try {
        const params: V1LogLineListQuery = {
          limit: LOGS_PER_PAGE,
          ...(lastPageTimestampRef.current && {
            since: lastPageTimestampRef.current,
          }),
          ...(parsedQuery.level && {
            levels: [parsedQuery.level.toUpperCase() as V1LogLineLevel],
          }),
          ...(parsedQuery.search && { search: parsedQuery.search }),
          ...(parsedQuery.attempt && { attempt: parsedQuery.attempt }),
        };

        const response = await api.v1LogLineList(taskRun.metadata.id, params);
        const newRows = response.data.rows;

        if (newRows && newRows.length > 0) {
          lastPageTimestampRef.current = newRows[0]?.createdAt;

          queryClient.setQueryData<InfiniteData<V1LogLineList>>(
            [
              'v1Tasks',
              'getLogs',
              taskRun.metadata.id,
              parsedQuery.level,
              parsedQuery.search,
              parsedQuery.attempt,
            ],
            (oldData) => {
              if (!oldData) {
                return oldData;
              }
              const firstPage = oldData.pages[0];
              const existingRows = firstPage?.rows || [];

              const existingKeys = new Set(
                existingRows.map((row) => getLogLineKey(row)),
              );

              const uniqueNewRows = newRows.filter(
                (row) => !existingKeys.has(getLogLineKey(row)),
              );

              const updatedFirstPage = {
                ...firstPage,
                rows: [...uniqueNewRows, ...existingRows],
              };
              return {
                ...oldData,
                pages: [updatedFirstPage, ...oldData.pages.slice(1)],
              };
            },
          );
        }
      } catch (error) {
        console.error('Failed to poll for new logs:', error);
      } finally {
        isPollingRef.current = false;
        // Schedule next poll using setTimeout chaining
        timeoutIdRef.current = setTimeout(pollForNewLogs, 1000);
      }
    };

    pollForNewLogs();

    return () => {
      if (timeoutIdRef.current) {
        clearTimeout(timeoutIdRef.current);
      }
      // Reset polling state to ensure clean teardown
      // Note: If a poll is in-flight, it will also set this to false when complete
      isPollingRef.current = false;
    };
  }, [
    isTaskRunning,
    isPollingEnabled,
    taskRun?.metadata.id,
    parsedQuery.level,
    parsedQuery.search,
    parsedQuery.attempt,
    queryClient,
  ]);

  const logs = useMemo((): LogLine[] => {
    if (!getLogsQuery.data?.pages) {
      return [];
    }

    const uniqueLogsMap = new Map<string, LogLine>();

    getLogsQuery.data.pages.forEach((page) => {
      page?.rows?.forEach((row) => {
        const key = getLogLineKey(row);
        if (!uniqueLogsMap.has(key)) {
          uniqueLogsMap.set(key, {
            timestamp: row.createdAt,
            line: row.message,
            instance: taskRun?.displayName,
            level: row.level,
            metadata: row.metadata as Record<string, unknown> | undefined,
            attempt: row.attempt,
          });
        }
      });
    });

    return Array.from(uniqueLogsMap.values());
  }, [getLogsQuery.data?.pages, taskRun?.displayName]);

  const availableAttempts = useMemo(
    () =>
      Array.from({ length: (taskRun?.retryCount ?? 0) + 1 }, (_, i) => i + 1),
    [taskRun?.retryCount],
  );

  const fetchOlderLogs = useCallback(() => {
    if (!getLogsQuery.isFetchingNextPage && getLogsQuery.hasNextPage) {
      getLogsQuery.fetchNextPage();
    }
  }, [getLogsQuery]);

  return {
    logs,
    isLoading: getLogsQuery.isLoading,
    isFetchingMore: getLogsQuery.isFetchingNextPage,
    queryString,
    setQueryString,
    parsedQuery,
    fetchOlderLogs,
    setPollingEnabled,
    taskStatus: taskRun?.status,
    availableAttempts,
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
