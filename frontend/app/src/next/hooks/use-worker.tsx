import api from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCurrentTenantId } from './use-tenant';

interface UseWorkerProps {
  workerId: string;
  refetchInterval?: number;
}

export function useWorker({ workerId, refetchInterval }: UseWorkerProps) {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading } = useQuery({
    queryKey: ['worker', tenantId, workerId],
    queryFn: async () => {
      const res = await api.workerGet(workerId);

      return res.data;
    },
    refetchInterval,
    enabled: !!workerId,
  });

  return {
    data,
    isLoading,
  };
}
