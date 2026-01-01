import {
  TaskEventCell,
  TimestampCell,
  EventTypeCell,
  DescriptionCell,
} from './events-columns';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries, V1TaskEvent } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export type EventWithMetadata = V1TaskEvent & {
  metadata: {
    id: string;
  };
};

export function StepRunEvents({
  taskRunId,
  workflowRunId,
  fallbackTaskDisplayName,
  onClick,
}: {
  taskRunId?: string | undefined;
  workflowRunId?: string | undefined;
  fallbackTaskDisplayName: string;
  onClick?: (stepRunId: string) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(
      tenantId,
      {
        // todo: implement pagination
        limit: 50,
        offset: 0,
      },
      taskRunId,
      workflowRunId,
    ),
    refetchInterval,
  });

  const events: EventWithMetadata[] =
    eventsQuery.data?.rows?.map((row) => ({
      ...row,
      metadata: {
        id: `${row.id}`,
      },
    })) || [];

  const eventColumns = useMemo(
    () => [
      {
        columnLabel: 'Task',
        cellRenderer: (event: EventWithMetadata) => (
          <TaskEventCell
            event={event}
            fallbackTaskDisplayName={fallbackTaskDisplayName}
            onRowClick={(event) => {
              if (onClick) {
                onClick(`${event.taskId}`);
              }
            }}
          />
        ),
      },
      {
        columnLabel: 'Seen at',
        cellRenderer: (event: EventWithMetadata) => (
          <TimestampCell event={event} />
        ),
      },
      {
        columnLabel: 'Event',
        cellRenderer: (event: EventWithMetadata) => (
          <EventTypeCell event={event} />
        ),
      },
      {
        columnLabel: 'Description',
        cellRenderer: (event: EventWithMetadata) => (
          <DescriptionCell event={event} tenantId={tenantId} />
        ),
      },
    ],
    [tenantId, fallbackTaskDisplayName, onClick],
  );

  if (eventsQuery.isLoading) {
    return (
      <div className="h-[400px] min-h-0 flex-1">
        <div className="py-8 text-center text-sm text-muted-foreground">
          Loading events...
        </div>
      </div>
    );
  }

  return (
    <div>
      {events.length > 0 ? (
        <SimpleTable columns={eventColumns} data={events} />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No events found.
        </div>
      )}
    </div>
  );
}
