import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { ColumnFiltersState, OnChangeFn, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { scopeKey, workflowIdKey } from '../components/filter-columns';

type UseFiltersProps = {
  key: string;
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

export const useFilters = ({ key }: UseFiltersProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenantId } = useCurrentTenantId();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `filters-${key}`;

  const selectedWorkflowIds = useMemo(() => {
    const { w } = parseFilterParam(searchParams, paramKey);

    return w;
  }, [searchParams]);

  const selectedScopes = useMemo(() => {
    const { s } = parseFilterParam(searchParams, paramKey);

    return s;
  }, [searchParams]);

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
          [paramKey]: JSON.stringify(filterState),
        };
      });
    },
    [columnFilters, paramKey, setSearchParams],
  );

  const { data, isLoading, refetch, error } = useQuery({
    queryKey: ['v1:filter:list', tenantId, key],
    queryFn: async () => {
      const response = await api.v1FilterList(tenantId, {
        offset,
        limit,
        workflowIds: selectedWorkflowIds,
        scopes: selectedScopes,
      });

      return response.data;
    },
    refetchInterval: 10000,
  });

  const filters = data?.rows ?? [];

  return {
    filters,
    isLoading,
    refetch,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    selectedWorkflowIds,
    selectedScopes,
  };
};
