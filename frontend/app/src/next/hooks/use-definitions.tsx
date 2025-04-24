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
            slots: {},
          };
        }

        // Fetch workers count for all workflows
        const workersPromises = res.data.rows.map((workflow) =>
          api.workflowGetWorkersCount(
            tenant?.metadata.id || '',
            workflow.metadata.id,
          ),
        );
        const workersResults = await Promise.all(workersPromises);

        // Create slots object with all workflows
        const slots = res.data.rows.reduce(
          (acc, workflow, index) => {
            acc[workflow.name] = workersResults[index].data;
            return acc;
          },
          {} as { [name: string]: WorkflowWorkersCount },
        );

        return {
          rows: res.data.rows,
          slots,
        };
      } catch (error) {
        toast({
          title: 'Error fetching workflow definitions',

          variant: 'destructive',
          error,
        });
        return {
          rows: [],
          slots: {},
        };
      }
    },
  });

  return {
    data: listDefinitionsQuery.data?.rows || [],
    slots: listDefinitionsQuery.data?.slots || {},
    isLoading: listDefinitionsQuery.isLoading,
  };
}
