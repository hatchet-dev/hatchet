import { createContext, useContext, useCallback, useMemo } from 'react';
import api from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import useTenant from './use-tenant';
import { FilterProvider, useFilters } from './utils/use-filters';
import { PaginationProvider, usePagination } from './utils/use-pagination';

interface WorkflowsFilters {
  statuses?: string[];
  search?: string;
}

interface WorkflowsState {
  data: any[];
  paginationData?: { current_page: number; num_pages: number };
  isLoading: boolean;
  invalidate: () => Promise<void>;
  filters: ReturnType<typeof useFilters<WorkflowsFilters>>;
  pagination: ReturnType<typeof usePagination>;
}

interface WorkflowsProviderProps {
  children: React.ReactNode;
  refetchInterval?: number;
}

const WorkflowsContext = createContext<WorkflowsState | null>(null);

export function useWorkflows() {
  const context = useContext(WorkflowsContext);
  if (!context) {
    throw new Error('useWorkflows must be used within a WorkflowsProvider');
  }
  return context;
}

function WorkflowsProviderContent({
  children,
  refetchInterval,
}: WorkflowsProviderProps) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();
  const filters = useFilters<WorkflowsFilters>();
  const pagination = usePagination();

  const listWorkflowsQuery = useQuery({
    queryKey: [
      'workflows',
      'list',
      tenant?.metadata.id || 'tenant',
      filters.filters,
      pagination,
    ],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      const res = await api.workflowList(tenant.metadata.id, {
        ...filters.filters,
        offset: Math.max(0, (pagination.currentPage - 1) * pagination.pageSize),
        limit: pagination.pageSize,
      });
      return {
        rows: res.data.rows,
        pagination: {
          current_page: res.data.pagination?.current_page || 0,
          num_pages: res.data.pagination?.num_pages || 0,
        },
      };
    },
    refetchInterval,
  });

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: [
        'workflows',
        'list',
        tenant?.metadata.id || 'tenant',
        filters.filters,
        pagination,
      ],
    });
  }, [queryClient, tenant?.metadata.id, filters.filters, pagination]);

  const value = useMemo(
    () => ({
      data: listWorkflowsQuery.data?.rows || [],
      paginationData: listWorkflowsQuery.data?.pagination,
      isLoading: listWorkflowsQuery.isLoading,
      invalidate,
      filters,
      pagination,
    }),
    [
      listWorkflowsQuery.data,
      listWorkflowsQuery.isLoading,
      invalidate,
      filters,
      pagination,
    ],
  );

  return (
    <WorkflowsContext.Provider value={value}>
      {children}
    </WorkflowsContext.Provider>
  );
}

export function WorkflowsProvider({
  children,
  refetchInterval,
}: WorkflowsProviderProps) {
  return (
    <FilterProvider<WorkflowsFilters>
      initialFilters={{
        statuses: [],
        search: '',
      }}
    >
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <WorkflowsProviderContent refetchInterval={refetchInterval}>
          {children}
        </WorkflowsProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
