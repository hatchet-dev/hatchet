import api, { UpdateWorkerRequest, Worker } from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCurrentTenantId } from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';
import { useToast } from './utils/use-toast';

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
  const { tenantId } = useCurrentTenantId();
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const workerDetailQuery = useQuery({
    queryKey: ['worker:detail', tenantId, workerId],
    queryFn: async () => {
      if (!workerId) {
        return undefined;
      }

      try {
        const res = await api.workerGet(workerId);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching worker details',

          variant: 'destructive',
          error,
        });
        return undefined;
      }
    },
    enabled: !!workerId,
  });

  const updateWorkerMutation = useMutation({
    mutationKey: ['worker:update', tenantId, workerId],
    mutationFn: async ({ workerId, data }: UpdateWorkerParams) => {
      try {
        const res = await api.workerUpdate(workerId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error updating worker',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['worker:detail', tenantId, workerId],
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
