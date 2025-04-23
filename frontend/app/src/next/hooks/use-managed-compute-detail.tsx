import { createContext, useContext, useMemo, useState } from 'react';
import { useQuery, UseQueryResult } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { ManagedWorker } from '@/lib/api/generated/cloud/data-contracts';

interface ManagedComputeDetailState {
  data: ManagedWorker | undefined;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<UseQueryResult<ManagedWorker | undefined, Error>>;
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

  const managedWorkerMetricsQuery = useQuery({
    queryKey: ['managed-worker:metrics', managedWorkerId],
    queryFn: async () => {
      const res = await cloudApi.managedWorkerMetrics(managedWorkerId);
      return res.data;
    },
    refetchInterval,
  });

  const value = useMemo(
    () => ({
      data: managedWorkerQuery.data,
      isLoading: managedWorkerQuery.isLoading,
      error: managedWorkerQuery.error,
      refetch: managedWorkerQuery.refetch,
    }),
    [managedWorkerQuery],
  );

  return (
    <ManagedComputeDetailContext.Provider value={value}>
      {children}
    </ManagedComputeDetailContext.Provider>
  );
}
