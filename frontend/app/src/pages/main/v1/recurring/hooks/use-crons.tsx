import { workflowKey, metadataKey } from '../components/recurring-columns';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import api, {
  queries,
  CronWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
  UpdateCronWorkflowTriggerRequest,
} from '@/lib/api';
import queryClient from '@/query-client';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { z } from 'zod';

type UseCronsProps = {
  key: string;
};

const cronFilterSchema = z
  .object({
    w: z.array(z.string()).default([]), // workflow ids
    m: z.array(z.string()).default([]), // metadata
  })
  .default({});

export const useCrons = ({ key }: UseCronsProps) => {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { limit, offset, pagination, setPagination, setPageSize } =
    usePagination({
      key,
    });

  const paramKey = `crons-${key}`;
  const {
    state: { w: selectedWorkflowIds, m: selectedMetadata },
    columnFilters,
    setColumnFilters,
    resetFilters,
  } = useZodColumnFilters(cronFilterSchema, paramKey, {
    w: workflowKey,
    m: metadataKey,
  });

  const { data, isLoading, refetch, error, isRefetching } = useQuery({
    ...queries.cronJobs.list(tenantId, {
      orderByField: CronWorkflowsOrderByField.CreatedAt,
      orderByDirection: WorkflowRunOrderByDirection.DESC,
      offset,
      limit,
      // todo: allow multiple workflow ids here
      workflowId: selectedWorkflowIds[0],
      additionalMetadata:
        selectedMetadata.length > 0 ? selectedMetadata : undefined,
    }),
    refetchInterval,
  });

  const updateCronMutation = useMutation({
    mutationKey: ['cron:update'],
    mutationFn: async ({
      tenantId,
      cronId,
      data,
    }: {
      tenantId: string;
      cronId: string;
      data: UpdateCronWorkflowTriggerRequest;
    }) => {
      await api.workflowCronUpdate(tenantId, cronId, data);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.cronJobs.list(tenantId, {}).queryKey,
      });
    },
  });

  const updatingCronId = updateCronMutation.variables?.cronId;

  const updateCron = useCallback(
    (
      tenantId: string,
      cronId: string,
      data: UpdateCronWorkflowTriggerRequest,
    ) =>
      updateCronMutation.mutate({
        tenantId: tenantId,
        cronId: cronId,
        data: data,
      }),
    [updateCronMutation],
  );

  const crons = data?.rows ?? [];
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
    crons,
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
    selectedMetadata,
    workflowKeyFilters,
    isRefetching,
    resetFilters,
    updateCron,
    isUpdatePending: updateCronMutation.isPending,
    updatingCronId,
  };
};
