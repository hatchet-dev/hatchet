import { AdditionalMetadata } from './additional-metadata';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { V1Event } from '@/lib/api';
import { ColumnDef } from '@tanstack/react-table';

export const EventColumn = {
  id: 'Event ID',
  key: 'Event',
  seenAt: 'Seen at',
  workflowId: 'Workflow',
  status: 'Status',
  runs: 'Runs',
  metadata: 'Metadata',
  payload: 'Payload',
  scope: 'Scope',
};

type EventColumnKeys = keyof typeof EventColumn;

export const idKey: EventColumnKeys = 'id';
export const keyKey: EventColumnKeys = 'key';
const seenAtKey: EventColumnKeys = 'seenAt';
export const workflowKey: EventColumnKeys = 'workflowId';
export const statusKey: EventColumnKeys = 'status';
const runsKey: EventColumnKeys = 'runs';
export const metadataKey: EventColumnKeys = 'metadata';
const payloadKey: EventColumnKeys = 'payload';
export const scopeKey: EventColumnKeys = 'scope';

export const columns = ({
  onRowClick,
  openMetadataPopover,
  setOpenMetadataPopover,
  openPayloadPopover,
  setOpenPayloadPopover,
}: {
  onRowClick?: (row: V1Event) => void;
  openMetadataPopover: string | null;
  setOpenMetadataPopover: (id: string | null) => void;
  openPayloadPopover: string | null;
  setOpenPayloadPopover: (id: string | null) => void;
}): ColumnDef<V1Event>[] => {
  return [
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.id} />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="w-fit pl-0"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            {row.original.metadata.id}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: keyKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.key} />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="h-auto min-w-0 justify-start whitespace-normal pl-0 text-left"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            <span className="break-all">{row.original.key}</span>
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: seenAtKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.seenAt} />
      ),
      cell: ({ row }) => {
        return (
          <div>
            <RelativeDate date={row.original.metadata.createdAt} />
          </div>
        );
      },
      enableSorting: false,
    },
    // empty columns to get column filtering to work properly
    {
      accessorKey: workflowKey,
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: statusKey,
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: runsKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.runs} />
      ),
      cell: ({ row }) => {
        if (!row.original.workflowRunSummary) {
          return <div>None</div>;
        }

        return <WorkflowRunSummary event={row.original} />;
      },
      enableSorting: false,
    },
    {
      accessorKey: metadataKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.metadata} />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.additionalMetadata}
            isOpen={openMetadataPopover === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setOpenMetadataPopover(row.original.metadata.id);
              } else {
                setOpenMetadataPopover(null);
              }
            }}
            title="Metadata"
            align="end"
          />
        );
      },
      enableSorting: false,
    },
    {
      accessorKey: payloadKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.payload} />
      ),
      cell: ({ row }) => {
        if (!row.original.payload) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.payload}
            isOpen={openPayloadPopover === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setOpenPayloadPopover(row.original.metadata.id);
              } else {
                setOpenPayloadPopover(null);
              }
            }}
            title="Payload"
            align="start"
          />
        );
      },
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: scopeKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.scope} />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.scope}</div>,
      enableSorting: false,
      enableHiding: true,
    },
  ];
};

type BadgeProps = {
  variant: 'failed' | 'successful' | 'inProgress' | 'cancelled' | 'queued';
  count: number;
  label: string;
};

function WorkflowRunSummary({ event }: { event: V1Event }) {
  const numFailed = event.workflowRunSummary?.failed || 0;
  const numSucceeded = event.workflowRunSummary?.succeeded || 0;
  const numRunning = event.workflowRunSummary?.running || 0;
  const numCancelled = event.workflowRunSummary?.cancelled || 0;
  const numQueued = event.workflowRunSummary?.queued || 0;

  const badges: BadgeProps[] = [
    { variant: 'failed', count: numFailed, label: 'Failed' },
    { variant: 'successful', count: numSucceeded, label: 'Succeeded' },
    { variant: 'inProgress', count: numRunning, label: 'Running' },
    { variant: 'cancelled', count: numCancelled, label: 'Cancelled' },
    { variant: 'queued', count: numQueued, label: 'Queued' },
  ];

  return (
    <div className="flex flex-row items-center justify-start gap-2">
      {badges.map(
        ({ variant, count, label }) =>
          count > 0 && (
            <Badge variant={variant} key={variant}>
              {count} {label}
            </Badge>
          ),
      )}
    </div>
  );
}
