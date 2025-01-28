import { queries, V2StepRunEvent } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './events-columns';
import { useTenant } from '@/lib/atoms';

export function StepRunEvents({
  taskRunId,
  onClick,
}: {
  taskRunId: string;
  onClick: (stepRunId?: string) => void;
}) {
  const tenant = useTenant();
  const tenantId = tenant.tenant?.metadata.id;

  if (!tenantId) {
    throw new Error('Tenant ID not found');
  }

  const eventsQuery = useQuery({
    ...queries.v2StepRunEvents.list(tenantId, taskRunId, {
      // TODO: Pagination here
      limit: 50,
      offset: 0,
    }),
    refetchInterval: () => {
      return 5000;
    },
  });

  type EventWithMetadata = V2StepRunEvent & {
    metadata: {
      id: string;
    };
  };

  const events: EventWithMetadata[] =
    eventsQuery.data?.rows?.map((row) => ({
      ...row,
      metadata: {
        id: row.taskId,
      },
    })) || [];

  const cols = columns({
    onRowClick: (row) => onClick(row.taskId),
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
