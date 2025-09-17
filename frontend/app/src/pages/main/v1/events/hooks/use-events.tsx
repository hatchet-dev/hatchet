import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { queries, V1TaskStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { ColumnFiltersState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  keyKey,
  workflowKey,
  statusKey,
  metadataKey,
  idKey,
  scopeKey,
} from '../components/event-columns';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';

type UseEventsProps = {
  key: string;
  hoveredEventId?: string | null;
};

type EventFilterQueryShape = {
  k: string[]; // keys
  w: string[]; // workflow ids
  s: string[]; // scopes
  st: V1TaskStatus[]; // statuses
  m: string[]; // metadata
  i: string[]; // event ids
};

const parseEventFilterParam = (searchParams: URLSearchParams, key: string) => {
  const rawFilterParamValue = searchParams.get(key);

  if (!rawFilterParamValue) {
    return {
      k: [],
      w: [],
      s: [],
      st: [],
      m: [],
      i: [],
    };
  }

  const parsedFilterState = JSON.parse(rawFilterParamValue);

  if (!parsedFilterState || typeof parsedFilterState !== 'object') {
    return {
      k: [],
      w: [],
      s: [],
      st: [],
      m: [],
      i: [],
    };
  }

  const { k, w, s, st, m, i }: EventFilterQueryShape = parsedFilterState;

  return {
    k: Array.isArray(k) && k.length > 0 ? k : undefined,
    w: Array.isArray(w) && w.length > 0 ? w : undefined,
    s: Array.isArray(s) && s.length > 0 ? s : undefined,
    st: Array.isArray(st) && st.length > 0 ? st : undefined,
    m: Array.isArray(m) && m.length > 0 ? m : undefined,
    i: Array.isArray(i) && i.length > 0 ? i : undefined,
  };
};

export const useEvents = ({ key, hoveredEventId }: UseEventsProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { tenantId } = useCurrentTenantId();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `events-${key}`;

  const selectedKeys = useMemo(() => {
    const { k } = parseEventFilterParam(searchParams, paramKey);
    return k;
  }, [searchParams, paramKey]);

  const selectedWorkflowIds = useMemo(() => {
    const { w } = parseEventFilterParam(searchParams, paramKey);
    return w;
  }, [searchParams, paramKey]);

  const selectedScopes = useMemo(() => {
    const { s } = parseEventFilterParam(searchParams, paramKey);
    return s;
  }, [searchParams, paramKey]);

  const selectedStatuses = useMemo(() => {
    const { st } = parseEventFilterParam(searchParams, paramKey);
    return st;
  }, [searchParams, paramKey]);

  const selectedMetadata = useMemo(() => {
    const { m } = parseEventFilterParam(searchParams, paramKey);
    return m;
  }, [searchParams, paramKey]);

  const selectedEventIds = useMemo(() => {
    const { i } = parseEventFilterParam(searchParams, paramKey);
    return i;
  }, [searchParams, paramKey]);

  const columnFilters = useMemo<ColumnFiltersState>(() => {
    const { k, w, s, st, m, i } = parseEventFilterParam(searchParams, paramKey);
    const filters: ColumnFiltersState = [];

    if (k && k.length > 0) {
      filters.push({ id: keyKey, value: k });
    }

    if (w && w.length > 0) {
      filters.push({ id: workflowKey, value: w });
    }

    if (s && s.length > 0) {
      filters.push({ id: scopeKey, value: s });
    }

    if (st && st.length > 0) {
      filters.push({ id: statusKey, value: st });
    }

    if (m && m.length > 0) {
      filters.push({ id: metadataKey, value: m });
    }

    if (i && i.length > 0) {
      filters.push({ id: idKey, value: i });
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

        const keyFilter = newColumnFilters.find((f) => f.id === keyKey);
        const workflowFilter = newColumnFilters.find(
          (f) => f.id === workflowKey,
        );
        const scopeFilter = newColumnFilters.find((f) => f.id === scopeKey);
        const statusFilter = newColumnFilters.find((f) => f.id === statusKey);
        const metadataFilter = newColumnFilters.find(
          (f) => f.id === metadataKey,
        );
        const idFilter = newColumnFilters.find((f) => f.id === idKey);

        const keys = keyFilter?.value
          ? Array.isArray(keyFilter.value)
            ? (keyFilter.value as string[])
            : [keyFilter.value as string]
          : [];

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

        const statuses = statusFilter?.value
          ? Array.isArray(statusFilter.value)
            ? (statusFilter.value as V1TaskStatus[])
            : [statusFilter.value as V1TaskStatus]
          : [];

        const metadata = metadataFilter?.value
          ? Array.isArray(metadataFilter.value)
            ? (metadataFilter.value as string[])
            : [metadataFilter.value as string]
          : [];

        const eventIds = idFilter?.value
          ? Array.isArray(idFilter.value)
            ? (idFilter.value as string[])
            : [idFilter.value as string]
          : [];

        const filterState: EventFilterQueryShape = {
          k: keys,
          w: workflowIds,
          s: scopes,
          st: statuses,
          m: metadata,
          i: eventIds,
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
    queryKey: [
      'v1:events:list',
      tenantId,
      {
        keys: selectedKeys,
        workflows: selectedWorkflowIds,
        offset,
        limit,
        statuses: selectedStatuses,
        additionalMetadata: selectedMetadata,
        eventIds: selectedEventIds,
        scopes: selectedScopes,
      },
    ],
    queryFn: async () => {
      const response = await api.v1EventList(tenantId, {
        offset,
        limit,
        keys: selectedKeys,
        since: undefined,
        until: undefined,
        eventIds: selectedEventIds,
        workflowRunStatuses: selectedStatuses,
        additionalMetadata: selectedMetadata,
        workflowIds: selectedWorkflowIds,
        scopes: selectedScopes,
      });

      return response.data;
    },
    refetchInterval: hoveredEventId || selectedEventIds?.length ? false : 5000,
    placeholderData: (prev) => prev,
  });

  const events = data?.rows ?? [];
  const numEvents = data?.pagination?.num_pages ?? 1;

  const {
    data: eventKeys,
    isLoading: eventKeysIsLoading,
    error: eventKeysError,
  } = useQuery({
    queryKey: ['v1:events:listKeys', tenantId],
    queryFn: async () => {
      const response = await api.v1EventKeyList(tenantId);
      return response.data;
    },
  });

  const eventKeyFilters = useMemo((): FilterOption[] => {
    return (
      eventKeys?.rows?.map((key) => ({
        value: key,
        label: key,
      })) || []
    );
  }, [eventKeys]);

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowRunStatusFilters = useMemo((): FilterOption[] => {
    return [
      {
        value: V1TaskStatus.COMPLETED,
        label: 'Succeeded',
      },
      {
        value: V1TaskStatus.FAILED,
        label: 'Failed',
      },
      {
        value: V1TaskStatus.RUNNING,
        label: 'Running',
      },
      {
        value: V1TaskStatus.QUEUED,
        label: 'Queued',
      },
      {
        value: V1TaskStatus.CANCELLED,
        label: 'Cancelled',
      },
    ];
  }, []);

  return {
    events,
    numEvents,
    isLoading: isLoading || eventKeysIsLoading || workflowKeysIsLoading,
    refetch,
    error: error || eventKeysError || workflowKeysError,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    selectedKeys,
    selectedWorkflowIds,
    selectedScopes,
    selectedStatuses,
    selectedMetadata,
    selectedEventIds,
    eventKeyFilters,
    workflowKeyFilters,
    workflowRunStatusFilters,
  };
};
