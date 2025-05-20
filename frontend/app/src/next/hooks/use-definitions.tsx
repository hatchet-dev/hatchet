import api, { Workflow, WorkflowWorkersCount } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useTenant } from './use-tenant';
import {
  PaginationManager,
  PaginationManagerNoOp,
} from './utils/use-pagination';
import { useToast } from './utils/use-toast';

// Main hook return type
interface DefinitionsState {
  data?: Workflow[]; // The definitions data
  slots?: { [name: string]: WorkflowWorkersCount };
  isLoading: boolean;
}

interface UseDefinitionsOptions {
  refetchInterval?: number;
  pagination?: PaginationManager;
}

export default function useDefinitions({
  pagination = PaginationManagerNoOp,
}: UseDefinitionsOptions = {}): DefinitionsState {
  const { tenantId } = useTenant();
  const { toast } = useToast();

  const listDefinitionsQuery = useQuery({
    queryKey: ['definition:list', tenantId, pagination],
    queryFn: async () => {
      try {
        // Fetch workflow list as a basis for definitions
        const res = await api.workflowList(tenantId);

        if (!res.data.rows) {
          return {
            rows: [],
          };
        }

        return {
          rows: res.data.rows,
        };
      } catch (error) {
        toast({
          title: 'Error fetching workflow definitions',

          variant: 'destructive',
          error,
        });
        return {
          rows: [],
        };
      }
    },
  });

  return {
    data: listDefinitionsQuery.data?.rows || [],
    isLoading: listDefinitionsQuery.isLoading,
  };
}
