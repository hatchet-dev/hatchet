import { usePagination } from '@/hooks/use-pagination';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import {
  queries,
  CronWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useZodColumnFilters } from '@/hooks/use-zod-column-filters';
import { z } from 'zod';
import { workflowKey, metadataKey } from '../components/recurring-columns';

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
  };
};
