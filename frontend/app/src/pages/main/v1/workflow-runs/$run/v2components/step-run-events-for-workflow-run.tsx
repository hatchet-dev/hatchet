import { queries, V1TaskEvent } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './events-columns';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export function StepRunEvents({
  taskRunId,
  taskDisplayName,
  onClick,
}: {
  taskRunId: string;
  taskDisplayName: string;
  onClick: (stepRunId?: string) => void;
}) {
  const { tenantId } = useCurrentTenantId();

  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(
      tenantId,
      {
        // TODO: Pagination here
        limit: 50,
        offset: 0,
      },
      taskRunId,
    ),
    refetchInterval: () => {
      return 5000;
    },
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
    taskDisplayName,
  });

  return (
    <DataTable
      emptyState={<>No events found.</>}
      isLoading={eventsQuery.isLoading}
      columns={cols as any} // TODO: This is a hack, figure out how to type this properly later
      filters={[]}
      data={events}
    />
  );
}
