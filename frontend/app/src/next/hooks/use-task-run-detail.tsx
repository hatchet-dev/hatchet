import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import api from '@/lib/api';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { V1TaskSummary } from '@/lib/api';
import { useToast } from './utils/use-toast';

interface TaskRunDetailState {
  data: V1TaskSummary | null | undefined;
  isLoading: boolean;
  error: Error | null;
  cancel: {
    mutateAsync: (params: { tasks: V1TaskSummary[] }) => Promise<unknown>;
    isPending: boolean;
  };
  replay: {
    mutateAsync: (params: { tasks: V1TaskSummary[] }) => Promise<unknown>;
    isPending: boolean;
  };
  refetch: () => Promise<
    UseQueryResult<V1TaskSummary | null | undefined, Error>
  >;
  lastRefetchTime: number;
  refetchInterval: number | undefined;
}

const TaskRunDetailContext = createContext<TaskRunDetailState | null>(null);

export function useTaskRunDetail() {
  const context = useContext(TaskRunDetailContext);
  if (!context) {
    throw new Error('useTaskRunDetail must be used within a TaskRunDetailProvider');
  }
  return context;
}

interface TaskRunDetailProviderProps {
  children: React.ReactNode;
  taskRunId: string;
  attempt?: number;
  defaultRefetchInterval?: number;
}

export function TaskRunDetailProvider({
  children,
  taskRunId,
  attempt,
  defaultRefetchInterval,
}: TaskRunDetailProviderProps) {
  return (
    <RunsProvider>
      <TaskRunDetailProviderContent
        taskRunId={taskRunId}
        attempt={attempt}
        defaultRefetchInterval={defaultRefetchInterval}
      >
        {children}
      </TaskRunDetailProviderContent>
    </RunsProvider>
  );
}

function TaskRunDetailProviderContent({
  children,
  taskRunId,
  defaultRefetchInterval,
  attempt,
}: TaskRunDetailProviderProps) {
  const { cancel: cancelRun, replay: replayRun } = useRuns();
  const [refetchInterval] = useState(defaultRefetchInterval);
  const [lastRefetchTime, setLastRefetchTime] = useState(Date.now());
  const { toast } = useToast();

  const runDetails = useQuery({
    queryKey: ['task-run-details:get', taskRunId, attempt],
    queryFn: async () => {
      if (taskRunId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }
      try {
        const run = (await api.v1TaskGet(taskRunId, { attempt })).data;
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


  const value = useMemo(
    () => ({
      data: runDetails.data,
      isLoading: runDetails.isLoading,
      error: runDetails.error,
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
      cancelRun,
      replayRun,
      lastRefetchTime,
      refetchInterval,
    ],
  );

  return (
    <TaskRunDetailContext.Provider value={value}>
      {children}
    </TaskRunDetailContext.Provider>
  );
}
