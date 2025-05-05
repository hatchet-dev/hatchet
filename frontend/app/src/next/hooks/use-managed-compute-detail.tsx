import { createContext, useContext, useMemo } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import {
  LogLineList,
  ManagedWorker,
  Matrix,
} from '@/lib/api/generated/cloud/data-contracts';
import { ListCloudLogsQuery } from '@/lib/api/queries';
import { subDays } from 'date-fns';
import { useToast } from '@/next/hooks/utils/use-toast';

interface ManagedComputeDetailState {
  data: ManagedWorker | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<UseQueryResult<ManagedWorker | undefined, Error>>;
  logs: UseQueryResult<LogLineList, Error> | undefined;
  activity: UseQueryResult<LogLineList, Error> | undefined;
  metrics: {
    cpu: UseQueryResult<Matrix, Error> | undefined;
    memory: UseQueryResult<Matrix, Error> | undefined;
    disk: UseQueryResult<Matrix, Error> | undefined;
  };
}

const ManagedComputeDetailContext =
  createContext<ManagedComputeDetailState | null>(null);

export function useManagedComputeDetail() {
  const context = useContext(ManagedComputeDetailContext);
  if (!context) {
    throw new Error(
      'useManagedComputeDetail must be used within a ManagedComputeDetailProvider',
    );
  }
  return context;
}

interface ManagedComputeDetailProviderProps {
  children: React.ReactNode;
  managedWorkerId: string;
  defaultRefetchInterval?: number;
}

export function ManagedComputeDetailProvider({
  children,
  managedWorkerId,
  defaultRefetchInterval,
}: ManagedComputeDetailProviderProps) {
  const { toast } = useToast();

  const logsQuery = useMemo<ListCloudLogsQuery>(
    () => ({
      before: subDays(new Date(), 1).toISOString(),
      after: new Date().toISOString(),
      direction: 'backward',
      search: '',
    }),
    [],
  );

  const managedWorkerQuery = useQuery({
    queryKey: ['managed-worker:get', managedWorkerId],
    queryFn: async () => {
      try {
        const res = await cloudApi.managedWorkerGet(managedWorkerId);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching managed worker',

          variant: 'destructive',
          error,
        });
      }
    },
    refetchInterval: defaultRefetchInterval,
  });

  const managedWorkerLogsQuery = useQuery({
    queryKey: ['managed-worker:logs', managedWorkerId],
    queryFn: async () => {
      try {
        const res = await cloudApi.logList(managedWorkerId, logsQuery);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching managed worker logs',

          variant: 'destructive',
          error,
        });
      }
    },
    refetchInterval: defaultRefetchInterval,
  });

  const getCpuMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:cpu-metrics', managedWorkerId, logsQuery],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsCpuGet(managedWorkerId, logsQuery);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching managed worker CPU metrics',

          variant: 'destructive',
          error,
        });
      }
    },
    enabled: !!managedWorkerId,
    refetchInterval: defaultRefetchInterval,
  });

  const getMemoryMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:memory-metrics', managedWorkerId, logsQuery],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsMemoryGet(managedWorkerId, logsQuery);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching managed worker memory metrics',

          variant: 'destructive',
          error,
        });
      }
    },
    enabled: !!managedWorkerId,
    refetchInterval: defaultRefetchInterval,
  });

  const getDiskMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:disk-metrics', managedWorkerId, logsQuery],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsDiskGet(managedWorkerId, logsQuery);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching managed worker disk metrics',

          variant: 'destructive',
          error,
        });
      }
    },
    enabled: !!managedWorkerId,
    refetchInterval: defaultRefetchInterval,
  });

  const value = useMemo(
    () =>
      ({
        data: managedWorkerQuery.data,
        isLoading: managedWorkerQuery.isLoading,
        error: managedWorkerQuery.error,
        refetch: managedWorkerQuery.refetch,
        logs: managedWorkerLogsQuery,
        metrics: {
          cpu: getCpuMetricsQuery,
          memory: getMemoryMetricsQuery,
          disk: getDiskMetricsQuery,
        },
      }) as ManagedComputeDetailState,
    [
      managedWorkerQuery,
      managedWorkerLogsQuery,
      getCpuMetricsQuery,
      getMemoryMetricsQuery,
      getDiskMetricsQuery,
    ],
  );

  return (
    <ManagedComputeDetailContext.Provider value={value}>
      {children}
    </ManagedComputeDetailContext.Provider>
  );
}
