import { StreamTopicWithRetention } from '../hooks/use-stream-topics';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { ColumnDef } from '@tanstack/react-table';

export const StreamTopicColumn = {
  topic: 'Topic',
  namespace: 'Namespace',
  messageCount: 'Messages',
  retention: 'Retention',
  createdAt: 'Created',
  lastPublishedAt: 'Last Published',
};

type StreamTopicColumnKeys = keyof typeof StreamTopicColumn;

export const topicKey: StreamTopicColumnKeys = 'topic';
const namespaceKey: StreamTopicColumnKeys = 'namespace';
const messageCountKey: StreamTopicColumnKeys = 'messageCount';
const retentionKey: StreamTopicColumnKeys = 'retention';
const createdAtKey: StreamTopicColumnKeys = 'createdAt';
const lastPublishedAtKey: StreamTopicColumnKeys = 'lastPublishedAt';

export function formatRetention(hours: number) {
  if (hours % 24 === 0) {
    const days = hours / 24;
    return `${days} ${days === 1 ? 'day' : 'days'}`;
  }

  return `${hours} ${hours === 1 ? 'hour' : 'hours'}`;
}

export const columns = (): ColumnDef<StreamTopicWithRetention>[] => [
  {
    accessorKey: topicKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={StreamTopicColumn.topic} />
    ),
    cell: ({ row }) => (
      <div className="pl-4 font-mono">{row.original.topic}</div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: namespaceKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.namespace}
      />
    ),
    cell: ({ row }) =>
      row.original.namespace ? (
        <div className="font-mono">{row.original.namespace}</div>
      ) : (
        <div className="text-muted-foreground">default</div>
      ),
    enableSorting: false,
  },
  {
    accessorKey: messageCountKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.messageCount}
      />
    ),
    cell: ({ row }) => <div>{row.original.messageCount.toLocaleString()}</div>,
    enableSorting: false,
  },
  {
    accessorKey: retentionKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.retention}
      />
    ),
    cell: ({ row }) => (
      <div>{formatRetention(row.original.retentionHours)}</div>
    ),
    enableSorting: false,
  },
  {
    accessorKey: createdAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.createdAt}
      />
    ),
    cell: ({ row }) => <RelativeDate date={row.original.createdAt} />,
    enableSorting: false,
  },
  {
    accessorKey: lastPublishedAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.lastPublishedAt}
      />
    ),
    cell: ({ row }) => <RelativeDate date={row.original.lastPublishedAt} />,
    enableSorting: false,
  },
];
