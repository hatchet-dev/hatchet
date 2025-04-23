import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import {
  LogLineList,
  ManagedWorker,
  Matrix,
} from '@/lib/api/generated/cloud/data-contracts';
import { ListCloudLogsQuery } from '@/lib/api/queries';
import { subDays } from 'date-fns';
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
  const [refetchInterval, setRefetchInterval] = useState(
    defaultRefetchInterval,
  );

  const managedWorkerQuery = useQuery({
    queryKey: ['managed-worker:get', managedWorkerId],
    queryFn: async () => {
      const res = await cloudApi.managedWorkerGet(managedWorkerId);
      return res.data;
    },
    refetchInterval,
  });

  const logsQuery: ListCloudLogsQuery = {
    before: subDays(new Date(), 1).toISOString(),
    after: new Date().toISOString(),
    direction: 'backward',
    search: '',
  };

  const managedWorkerLogsQuery = useQuery({
    queryKey: ['managed-worker:logs', managedWorkerId],
    queryFn: async () => {
      const res = await cloudApi.logList(managedWorkerId, logsQuery);
      return res.data;
    },
    refetchInterval,
  });

  const getCpuMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:cpu-metrics', managedWorkerId, logsQuery],
    queryFn: async () =>
      (await cloudApi.metricsCpuGet(managedWorkerId, logsQuery)).data,
    enabled: !!managedWorkerId,
    refetchInterval: refetchInterval,
  });

  const getMemoryMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:memory-metrics', managedWorkerId, logsQuery],
    queryFn: async () =>
      (await cloudApi.metricsMemoryGet(managedWorkerId, logsQuery)).data,
    enabled: !!managedWorkerId,
    refetchInterval: refetchInterval,
  });

  const getDiskMetricsQuery = useQuery({
    queryKey: ['managed-worker:get:disk-metrics', managedWorkerId, logsQuery],
    queryFn: async () =>
      (await cloudApi.metricsDiskGet(managedWorkerId, logsQuery)).data,
    enabled: !!managedWorkerId,
    refetchInterval: refetchInterval,
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
