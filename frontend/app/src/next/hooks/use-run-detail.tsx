import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import api from '@/lib/api';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { V1TaskSummary, V1WorkflowRunDetails, V1TaskEvent } from '@/lib/api';

interface RunDetailState {
  data: V1WorkflowRunDetails | undefined;
  parentData: V1WorkflowRunDetails | undefined;
  isLoading: boolean;
  error: Error | null;
  activity:
    | {
        events: V1TaskEvent[];
        logs: Array<{
          taskId: string;
          createdAt: string;
          message: string;
          metadata: object;
        }>;
      }
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
    UseQueryResult<V1WorkflowRunDetails | undefined, Error>
  >;
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
  const [refetchInterval, setRefetchInterval] = useState(
    defaultRefetchInterval,
  );

  const runDetails = useQuery({
    queryKey: ['workflow-run-details:get', runId],
    queryFn: async () => {
      if (runId === '00000000-0000-0000-0000-000000000000') {
        return;
      }
      const run = (await api.v1WorkflowRunGet(runId)).data;

      if (refetchInterval) {
        setRefetchInterval(
          run.run.status === 'RUNNING' ? 1000 : defaultRefetchInterval || 0,
        );
      }

      return run;
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
        return;
      }
      const run = (await api.v1WorkflowRunGet(runId)).data;

      if (refetchInterval) {
        setRefetchInterval(
          run.run.status === 'RUNNING' ? 1000 : defaultRefetchInterval || 0,
        );
      }

      return run;
    },
    refetchInterval: refetchInterval,
  });

  const activity = useQuery({
    queryKey: ['task-events:get', runId, runDetails.data?.tasks],
    queryFn: async () => {
      if (runId === '00000000-0000-0000-0000-000000000000') {
        return;
      }

      const tasks = runDetails.data?.tasks || [];

      const logPromises = tasks.map(async (task) =>
        ((await api.v1LogLineList(task.metadata.id)).data?.rows || []).map(
          (log) => ({
            ...log,
            taskId: task.metadata.id,
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
    },
    refetchInterval: refetchInterval,
  });

  const value = useMemo(
    () => ({
      data: runDetails.data,
      parentData: parentDetails.data,
      isLoading: runDetails.isLoading,
      error: runDetails.error,
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
    }),
    [runDetails, activity.data, cancelRun, replayRun, parentDetails],
  );

  return (
    <RunDetailContext.Provider value={value}>
      {children}
    </RunDetailContext.Provider>
  );
}
