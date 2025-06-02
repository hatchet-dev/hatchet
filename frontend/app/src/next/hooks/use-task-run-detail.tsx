import {
  createContext,
  useContext,
  useMemo,
  useState,
  useCallback,
  useRef,
} from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import api from '@/lib/api';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { V1TaskSummary } from '@/lib/api';
import { useToast } from './utils/use-toast';
import { AxiosError } from 'axios';

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
  attempt?: number;
}

const TaskRunDetailContext = createContext<TaskRunDetailState | null>(null);

export function useTaskRunDetail() {
  const context = useContext(TaskRunDetailContext);
  if (!context) {
    throw new Error(
      'useTaskRunDetail must be used within a TaskRunDetailProvider',
    );
  }
  return context;
}

interface TaskRunDetailProviderProps {
  children: React.ReactNode;
  taskRunId?: string;
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
  const lastRefetchTimeRef = useRef(Date.now());
  const { toast } = useToast();

  // Memoize query key to prevent unnecessary refetches
  const queryKey = useMemo(
    () => ['task-run-details:get', taskRunId, attempt],
    [taskRunId, attempt],
  );

  // Memoize error handler to prevent unnecessary re-renders
  const handleError = useCallback(
    (error: unknown) => {
      if (error instanceof AxiosError) {
        // Handle 404s gracefully for DAG runs
        if (error.response?.status === 404) {
          return null;
        }

        // Handle other API errors
        toast({
          title: 'Error fetching task run details',
          variant: 'destructive',
          error,
        });
      } else {
        // Handle unexpected errors
        toast({
          title: 'Unexpected error',
          variant: 'destructive',
          error,
        });
      }
      return null;
    },
    [toast],
  );

  const runDetails = useQuery<V1TaskSummary | null>({
    queryKey,
    queryFn: async () => {
      // Early return for invalid task IDs
      if (!taskRunId || taskRunId === '00000000-0000-0000-0000-000000000000') {
        return null;
      }

      try {
        const run = (await api.v1TaskGet(taskRunId, { attempt })).data;
        lastRefetchTimeRef.current = Date.now();
        return run;
      } catch (error) {
        return handleError(error);
      }
    },
    refetchInterval,
    retry: (failureCount, error) => {
      // Don't retry on 404s
      if (error instanceof AxiosError && error.response?.status === 404) {
        return false;
      }
      // Retry up to 2 times for other errors
      return failureCount < 2;
    },
    enabled: !!taskRunId,
  });

  // Memoize cancel and replay functions to prevent unnecessary re-renders
  const cancel = useMemo(
    () => ({
      mutateAsync: cancelRun.mutateAsync,
      isPending: cancelRun.isPending,
    }),
    [cancelRun.mutateAsync, cancelRun.isPending],
  );

  const replay = useMemo(
    () => ({
      mutateAsync: replayRun.mutateAsync,
      isPending: replayRun.isPending,
    }),
    [replayRun.mutateAsync, replayRun.isPending],
  );

  const value = useMemo<TaskRunDetailState>(
    () => ({
      data: runDetails.data,
      isLoading: runDetails.isLoading,
      error: runDetails.error,
      cancel,
      replay,
      refetch: runDetails.refetch,
      lastRefetchTime: lastRefetchTimeRef.current,
      refetchInterval,
      attempt,
    }),
    [
      runDetails.data,
      runDetails.isLoading,
      runDetails.error,
      runDetails.refetch,
      cancel,
      replay,
      refetchInterval,
      attempt,
    ],
  );

  return (
    <TaskRunDetailContext.Provider value={value}>
      {children}
    </TaskRunDetailContext.Provider>
  );
}
