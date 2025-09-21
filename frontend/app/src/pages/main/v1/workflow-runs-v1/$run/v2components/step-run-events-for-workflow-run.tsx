import { queries, V1TaskEvent } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './events-columns';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';

export function StepRunEvents({
  taskRunId,
  workflowRunId,
  fallbackTaskDisplayName,
  onClick,
}: {
  taskRunId?: string | undefined;
  workflowRunId?: string | undefined;
  fallbackTaskDisplayName: string;
  onClick: (stepRunId: string) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(
      tenantId,
      {
        // TODO: Pagination here
        limit: 50,
        offset: 0,
      },
      taskRunId,
      workflowRunId,
    ),
    refetchInterval,
  });

  type EventWithMetadata = V1TaskEvent & {
    metadata: {
      id: string;
    };
  };

  const events: EventWithMetadata[] =
    eventsQuery.data?.rows?.map((row) => ({
      ...row,
      metadata: {
        id: `${row.id}`,
      },
    })) || [];

  const cols = columns({
    tenantId,
    onRowClick: (row) => onClick(`${row.taskId}`),
    fallbackTaskDisplayName,
  });

  return (
    <DataTable
      emptyState={<>No events found.</>}
      isLoading={eventsQuery.isLoading}
      columns={cols as any} // TODO: This is a hack, figure out how to type this properly later
      data={events}
    />
  );
}
