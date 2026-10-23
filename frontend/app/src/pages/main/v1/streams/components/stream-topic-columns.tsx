import { StreamTopicWithMetadata } from '../hooks/use-stream-topics';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { ColumnDef } from '@tanstack/react-table';

export const StreamTopicColumn = {
  topic: 'Topic',
  namespace: 'Namespace',
  messageCount: 'Messages',
  createdAt: 'Created',
  lastPublishedAt: 'Last Published',
};

type StreamTopicColumnKeys = keyof typeof StreamTopicColumn;

export const topicKey: StreamTopicColumnKeys = 'topic';
const namespaceKey: StreamTopicColumnKeys = 'namespace';
const messageCountKey: StreamTopicColumnKeys = 'messageCount';
const createdAtKey: StreamTopicColumnKeys = 'createdAt';
const lastPublishedAtKey: StreamTopicColumnKeys = 'lastPublishedAt';

export const columns = (): ColumnDef<StreamTopicWithMetadata>[] => [
  {
    accessorKey: topicKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={StreamTopicColumn.topic} />
    ),
    cell: ({ row }) => <div className="pl-4">{row.original.topic}</div>,
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
        <div>{row.original.namespace}</div>
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
