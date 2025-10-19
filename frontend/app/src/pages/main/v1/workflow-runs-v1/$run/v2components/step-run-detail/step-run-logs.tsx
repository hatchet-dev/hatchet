import { V1TaskSummary, V1LogLineList, V1TaskStatus } from '@/lib/api';
import { V1LogLineListQuery } from '@/lib/api/queries';
import api from '@/lib/api/api';
import { useInfiniteQuery, InfiniteData } from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useMemo, useCallback, useRef } from 'react';

const LOGS_PER_PAGE = 100;
const FETCH_THRESHOLD_DOWN = 0.5;
const FETCH_THRESHOLD_UP = 0.4;
const MAX_PAGES = 2;

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
  interface PageBoundary {
    rowFirstTimestamp: string;
    rowLastTimestamp: string;
    fetchedNext?: boolean;
    fetchedPrevious?: boolean;
  }
  const pageBoundariesRef = useRef<Record<number, PageBoundary>>({});
  const currentPageNumberRef = useRef<number>(0);
  const isRefetchingRef = useRef<boolean>(false);
  const lastScrollTopRef = useRef<number>(0);
  const lastScrollPercentageRef = useRef<number>(0);
  const lastPageNumberRef = useRef<number>(0);

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

      if (pageParam.since == null && pageParam.until == null) {
        currentPageNumberRef.current = 1;
      } else if (
        pageParam.since != null &&
        pageParam.until == null &&
        !isRefetchingRef.current
      ) {
        currentPageNumberRef.current += 1;
      } else if (!isRefetchingRef.current) {
        currentPageNumberRef.current = Math.max(
          1,
          currentPageNumberRef.current - 1,
        );
      }

      if (!pageBoundariesRef.current[currentPageNumberRef.current]) {
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
    refetchInterval: taskRun?.status === V1TaskStatus.RUNNING ? 1000 : false,
    staleTime: taskRun?.status === V1TaskStatus.RUNNING ? 1000 : Infinity,
    getPreviousPageParam: (firstPage) => {
      if (currentPageNumberRef.current <= 2) {
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
      if (rows && rows.length > 0 && rows.length === LOGS_PER_PAGE) {
        const lastLog = rows?.[rows.length - 1];
        return { since: lastLog?.createdAt, until: undefined };
      }
      return undefined;
    },
  });

  isRefetchingRef.current = getLogsQuery.isRefetching;

  const allLogs = useMemo(() => {
    if (!getLogsQuery.data?.pages) return [];

    return getLogsQuery.data.pages.flatMap(
      (page) =>
        page?.rows?.map((row: any) => ({
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
        isRefetchingRef.current
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
    [getLogsQuery],
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
