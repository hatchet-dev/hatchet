import {
  workflowKey,
  statusKey,
  metadataKey,
} from '../components/scheduled-runs-columns';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import {
  queries,
  ScheduledRunStatus,
  ScheduledWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { z } from 'zod';

type UseScheduledRunsProps = {
  key: string;
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
};

const scheduledRunFilterSchema = z
  .object({
    w: z.array(z.string()).default([]), // workflow ids
    s: z.array(z.nativeEnum(ScheduledRunStatus)).default([]), // statuses
    m: z.array(z.string()).default([]), // metadata
  })
  .default({});

export const useScheduledRuns = ({
  key,
  workflowId,
  parentWorkflowRunId,
  parentStepRunId,
}: UseScheduledRunsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `scheduled-runs-${key}`;
  const {
    state: { w: selectedWorkflowIds, s: selectedStatuses, m: selectedMetadata },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(scheduledRunFilterSchema, paramKey, {
    w: workflowKey,
    s: statusKey,
    m: metadataKey,
  });

  // todo: allow multiple workflow ids here
  const effectiveWorkflowId = workflowId || selectedWorkflowIds[0];

  const { data, isLoading, refetch, error, isRefetching } = useQuery({
    ...queries.scheduledRuns.list(tenantId, {
      offset,
      limit,
      statuses: selectedStatuses.length > 0 ? selectedStatuses : undefined,
      workflowId: effectiveWorkflowId,
      parentWorkflowRunId,
      parentStepRunId,
      orderByDirection: WorkflowRunOrderByDirection.DESC,
      orderByField: ScheduledWorkflowsOrderByField.TriggerAt,
      additionalMetadata:
        selectedMetadata.length > 0 ? selectedMetadata : undefined,
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const scheduledRuns = data?.rows ?? [];
  const numPages = data?.pagination?.num_pages ?? 1;

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
    refetchInterval,
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
    scheduledRuns,
    numPages,
    isLoading: isLoading || workflowKeysIsLoading,
    refetch,
    error: error || workflowKeysError,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    selectedWorkflowIds,
    selectedStatuses,
    selectedMetadata,
    workflowKeyFilters,
    isRefetching,
    resetFilters,
  };
};
