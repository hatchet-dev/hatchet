import { scopeKey, workflowIdKey } from '../components/filter-columns';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  queries,
  V1CreateFilterRequest,
  V1UpdateFilterRequest,
} from '@/lib/api';
import { useSearchParams } from '@/lib/router-helpers';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ColumnFiltersState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';

type UseFiltersProps = {
  key: string;
  scopeOverrides?: string[];
};

type FilterQueryShape = {
  w: string[]; // workflow ids
  s: string[]; // scopes
};

const parseFilterParam = (searchParams: URLSearchParams, key: string) => {
  const rawFilterParamValue = searchParams.get(key);

  if (!rawFilterParamValue) {
    return {
      w: [],
      s: [],
    };
  }

  const parsedFilterState = JSON.parse(rawFilterParamValue);

  if (
    !parsedFilterState ||
    typeof parsedFilterState !== 'object' ||
    !('w' in parsedFilterState) ||
    !('s' in parsedFilterState)
  ) {
    return {
      w: [],
      s: [],
    };
  }

  const { w, s }: FilterQueryShape = parsedFilterState;

  return {
    w: Array.isArray(w) && w.length > 0 ? w : undefined,
    s: Array.isArray(s) && s.length > 0 ? s : undefined,
  };
};

export const useFilters = ({ key, scopeOverrides }: UseFiltersProps) => {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `filters-${key}`;

  const selectedWorkflowIds = useMemo(() => {
    const { w } = parseFilterParam(searchParams, paramKey);

    return w;
  }, [searchParams, paramKey]);

  const selectedScopes = useMemo(() => {
    if (scopeOverrides && scopeOverrides.length > 0) {
      return scopeOverrides;
    }

    const { s } = parseFilterParam(searchParams, paramKey);

    return s;
  }, [searchParams, paramKey, scopeOverrides]);

  const columnFilters = useMemo<ColumnFiltersState>(() => {
    const { w, s } = parseFilterParam(searchParams, paramKey);
    const filters: ColumnFiltersState = [];

    if (w && w.length > 0) {
      filters.push({ id: workflowIdKey, value: w });
    }

    if (s && s.length > 0) {
      filters.push({ id: scopeKey, value: s });
    }

    return filters;
  }, [searchParams, paramKey]);

  const setColumnFilters = useCallback(
    (updater: Updater<ColumnFiltersState>) => {
      setSearchParams((prev) => {
        const currentColumnFilters = columnFilters;
        const newColumnFilters =
          typeof updater === 'function'
            ? updater(currentColumnFilters)
            : updater;

        const workflowFilter = newColumnFilters.find(
          (f) => f.id === workflowIdKey,
        );
        const scopeFilter = newColumnFilters.find((f) => f.id === scopeKey);

        const workflowIds = workflowFilter?.value
          ? Array.isArray(workflowFilter.value)
            ? (workflowFilter.value as string[])
            : [workflowFilter.value as string]
          : [];
        const scopes = scopeFilter?.value
          ? Array.isArray(scopeFilter.value)
            ? (scopeFilter.value as string[])
            : [scopeFilter.value as string]
          : [];

        const filterState: FilterQueryShape = {
          w: workflowIds,
          s: scopes,
        };

        return {
          ...Object.fromEntries(prev.entries()),
          [paramKey]: filterState,
        };
      });
    },
    [columnFilters, paramKey, setSearchParams],
  );

  const { data, isLoading, isRefetching, refetch, error } = useQuery({
    queryKey: [
      'v1:filter:list',
      tenantId,
      key,
      offset,
      limit,
      selectedWorkflowIds,
      selectedScopes,
    ],
    queryFn: async () => {
      const response = await api.v1FilterList(tenantId, {
        offset,
        limit,
        workflowIds: selectedWorkflowIds,
        scopes: selectedScopes,
      });

      return response.data;
    },
    refetchInterval,
    placeholderData: (prev) => prev,
  });

  const filters = data?.rows ?? [];
  const numFilters = data?.pagination?.num_pages ?? 1;

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  const workflowNameFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowIdToName = useMemo(
    () =>
      workflowNameFilters.reduce(
        (acc, curr) => {
          acc[curr.value] = curr.label;
          return acc;
        },
        {} as Record<string, string>,
      ),
    [workflowNameFilters],
  );

  const { mutate: deleteFilterMutation, isPending: isDeleting } = useMutation({
    mutationFn: async (filterId: string) => {
      const response = await api.v1FilterDelete(tenantId, filterId);

      return response.data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['v1:filter:list', tenantId],
      });
    },
  });

  const { mutate: updateFilterMutation, isPending: isUpdating } = useMutation({
    mutationFn: async ({
      filterId,
      data,
    }: {
      filterId: string;
      data: V1UpdateFilterRequest;
    }) => {
      const response = await api.v1FilterUpdate(tenantId, filterId, data);

      return response.data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['v1:filter:list', tenantId],
      });
      await queryClient.invalidateQueries({
        queryKey: ['v1:filter:get', tenantId],
      });
    },
  });

  const { mutate: createFilterMutation, isPending: isCreating } = useMutation({
    mutationFn: async (data: V1CreateFilterRequest) => {
      const response = await api.v1FilterCreate(tenantId, data);

      return response.data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['v1:filter:list', tenantId],
      });
    },
  });

  const deleteFilter = useCallback(
    async (filterId: string) => {
      return deleteFilterMutation(filterId);
    },
    [deleteFilterMutation],
  );

  const updateFilter = useCallback(
    async (filterId: string, data: V1UpdateFilterRequest) => {
      return updateFilterMutation({ filterId, data });
    },
    [updateFilterMutation],
  );

  const createFilter = useCallback(
    async (data: V1CreateFilterRequest) => {
      return createFilterMutation(data);
    },
    [createFilterMutation],
  );

  return {
    filters,
    numFilters,
    isLoading: isLoading || workflowKeysIsLoading,
    refetch,
    isRefetching,
    error: error || workflowKeysError,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    selectedWorkflowIds,
    selectedScopes,
    workflowNameFilters,
    workflowIdToName,
    mutations: {
      delete: {
        perform: deleteFilter,
        isPending: isDeleting,
      },
      update: {
        perform: updateFilter,
        isPending: isUpdating,
      },
      create: {
        perform: createFilter,
        isPending: isCreating,
      },
    },
  };
};

export const useFilterDetails = (filterId: string) => {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['v1:filter:get', tenantId, filterId],
    queryFn: async () => {
      const response = await api.v1FilterGet(tenantId, filterId);

      return response.data;
    },
  });

  return {
    filter: data,
    isLoading,
    error,
    refetch,
  };
};
