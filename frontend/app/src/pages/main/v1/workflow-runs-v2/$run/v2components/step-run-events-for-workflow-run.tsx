import { queries, V2TaskEvent } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './events-columns';
import { useTenant } from '@/lib/atoms';

export function StepRunEvents({
  taskRunId,
  taskDisplayName,
  onClick,
}: {
  taskRunId: string;
  taskDisplayName: string;
  onClick: (stepRunId?: string) => void;
}) {
  const tenant = useTenant();
  const tenantId = tenant.tenant?.metadata.id;

  if (!tenantId) {
    throw new Error('Tenant ID not found');
  }

  const eventsQuery = useQuery({
    ...queries.v2TaskEvents.list(tenantId, taskRunId, {
      // TODO: Pagination here
      limit: 50,
      offset: 0,
    }),
    refetchInterval: () => {
      return 5000;
    },
  });

  type EventWithMetadata = V2TaskEvent & {
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
