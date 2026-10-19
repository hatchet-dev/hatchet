import {
  nameKey,
  statusKey,
  triggersKey,
} from '../components/workflow-columns';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { queries, WorkflowTriggerType } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { useDebounce } from 'use-debounce';
import { z } from 'zod';

type UseWorkflowsProps = {
  key: string;
};

const workflowQuerySchema = z
  .object({
    s: z.string().optional(), // search
    st: z.array(z.enum(['ACTIVE', 'PAUSED'])).optional(), // statuses
    t: z.array(z.nativeEnum(WorkflowTriggerType)).optional(), // trigger types
  })
  .default({});

export const useWorkflows = ({ key }: UseWorkflowsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const paramKey = `workflows-${key}`;

  const {
    state: { s: search, st: statuses, t: triggerTypes },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(
    workflowQuerySchema,
    paramKey,
    {
      s: nameKey,
      st: statusKey,
      t: triggersKey,
    },
    { replaceHistoryEntryOnChange: true },
  );

  const [debouncedSearch] = useDebounce(search, 300);

  // selecting both statuses is the same as selecting none
  const isPaused =
    statuses?.length === 1 ? statuses[0] === 'PAUSED' : undefined;

  const { pagination, setPagination, setPageSize, offset, limit } =
    usePagination({
      key,
      resetPageOnChange: [debouncedSearch, isPaused, triggerTypes],
    });

  const setSearch = useCallback(
    (value: string) => {
      setColumnFilters((prev) => [
        ...prev.filter((f) => f.id !== nameKey),
        ...(value ? [{ id: nameKey, value }] : []),
      ]);
    },
    [setColumnFilters],
  );

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenantId, {
      limit,
      offset,
      name: debouncedSearch,
      isPaused,
      triggerTypes: triggerTypes?.length ? triggerTypes : undefined,
    }),
    refetchInterval,
    placeholderData: (data) => data,
  });

  const workflows = useMemo(
    () => listWorkflowQuery.data?.rows || [],
    [listWorkflowQuery.data?.rows],
  );
  const numWorkflows = listWorkflowQuery.data?.pagination?.num_pages || 0;

  const workflowNameToId = useMemo(
    () =>
      workflows.reduce(
        (acc, wf) => {
          acc[wf.name] = wf.metadata.id;
          return acc;
        },
        {} as Record<string, string>,
      ),
    [workflows],
  );

  return {
    workflows,
    numWorkflows,
    isLoading: listWorkflowQuery.isLoading,
    isRefetching: listWorkflowQuery.isRefetching,
    isPlaceholderData: listWorkflowQuery.isPlaceholderData,
    error: listWorkflowQuery.error,
    refetch: listWorkflowQuery.refetch,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    resetFilters,
    workflowNameToId,
    search: search ?? '',
    setSearch,
  };
};
