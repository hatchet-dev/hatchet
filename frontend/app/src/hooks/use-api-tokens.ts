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

// Types for filters and pagination
interface TokensFilters {
  search?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
}

interface TokensPagination {
  currentPage: number;
  pageSize: number;
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
  paginationState: TokensPagination;
  setPagination: (pagination: TokensPagination) => void;
}

interface UseApiTokensOptions {
  refetchInterval?: number;
  initialFilters?: TokensFilters;
  initialPagination?: TokensPagination;
}

export default function useApiTokens({
  refetchInterval,
  initialFilters = {},
  initialPagination = { currentPage: 1, pageSize: 10 },
}: UseApiTokensOptions = {}): ApiTokensState {
  const { tenant } = useTenant();

  // State from the former context
  const [filters, setFilters] = useState<TokensFilters>(initialFilters);
  const [paginationState, setPagination] =
    useState<TokensPagination>(initialPagination);

  const listTokensQuery = useQuery({
    queryKey: [
      'api-token:list',
      tenant,
      filters.search,
      filters.sortBy,
      filters.sortDirection,
      paginationState.currentPage,
      paginationState.pageSize,
    ],
    queryFn: async () => {
      if (!tenant) {
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }

      // In a real implementation, these params would be passed to the API
      // This is a simplified example as the current API may not support these filters
      const res = await api.apiTokenList(tenant?.metadata.id || '');

      // Client-side filtering for search if API doesn't support it
      let filteredRows = res.data.rows || [];
      if (filters.search) {
        const searchLower = filters.search.toLowerCase();
        filteredRows = filteredRows.filter((token) =>
          token.name.toLowerCase().includes(searchLower),
        );
      }

      // Client-side sorting if API doesn't support it
      if (filters.sortBy) {
        filteredRows.sort((a, b) => {
          let valueA: any;
          let valueB: any;

          switch (filters.sortBy) {
            case 'name':
              valueA = a.name;
              valueB = b.name;
              break;
            case 'createdAt':
              valueA = new Date(a.metadata.createdAt).getTime();
              valueB = new Date(b.metadata.createdAt).getTime();
              break;
            case 'expiresAt':
              valueA = new Date(a.expiresAt).getTime();
              valueB = new Date(b.expiresAt).getTime();
              break;
            default:
              return 0;
          }

          const direction = filters.sortDirection === 'desc' ? -1 : 1;
          if (valueA < valueB) {
            return -1 * direction;
          }
          if (valueA > valueB) {
            return 1 * direction;
          }
          return 0;
        });
      }

      return {
        ...res.data,
        rows: filteredRows,
      };
    },
    refetchInterval,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenant],
    mutationFn: async (data: CreateAPITokenRequest) => {
      if (!tenant) {
        throw new Error('Tenant not found');
      }
      const res = await api.apiTokenCreate(tenant.metadata.id, data);
      return res.data;
    },
    onSuccess: (data) => {
      listTokensQuery.refetch();
      return data;
    },
  });

  const revokeMutation = useMutation({
    mutationKey: ['api-token:revoke', tenant],
    mutationFn: async (apiToken: APIToken) => {
      await api.apiTokenUpdateRevoke(apiToken.metadata.id);
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

    // Added from context
    filters,
    setFilters,
    paginationState,
    setPagination,
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
