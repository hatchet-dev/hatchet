import { StepRun, queries, Step } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { Event, columns } from './events-columns';
import { useTenant } from '@/lib/atoms';
import { useLocation } from 'react-router-dom';

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

  const tableData: Event[] =
    events?.map((item) => {
      return {
        eventName: item.taskId,
        timestamp: item.timestamp,
        taskName: item.taskId,
        description: item.message,
      };
    }) || [];

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
      data={tableData}
    />
  );
}
