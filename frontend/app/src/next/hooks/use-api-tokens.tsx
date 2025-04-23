import api, {
  APIToken,
  CreateAPITokenRequest,
  CreateAPITokenResponse,
  ListAPITokensResponse,
} from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  useState,
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';
import {
  PaginationManager,
  PaginationManagerNoOp,
} from '@/next/components/ui/pagination';
import { useToast } from './utils/use-toast';

// Types for filters
interface TokensFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  fromDate?: string;
  toDate?: string;
}

interface UseApiTokensOptions {
  refetchInterval?: number;
  initialFilters?: TokensFilters;
  paginationManager?: PaginationManager;
}

// Main hook return type
interface ApiTokensState {
  data?: ListAPITokensResponse['rows'];
  pagination?: ListAPITokensResponse['pagination'];
  isLoading: boolean;
  create: UseMutationResult<
    CreateAPITokenResponse,
    Error,
    CreateAPITokenRequest,
    unknown
  >;
  revoke: UseMutationResult<void, Error, APIToken, unknown>;

  // Added from context
  filters: TokensFilters;
  setFilters: (filters: TokensFilters) => void;
}

export default function useApiTokens({
  refetchInterval,
  initialFilters = {},
  paginationManager = PaginationManagerNoOp,
}: UseApiTokensOptions = {}): ApiTokensState {
  const { tenant } = useTenant();
  const { toast } = useToast();

  // State for filters only
  const [filters, setFilters] = useState<TokensFilters>(initialFilters);

  const listTokensQuery = useQuery({
    queryKey: [
      'api-token:list',
      tenant,
      filters.search,
      filters.sortBy,
      filters.sortDirection,
      filters.fromDate,
      filters.toDate,
      paginationManager?.currentPage,
      paginationManager?.pageSize,
    ],
    queryFn: async () => {
      if (!tenant) {
        const pagination = {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
        paginationManager?.setNumPages(pagination.pagination.num_pages);
        return pagination;
      }

      try {
        // Build query params
        const queryParams: Record<string, any> = {
          limit: paginationManager?.pageSize || 10,
          offset:
            (paginationManager?.currentPage - 1) *
              paginationManager?.pageSize || 0,
        };

        if (filters.sortBy) {
          queryParams.orderByField = filters.sortBy;
          queryParams.orderByDirection = filters.sortDirection || 'asc';
        }

        const res = await api.apiTokenList(
          tenant?.metadata.id || '',
          queryParams,
        );

        // Client-side filtering for search if API doesn't support it
        let filteredRows = res.data.rows || [];
        if (filters.search) {
          const searchLower = filters.search.toLowerCase();
          filteredRows = filteredRows.filter((token) =>
            token.name.toLowerCase().includes(searchLower),
          );
        }

        // Client-side date filtering
        if (filters.fromDate) {
          const fromDate = new Date(filters.fromDate);
          filteredRows = filteredRows.filter((token) => {
            const createdAt = new Date(token.metadata.createdAt);
            return createdAt >= fromDate;
          });
        }

        if (filters.toDate) {
          const toDate = new Date(filters.toDate);
          filteredRows = filteredRows.filter((token) => {
            const createdAt = new Date(token.metadata.createdAt);
            return createdAt <= toDate;
          });
        }

        paginationManager?.setNumPages(res.data.pagination?.num_pages || 1);

        return {
          ...res.data,
          rows: filteredRows,
        };
      } catch (error) {
        toast({
          title: 'Error fetching API tokens',

          variant: 'destructive',
          error,
        });
        return {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        };
      }
    },
    refetchInterval,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenant],
    mutationFn: async (data: CreateAPITokenRequest) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      try {
        const res = await api.apiTokenCreate(tenant.metadata.id, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating API token',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: (data) => {
      listTokensQuery.refetch();
      return data;
    },
  });

  const revokeMutation = useMutation({
    mutationKey: ['api-token:revoke', tenant],
    mutationFn: async (apiToken: APIToken) => {
      try {
        await api.apiTokenUpdateRevoke(apiToken.metadata.id);
      } catch (error) {
        toast({
          title: 'Error revoking API token',
          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listTokensQuery.refetch();
    },
  });

  return {
    data: listTokensQuery.data?.rows || [],
    pagination: listTokensQuery.data?.pagination,
    isLoading: listTokensQuery.isLoading,
    create: createTokenMutation,
    revoke: revokeMutation,
    filters,
    setFilters,
  };
}

// Context implementation (to maintain compatibility with components)
interface ApiTokensContextType extends ApiTokensState {}

const ApiTokensContext = createContext<ApiTokensContextType | undefined>(
  undefined,
);

export const useApiTokensContext = () => {
  const context = useContext(ApiTokensContext);
  if (context === undefined) {
    throw new Error(
      'useApiTokensContext must be used within an ApiTokensProvider',
    );
  }
  return context;
};

interface ApiTokensProviderProps extends PropsWithChildren {
  options?: UseApiTokensOptions;
}

export function ApiTokensProvider(props: ApiTokensProviderProps) {
  const { children, options = {} } = props;
  const apiTokensState = useApiTokens(options);

  return createElement(
    ApiTokensContext.Provider,
    { value: apiTokensState },
    children,
  );
}
