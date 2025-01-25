import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './events-columns';
import { useTenant } from '@/lib/atoms';

export function StepRunEvents({
  taskRunId,
  onClick,
}: {
  taskRunId: string;
  onClick?: (stepRunId?: string) => void;
}) {
  const tenant = useTenant();
  const tenantId = tenant.tenant?.metadata.id;

  if (!tenantId) {
    throw new Error('Tenant ID not found');
  }

  const eventsQuery = useQuery({
    ...queries.v2StepRunEvents.list(tenantId, taskRunId, {
      limit: 50,
      offset: 0,
    }),
    refetchInterval: () => {
      return 5000;
    },
  });

  const events = eventsQuery.data?.rows || [];

  const cols = columns({
    onRowClick: undefined,
    // onClick
    //   ? (row) => onClick(row.stepRun?.metadata.id)
    //   : undefined,
  });

  return (
    <DataTable
      emptyState={<>No events found.</>}
      isLoading={eventsQuery.isLoading}
      columns={cols}
      filters={[]}
      data={events}
    />
  );
}
