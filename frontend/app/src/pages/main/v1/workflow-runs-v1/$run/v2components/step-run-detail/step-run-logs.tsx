import { V1TaskStatus, V1TaskSummary, V1LogLineList } from '@/lib/api';
import api from '@/lib/api/api';
import {
  useInfiniteQuery,
  InfiniteData,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useMemo, useCallback, useEffect } from 'react';

const LOGS_PER_PAGE = 100;

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
  const queryClient = useQueryClient();
  const getLogsQuery = useInfiniteQuery<
    V1LogLineList,
    Error,
    InfiniteData<V1LogLineList>,
    string[],
    string | undefined
  >({
    queryKey: ['v1Tasks', 'getLogs', taskRun?.metadata.id],
    queryFn: async ({ pageParam }) => {
      const params: any = {
        limit: LOGS_PER_PAGE,
        ...(pageParam && { since: pageParam }),
      };

      const response = await api.v1LogLineList(
        taskRun?.metadata.id || '',
        params,
      );
      return response.data;
    },
    initialPageParam: undefined,
    enabled: !!taskRun,
    getNextPageParam: (lastPage) => {
      const hasMore = lastPage?.pagination?.next_page !== undefined;
      const rows = lastPage?.rows;

      if (hasMore && rows && rows.length > 0 && rows.length === LOGS_PER_PAGE) {
        const lastLog = rows[rows.length - 1];
        return lastLog.createdAt;
      }

      return undefined;
    },
  });

  //NOTE: Custom polling logic for log refresh, instead of using inefficient infinite query refetch
  const { data: newLogsData } = useQuery({
    queryKey: ['v1Tasks', 'pollLogs', taskRun?.metadata.id],
    queryFn: async () => {
      const currentData = getLogsQuery.data;
      let lastTimestamp: string | undefined;

      if (currentData?.pages && currentData.pages.length > 0) {
        const lastPage = currentData.pages[currentData.pages.length - 1];
        const lastLog = lastPage?.rows?.[lastPage.rows.length - 1];
        lastTimestamp = lastLog?.createdAt;
      }

      const params: any = {
        limit: LOGS_PER_PAGE,
      };

      if (lastTimestamp) {
        params.since = lastTimestamp;
      }

      const response = await api.v1LogLineList(
        taskRun?.metadata.id || '',
        params,
      );
      return { data: response.data, lastTimestamp };
    },
    enabled: !!taskRun,
    refetchInterval: taskRun?.status === V1TaskStatus.RUNNING ? 1000 : 5000,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (!newLogsData?.data?.rows || newLogsData.data.rows.length === 0) return;

    queryClient.setQueryData(
      ['v1Tasks', 'getLogs', taskRun?.metadata.id],
      (old: InfiniteData<V1LogLineList> | undefined) => {
        if (!old || !old.pages.length) return old;

        const existingLogs = new Set(
          old.pages.flatMap(
            (page) =>
              page?.rows?.map((row) => `${row.createdAt}-${row.message}`) || [],
          ),
        );

        const newUniqueRows = (newLogsData.data.rows || []).filter((row) => {
          const logSignature = `${row.createdAt}-${row.message}`;
          return !existingLogs.has(logSignature);
        });

        if (newUniqueRows.length === 0) return old;

        const updatedPages = [...old.pages];
        const lastPageIndex = updatedPages.length - 1;

        updatedPages[lastPageIndex] = {
          ...updatedPages[lastPageIndex],
          rows: [...(updatedPages[lastPageIndex].rows || []), ...newUniqueRows],
        };

        return {
          ...old,
          pages: updatedPages,
          pageParams: old.pageParams,
        };
      },
    );
  }, [newLogsData, queryClient, taskRun?.metadata.id]);

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

  const handleBottomReached = useCallback(() => {
    if (getLogsQuery.hasNextPage && !getLogsQuery.isFetchingNextPage) {
      getLogsQuery.fetchNextPage();
    }
  }, [getLogsQuery]);

  return (
    <div className="my-4">
      <LoggingComponent
        logs={allLogs}
        onTopReached={() => {}}
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
