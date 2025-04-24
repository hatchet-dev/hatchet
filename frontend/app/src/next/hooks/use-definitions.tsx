import api, { Workflow, WorkflowWorkersCount } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import useTenant from './use-tenant';
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
  const { tenant } = useTenant();
  const { toast } = useToast();

  const listDefinitionsQuery = useQuery({
    queryKey: ['definition:list', tenant, pagination],
    queryFn: async () => {
      if (!tenant) {
        pagination?.setNumPages(1);
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      try {
        // Fetch workflow list as a basis for definitions
        const res = await api.workflowList(tenant?.metadata.id || '');

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
