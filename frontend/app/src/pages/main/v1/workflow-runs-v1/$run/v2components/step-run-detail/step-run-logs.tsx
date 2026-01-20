import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { V1TaskSummary, V1LogLineList, V1TaskStatus } from '@/lib/api';
import api from '@/lib/api/api';
import { V1LogLineListQuery } from '@/lib/api/queries';
import {
  useInfiniteQuery,
  InfiniteData,
  useQueryClient,
} from '@tanstack/react-query';
import { useMemo, useCallback, useRef, useEffect } from 'react';

const LOGS_PER_PAGE = 100;
const FETCH_THRESHOLD_DOWN = 0.7;
const FETCH_THRESHOLD_UP = 0.4;

interface PageBoundary {
  rowFirstTimestamp: string;
  rowLastTimestamp: string;
  fetchedNext?: boolean;
  fetchedPrevious?: boolean;
  currentPageText?: string;
}

export function StepRunLogs({
  taskRun,
  resetTrigger,
}: {
  taskRun: V1TaskSummary;
  resetTrigger?: number;
}) {
  const queryClient = useQueryClient();
  const pageBoundariesRef = useRef<Record<number, PageBoundary>>({});
  const lastPageTimestampRef = useRef<string | undefined>(undefined);
  const currentPageNumberRef = useRef(0);
  const isRefetchingRef = useRef(false);
  const lastScrollTopRef = useRef(0);
  const lastScrollPercentageRef = useRef(0);
  const lastPageNumberRef = useRef(0);

  const isTaskRunning = taskRun?.status === V1TaskStatus.RUNNING;

  // NOTE: Old pages are retained when a task is running, 0 = unlimited
  const MAX_PAGES = isTaskRunning ? 0 : 2;

  // Reset state (when logs tab is reopened)
  useEffect(() => {
    if (resetTrigger !== undefined && resetTrigger > 0) {
      queryClient.resetQueries({
        queryKey: ['v1Tasks', 'getLogs', taskRun?.metadata.id],
        exact: true,
      });
    }
  }, [resetTrigger, queryClient, taskRun?.metadata.id]);

  const getLogsQuery = useInfiniteQuery<
    V1LogLineList,
    Error,
    InfiniteData<V1LogLineList>,
    string[],
    { since: string | undefined; until: string | undefined }
  >({
    queryKey: ['v1Tasks', 'getLogs', taskRun?.metadata.id],
    queryFn: async ({ pageParam }) => {
      const params: V1LogLineListQuery = {
        limit: LOGS_PER_PAGE,
        ...(pageParam && { since: pageParam.since, until: pageParam.until }),
      };

      const response = await api.v1LogLineList(
        taskRun?.metadata.id || '',
        params,
      );
      const rows = response.data.rows;

      const maxCreatedAt = rows?.[rows.length - 1]?.createdAt;

      if (isTaskRunning && maxCreatedAt) {
        lastPageTimestampRef.current = maxCreatedAt;
        return response.data;
      }

      if (pageParam.since == null && pageParam.until == null) {
        currentPageNumberRef.current = 1;
      } else if (
        !isRefetchingRef.current &&
        pageParam.since != null &&
        pageParam.until == null &&
        rows &&
        rows.length === LOGS_PER_PAGE
      ) {
        currentPageNumberRef.current += 1;
      } else if (!isRefetchingRef.current) {
        currentPageNumberRef.current = Math.max(
          MAX_PAGES,
          currentPageNumberRef.current - 1,
        );
      }

      if (
        !pageBoundariesRef.current[currentPageNumberRef.current]
          ?.rowLastTimestamp &&
        rows &&
        rows.length === LOGS_PER_PAGE
      ) {
        pageBoundariesRef.current[currentPageNumberRef.current] = {
          rowFirstTimestamp: rows?.[0]?.createdAt || '',
          rowLastTimestamp: rows?.[rows.length - 1]?.createdAt || '',
        };
      }

      pageBoundariesRef.current[currentPageNumberRef.current] = {
        ...pageBoundariesRef.current[currentPageNumberRef.current],
        fetchedNext: false,
        fetchedPrevious: false,
      };

      return response.data;
    },
    initialPageParam: { since: undefined, until: undefined },
    enabled: !!taskRun,
    maxPages: MAX_PAGES,
    refetchInterval: false,
    staleTime: Infinity,
    getPreviousPageParam: (firstPage) => {
      if (currentPageNumberRef.current <= MAX_PAGES) {
        return undefined;
      }
      const rows = firstPage?.rows;

      if (rows && rows.length > 0) {
        const sinceTsForPreviousPage =
          currentPageNumberRef.current > MAX_PAGES + 1
            ? pageBoundariesRef.current[
                currentPageNumberRef.current - (MAX_PAGES + 1)
              ].rowLastTimestamp
            : undefined;
        return {
          since: sinceTsForPreviousPage,
          until:
            pageBoundariesRef.current[currentPageNumberRef.current - 1]
              ?.rowFirstTimestamp,
        };
      }
      return undefined;
    },
    getNextPageParam: (lastPage) => {
      const rows = lastPage?.rows;
      if (!isTaskRunning && rows && rows.length === LOGS_PER_PAGE) {
        const lastLog = rows?.[rows.length - 1];
        return { since: lastLog?.createdAt, until: undefined };
      } else if (isTaskRunning) {
        return { since: lastPageTimestampRef.current, until: undefined };
      }
      return undefined;
    },
  });

  isRefetchingRef.current = getLogsQuery.isRefetching;

  //NOTE: Fetch logs every second while the task is running.
  useEffect(() => {
    if (!isTaskRunning) {
      return;
    }

    const interval = setInterval(() => {
      if (!getLogsQuery.isFetchingNextPage) {
        getLogsQuery.fetchNextPage();
      }
    }, 1000);

    return () => {
      if (interval) {
        clearInterval(interval);
      }
    };
  }, [isTaskRunning, getLogsQuery]);

  const allLogs = useMemo(() => {
    if (!getLogsQuery.data?.pages) {
      return [];
    }

    return getLogsQuery.data.pages.flatMap(
      (page) =>
        page?.rows?.map((row) => ({
          timestamp: row.createdAt,
          line: row.message,
          instance: taskRun.displayName,
        })) || [],
    );
  }, [getLogsQuery.data?.pages, taskRun.displayName]);

  const handleScroll = useCallback(
    (scrollData: {
      scrollTop: number;
      scrollHeight: number;
      clientHeight: number;
    }) => {
      const { scrollTop, scrollHeight, clientHeight } = scrollData;

      const scrollableHeight = scrollHeight - clientHeight;
      const scrollPercentage = scrollTop / scrollableHeight;

      const pageNumberChanged =
        currentPageNumberRef.current !== lastPageNumberRef.current;
      if (pageNumberChanged) {
        lastPageNumberRef.current = currentPageNumberRef.current;
        lastScrollTopRef.current = scrollTop;
        lastScrollPercentageRef.current = scrollTop / scrollableHeight;
        return;
      }

      if (
        scrollableHeight <= 0 ||
        getLogsQuery.isFetchingPreviousPage ||
        getLogsQuery.isFetchingNextPage ||
        isRefetchingRef.current ||
        isTaskRunning
      ) {
        return;
      }

      const scrollDirection =
        scrollTop > lastScrollTopRef.current ? 'down' : 'up';
      const isScrollingDown = scrollDirection === 'down';
      const isScrollingUp = scrollDirection === 'up';

      const crossedThresholdDown = scrollPercentage >= FETCH_THRESHOLD_DOWN;
      const crossedThresholdUp = scrollPercentage <= FETCH_THRESHOLD_UP;

      const currentPageBoundary =
        pageBoundariesRef.current[currentPageNumberRef.current];

      if (
        isScrollingUp &&
        crossedThresholdUp &&
        currentPageNumberRef.current > 1
      ) {
        if (currentPageBoundary && !currentPageBoundary.fetchedPrevious) {
          currentPageBoundary.fetchedPrevious = true;
          getLogsQuery.fetchPreviousPage();
        }
      }

      if (isScrollingDown && crossedThresholdDown) {
        if (currentPageBoundary && !currentPageBoundary.fetchedNext) {
          currentPageBoundary.fetchedNext = true;
          getLogsQuery.fetchNextPage();
        }
      }

      lastScrollTopRef.current = scrollTop;
      lastScrollPercentageRef.current = scrollPercentage;
    },
    [getLogsQuery, isTaskRunning],
  );

  return (
    <div className="my-4">
      <LoggingComponent
        logs={allLogs}
        onTopReached={() => {}}
        onBottomReached={() => {}}
        autoScroll={false}
        onInfiniteScroll={handleScroll}
      />
    </div>
  );
}
