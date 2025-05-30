import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import api, { V1TaskTimingList } from '@/lib/api';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { V1TaskSummary, V1WorkflowRunDetails, V1TaskEvent } from '@/lib/api';
import { useToast } from './utils/use-toast';

interface RunDetailState {
  data: V1WorkflowRunDetails | null | undefined;
  parentData: V1WorkflowRunDetails | null | undefined;
  isLoading: boolean;
  error: Error | null;
  timings: UseQueryResult<V1TaskTimingList | null | undefined, Error> & {
    depth: number;
    setDepth: (depth: number) => void;
  };
  activity:
    | {
        events: V1TaskEvent[];
        logs: Array<{
          taskId: string;
          retryCount: number;
          attempt: number;
          createdAt: string;
          message: string;
          metadata: object;
        }>;
      }
    | null
    | undefined;
  cancel: {
    mutateAsync: (params: { tasks: V1TaskSummary[] }) => Promise<unknown>;
    isPending: boolean;
  };
  replay: {
    mutateAsync: (params: { tasks: V1TaskSummary[] }) => Promise<unknown>;
    isPending: boolean;
  };
  refetch: () => Promise<
    UseQueryResult<V1WorkflowRunDetails | null | undefined, Error>
  >;
  lastRefetchTime: number;
  refetchInterval: number | undefined;
}

const RunDetailContext = createContext<RunDetailState | null>(null);

export function useRunDetail() {
  const context = useContext(RunDetailContext);
  if (!context) {
    throw new Error('useRunDetail must be used within a RunDetailProvider');
  }
  return context;
}

interface RunDetailProviderProps {
  children: React.ReactNode;
  runId: string;
  defaultRefetchInterval?: number;
}

export function RunDetailProvider({
  children,
  runId,
  defaultRefetchInterval,
}: RunDetailProviderProps) {
  return (
    <RunsProvider>
      <RunDetailProviderContent
        runId={runId}
        defaultRefetchInterval={defaultRefetchInterval}
      >
        {children}
      </RunDetailProviderContent>
    </RunsProvider>
  );
}

function RunDetailProviderContent({
  children,
  runId,
  defaultRefetchInterval,
}: RunDetailProviderProps) {
  const { cancel: cancelRun, replay: replayRun } = useRuns();
  const [refetchInterval] = useState(defaultRefetchInterval);
  const [depth, setDepth] = useState(2);
  const [lastRefetchTime, setLastRefetchTime] = useState(Date.now());
  const { toast } = useToast();

  const runDetails = useQuery({
    queryKey: ['workflow-run-details:get', runId],
    queryFn: async () => {
      if (!runId || runId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }
      try {
        const run = (await api.v1WorkflowRunGet(runId)).data;
        setLastRefetchTime(Date.now());
        return run;
      } catch (error) {
        toast({
          title: 'Error fetching run details',
          variant: 'destructive',
          error,
        });
        return null;
      }
    },
    refetchInterval,
  });


  const parentDetails = useQuery({
    queryKey: [
      'workflow-run-details:get',
      runDetails.data?.run.parentTaskExternalId,
    ],
    queryFn: async () => {
      if (
        !runDetails.data?.run.parentTaskExternalId ||
        runDetails.data?.run.parentTaskExternalId ===
          '00000000-0000-0000-0000-000000000000'
      ) {
        return null;
      }
      try {
        const run = (
          await api.v1WorkflowRunGet(runDetails.data?.run.parentTaskExternalId)
        ).data;
        return run;
      } catch (error) {
        toast({
          title: 'Error fetching parent run details',
          variant: 'destructive',
          error,
        });
        return null;
      }
    },
    refetchInterval,
    enabled: !!runDetails.data?.run.parentTaskExternalId, // Only fetch if we have a parent task ID
  });

  // Memoize tasks array to prevent unnecessary activity fetches
  const tasks = useMemo(
    () => runDetails.data?.tasks || [],
    [runDetails.data?.tasks],
  );

  const activity = useQuery({
    queryKey: ['task-events:get', runId, tasks],
    queryFn: async () => {
      if (runId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }

      try {
        // FIXME: this is potentially problematic and we should have a single unified endpoint for this.
        // Batch API calls for better performance
        const [logs, events] = await Promise.all([
          Promise.all(
            tasks.map((task) =>
              api.v1LogLineList(task.metadata.id).then((response) =>
                (response.data?.rows || []).map((log) => ({
                  ...log,
                  taskId: task.metadata.id,
                  retryCount: log.retryCount || 0,
                  attempt: log.attempt || 1,
                })),
              ),
            ),
          ),
          Promise.all(
            tasks.map((task) =>
              api
                .v1TaskEventList(task.metadata.id, {
                  limit: 50,
                  offset: 0,
                })
                .then((response) => response.data?.rows || []),
            ),
          ),
        ]);

        return {
          events: events.flat(),
          logs: logs.flat(),
        };
      } catch (error) {
        toast({
          title: 'Error fetching run activity',
          variant: 'destructive',
          error,
        });
        return { events: [], logs: [] };
      }
    },
    refetchInterval,
    enabled: tasks.length > 0, // Only fetch if we have tasks
  });

  const taskTimings = useQuery({
    queryKey: ['task-events:timings', runId, depth],
    queryFn: async () =>
      (
        await api.v1WorkflowRunGetTimings(runId, {
          depth,
        })
      ).data,
    refetchInterval,
    enabled: !!runId,
  });

  const value = useMemo(
    () => ({
      data: runDetails.data,
      parentData: parentDetails.data,
      isLoading: runDetails.isLoading,
      error: runDetails.error,
      timings: { ...taskTimings, depth, setDepth },
      activity: activity.data,
      cancel: {
        mutateAsync: cancelRun.mutateAsync,
        isPending: cancelRun.isPending,
      },
      replay: {
        mutateAsync: replayRun.mutateAsync,
        isPending: replayRun.isPending,
      },
      refetch: runDetails.refetch,
      lastRefetchTime,
      refetchInterval,
    }),
    [
      runDetails.data,
      runDetails.isLoading,
      runDetails.error,
      runDetails.refetch,
      parentDetails.data,
      activity.data,
      taskTimings,
      depth,
      cancelRun.mutateAsync,
      cancelRun.isPending,
      replayRun.mutateAsync,
      replayRun.isPending,
      lastRefetchTime,
      refetchInterval,
    ],
  );

  return (
    <RunDetailContext.Provider value={value}>
      {children}
    </RunDetailContext.Provider>
  );
}
