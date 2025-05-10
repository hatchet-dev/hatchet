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
      if (runId === '00000000-0000-0000-0000-000000000000') {
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
    refetchInterval: refetchInterval,
  });

  const parentDetails = useQuery({
    queryKey: [
      'workflow-run-details:get',
      runDetails.data?.run.parentTaskExternalId,
    ],
    queryFn: async () => {
      const runId = runDetails.data?.run.parentTaskExternalId;
      if (!runId || runId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }
      try {
        const run = (await api.v1WorkflowRunGet(runId)).data;
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
    refetchInterval: refetchInterval,
  });

  const activity = useQuery({
    queryKey: ['task-events:get', runId, runDetails.data?.tasks],
    queryFn: async () => {
      if (runId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }

      try {
        const tasks = runDetails.data?.tasks || [];

        const logPromises = tasks.map(async (task) =>
          ((await api.v1LogLineList(task.metadata.id)).data?.rows || []).map(
            (log) => ({
              ...log,
              taskId: task.metadata.id,
              retryCount: log.retryCount || 0,
              attempt: log.attempt || 0,
            }),
          ),
        );

        const eventPromises = tasks.map(
          async (task) =>
            (
              await api.v1TaskEventList(task.metadata.id, {
                limit: 50,
                offset: 0,
              })
            ).data?.rows || [],
        );

        const [logs, events] = await Promise.all([
          Promise.all(logPromises),
          Promise.all(eventPromises),
        ]);

        const mergedLogs = logs.flat();
        const mergedEvents = events.flat();

        return { events: mergedEvents, logs: mergedLogs };
      } catch (error) {
        toast({
          title: 'Error fetching run activity',
          variant: 'destructive',
          error,
        });
        return { events: [], logs: [] };
      }
    },
    refetchInterval: refetchInterval,
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
      runDetails,
      activity.data,
      cancelRun,
      replayRun,
      parentDetails,
      lastRefetchTime,
      refetchInterval,
      taskTimings,
      depth,
      setDepth,
    ],
  );

  return (
    <RunDetailContext.Provider value={value}>
      {children}
    </RunDetailContext.Provider>
  );
}
