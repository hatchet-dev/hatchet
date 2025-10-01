import { V1TaskSummary, V1LogLineList, V1TaskStatus } from '@/lib/api';
import { V1LogLineListQuery } from '@/lib/api/queries';
import api from '@/lib/api/api';
import {
  useInfiniteQuery,
  InfiniteData,
} from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useMemo, useCallback } from 'react';

const LOGS_PER_PAGE = 100;

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
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
      return response.data;
    },
    initialPageParam: { since: undefined, until: undefined },
    enabled: !!taskRun,
    maxPages: 2,
    refetchInterval: taskRun?.status === V1TaskStatus.RUNNING ? 1000 : 5000,
    getPreviousPageParam: (firstPage) => {
      const rows = firstPage?.rows;
      if (rows && rows.length > 0 && rows.length === LOGS_PER_PAGE) {
        const firstLog = rows?.[0];
        return { since: undefined, until: firstLog?.createdAt };
      }
      return { since: undefined, until: undefined };
    },
    getNextPageParam: (lastPage) => {
      const rows = lastPage?.rows;

      if (rows && rows.length > 0 && rows.length === LOGS_PER_PAGE) {
        const lastLog = rows?.[rows.length - 1];
        return { since: lastLog?.createdAt, until: undefined };
      }
      return { since: undefined, until: undefined };
    },
  });

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
      />
      {getLogsQuery.isFetchingNextPage && (
        <div className="flex justify-center py-4">
          <div className="text-sm text-gray-500">Loading more logs...</div>
        </div>
      )}
    </div>
  );
}
