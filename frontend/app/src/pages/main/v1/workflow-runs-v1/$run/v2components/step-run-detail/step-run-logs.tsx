import { V1TaskSummary, V1LogLineList, V1TaskStatus } from '@/lib/api';
import { V1LogLineListQuery } from '@/lib/api/queries';
import api from '@/lib/api/api';
import {
  useInfiniteQuery,
  InfiniteData,
} from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useMemo, useCallback, useRef } from 'react';

const LOGS_PER_PAGE = 100;

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
  interface PageBoundary {
    rowFirstTimestamp: string;
    rowLastTimestamp: string;
  }
  const pageBoundariesRef = useRef<Record<number, PageBoundary>>({});
  const currentPageNumberRef = useRef<number>(0);
  const isRefetchingRef = useRef<boolean>(false);
  const getLogsQuery = useInfiniteQuery<
  V1LogLineList,
  Error,
  InfiniteData<V1LogLineList>,
  string[],
  { since: string | undefined, until: string | undefined }
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
      } else if (pageParam.since != null && pageParam.until == null && !isRefetchingRef.current) {
        currentPageNumberRef.current += 1;
      } else if (!isRefetchingRef.current) {
        currentPageNumberRef.current = Math.max(1, currentPageNumberRef.current - 1);
      }

      pageBoundariesRef.current[currentPageNumberRef.current] = {
        "rowFirstTimestamp": rows?.[0]?.createdAt || '',
        "rowLastTimestamp": rows?.[rows.length - 1]?.createdAt || '',
      }

      return response.data;
    },
    initialPageParam: { since: undefined, until: undefined },
    enabled: !!taskRun,
    maxPages: 1,
    refetchInterval: taskRun?.status === V1TaskStatus.RUNNING ? 1000 : false,
    getPreviousPageParam: (firstPage) => {
      if (currentPageNumberRef.current === 1) {
        return undefined;
      }
      const rows = firstPage?.rows;

      if (rows && rows.length > 0) {
        const firstLog = rows?.[0];
        const sinceTsForPreviousPage = currentPageNumberRef.current > 2 ? pageBoundariesRef.current[currentPageNumberRef.current - 2].rowLastTimestamp : undefined;
        return { since: sinceTsForPreviousPage, until: firstLog?.createdAt };
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

  const handleTopReached = useCallback(() => {
    if (getLogsQuery.hasPreviousPage && !getLogsQuery.isFetchingPreviousPage) {
      getLogsQuery.fetchPreviousPage();
    }
  }, [getLogsQuery]);

  const handleBottomReached = useCallback(() => {
    if (getLogsQuery.hasNextPage && !getLogsQuery.isFetchingNextPage) {
      getLogsQuery.fetchNextPage();
    }
  }, [getLogsQuery]);

  return (
    <div className="my-4">
      <LoggingComponent
        logs={allLogs}
        onTopReached={handleTopReached}
        onBottomReached={handleBottomReached}
        autoScroll={false}
        isFetchingNextPage={getLogsQuery.isFetchingNextPage}
        isFetchingPreviousPage={getLogsQuery.isFetchingPreviousPage}
      />
      {getLogsQuery.isFetchingNextPage && (
        <div className="flex justify-center py-4">
          <div className="text-sm text-gray-500">Loading more logs...</div>
        </div>
      )}
    </div>
  );
}
