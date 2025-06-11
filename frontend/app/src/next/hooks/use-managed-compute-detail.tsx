import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import {
  LogLineList,
  ManagedWorker,
  Matrix,
} from '@/lib/api/generated/cloud/data-contracts';
import { queries } from '@/lib/api';
import { ListCloudLogsQuery, GetCloudMetricsQuery } from '@/lib/api/queries';
import { subDays } from 'date-fns';
import { useToast } from '@/next/hooks/utils/use-toast';

interface ManagedComputeDetailState {
  data: ManagedWorker | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<UseQueryResult<ManagedWorker | undefined, Error>>;
  logs: UseQueryResult<LogLineList, Error> | undefined;
  activity: UseQueryResult<LogLineList, Error> | undefined;
  instances: UseQueryResult<any, Error> | undefined;
  events: UseQueryResult<any, Error> | undefined;
  metrics: {
    cpu: UseQueryResult<Matrix, Error> | undefined;
    memory: UseQueryResult<Matrix, Error> | undefined;
    disk: UseQueryResult<Matrix, Error> | undefined;
  };
  setLogsQuery: (query: ListCloudLogsQuery) => void;
  setMetricsQuery: (query: GetCloudMetricsQuery) => void;
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
  initialLogsQuery?: ListCloudLogsQuery;
}

export function ManagedComputeDetailProvider({
  children,
  managedWorkerId,
  defaultRefetchInterval,
  initialLogsQuery,
}: ManagedComputeDetailProviderProps) {
  const { toast } = useToast();

  const [logsQuery, setLogsQuery] = useState<ListCloudLogsQuery>(
    initialLogsQuery || {
      before: new Date().toISOString(),
      after: subDays(new Date(), 1).toISOString(),
      direction: 'backward',
      search: '',
    },
  );

  const [metricsQuery, setMetricsQuery] = useState<GetCloudMetricsQuery>({
    after: subDays(new Date(), 1).toISOString(),
    before: new Date().toISOString(),
  });

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
    queryKey: ['managed-worker:logs', managedWorkerId, logsQuery],
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
    queryKey: ['managed-worker:get:cpu-metrics', managedWorkerId, metricsQuery],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsCpuGet(managedWorkerId, metricsQuery);
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
    queryKey: [
      'managed-worker:get:memory-metrics',
      managedWorkerId,
      metricsQuery,
    ],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsMemoryGet(
          managedWorkerId,
          metricsQuery,
        );
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
    queryKey: [
      'managed-worker:get:disk-metrics',
      managedWorkerId,
      metricsQuery,
    ],
    queryFn: async () => {
      try {
        const res = await cloudApi.metricsDiskGet(
          managedWorkerId,
          metricsQuery,
        );
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

  const managedWorkerInstancesQuery = useQuery({
    ...queries.cloud.listManagedWorkerInstances(managedWorkerId),
    enabled: !!managedWorkerId,
    refetchInterval: defaultRefetchInterval,
  });

  const managedWorkerEventsQuery = useQuery({
    ...queries.cloud.listManagedWorkerEvents(managedWorkerId),
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
        instances: managedWorkerInstancesQuery,
        events: managedWorkerEventsQuery,
        metrics: {
          cpu: getCpuMetricsQuery,
          memory: getMemoryMetricsQuery,
          disk: getDiskMetricsQuery,
        },
        setLogsQuery,
        setMetricsQuery,
      }) as ManagedComputeDetailState,
    [
      managedWorkerQuery,
      managedWorkerLogsQuery,
      managedWorkerInstancesQuery,
      managedWorkerEventsQuery,
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
