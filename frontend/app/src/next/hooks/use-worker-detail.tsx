import api, { UpdateWorkerRequest, Worker } from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';

interface UpdateWorkerParams {
  workerId: string;
  data: UpdateWorkerRequest;
}

interface WorkerDetailState {
  data?: Worker;
  isLoading: boolean;
  update: ReturnType<
    typeof useMutation<Worker, Error, UpdateWorkerParams, unknown>
  >;
}

interface WorkerDetailProviderProps extends PropsWithChildren {
  workerId?: string;
}

const WorkerDetailContext = createContext<WorkerDetailState | null>(null);

export function useWorkerDetail() {
  const context = useContext(WorkerDetailContext);
  if (!context) {
    throw new Error(
      'useWorkerDetail must be used within a WorkerDetailProvider',
    );
  }
  return context;
}

function WorkerDetailProviderContent({
  children,
  workerId,
}: WorkerDetailProviderProps) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  const workerDetailQuery = useQuery({
    queryKey: ['worker:detail', tenant, workerId],
    queryFn: async () => {
      if (!tenant || !workerId) {
        return undefined;
      }

      const res = await api.workerGet(workerId);
      return res.data;
    },
    enabled: !!workerId,
  });

  const updateWorkerMutation = useMutation({
    mutationKey: ['worker:update', tenant, workerId],
    mutationFn: async ({ workerId, data }: UpdateWorkerParams) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.workerUpdate(workerId, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['worker:detail', tenant, workerId],
      });
      queryClient.invalidateQueries({
        queryKey: ['worker:list'],
      });
    },
  });

  const value = {
    data: workerDetailQuery.data,
    isLoading: workerDetailQuery.isLoading,
    update: updateWorkerMutation,
  };

  return createElement(WorkerDetailContext.Provider, { value }, children);
}

export function WorkerDetailProvider({
  children,
  workerId,
}: WorkerDetailProviderProps) {
  return (
    <WorkerDetailProviderContent workerId={workerId}>
      {children}
    </WorkerDetailProviderContent>
  );
}
