import { workflowRunStatusFilters } from '../../workflow-runs-v1/hooks/use-toolbar-filters';
import {
  keyKey,
  workflowKey,
  statusKey,
  metadataKey,
  idKey,
  scopeKey,
} from '../components/event-columns';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import api, { queries, V1TaskStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { z } from 'zod';

type UseEventsProps = {
  key: string;
};

const eventFilterSchema = z
  .object({
    k: z.array(z.string()).default([]), // keys
    w: z.array(z.string()).default([]), // workflow ids
    s: z.array(z.string()).default([]), // scopes
    st: z.array(z.nativeEnum(V1TaskStatus)).default([]), // statuses
    m: z.array(z.string()).default([]), // metadata
    i: z.array(z.string()).default([]), // event ids
  })
  .default({});

export const useEvents = ({ key }: UseEventsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `events-${key}`;
  const {
    state: {
      k: selectedKeys,
      w: selectedWorkflowIds,
      s: selectedScopes,
      st: selectedStatuses,
      m: selectedMetadata,
      i: selectedEventIds,
    },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(eventFilterSchema, paramKey, {
    k: keyKey,
    w: workflowKey,
    s: scopeKey,
    st: statusKey,
    m: metadataKey,
    i: idKey,
  });

  const { data, isLoading, refetch, error, isRefetching } = useQuery({
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
    refetchInterval: selectedEventIds?.length ? false : refetchInterval,
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
    isRefetching,
    resetFilters,
  };
};
