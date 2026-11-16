import { StreamTopicWithMetadata } from '../hooks/use-stream-topics';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { cn } from '@/lib/utils';
import { ArrowDownIcon } from '@radix-ui/react-icons';
import { ColumnDef } from '@tanstack/react-table';

export const StreamTopicColumn = {
  topic: 'Topic',
  namespace: 'Namespace',
  messageCount: 'Messages',
  lastPublishedAt: 'Last Published',
  createdAt: 'Created',
  startingOffset: 'Read From',
};

type StreamTopicColumnKeys = keyof typeof StreamTopicColumn;

export const topicKey: StreamTopicColumnKeys = 'topic';
const namespaceKey: StreamTopicColumnKeys = 'namespace';
const messageCountKey: StreamTopicColumnKeys = 'messageCount';
const lastPublishedAtKey: StreamTopicColumnKeys = 'lastPublishedAt';
const createdAtKey: StreamTopicColumnKeys = 'createdAt';
const startingOffsetKey: StreamTopicColumnKeys = 'startingOffset';

const LIVE_MS = 5 * 60 * 1000;
const IDLE_MS = 24 * 60 * 60 * 1000;

function ActivityDot({ lastPublishedAt }: { lastPublishedAt: string }) {
  const sinceMs = Date.now() - new Date(lastPublishedAt).getTime();

  if (sinceMs < LIVE_MS) {
    return (
      <span
        className="relative flex size-2 shrink-0"
        title="Published in the last 5 minutes"
      >
        <span className="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-75" />
        <span className="relative inline-flex size-2 rounded-full bg-green-500" />
      </span>
    );
  }

  const idle = sinceMs > IDLE_MS;

  return (
    <span
      className={cn(
        'inline-flex size-2 shrink-0 rounded-full',
        idle ? 'bg-zinc-300 dark:bg-zinc-600' : 'bg-zinc-400 dark:bg-zinc-500',
      )}
      title={
        idle ? 'Nothing published for over a day' : 'Published in the last day'
      }
    />
  );
}

export const columns = (): ColumnDef<StreamTopicWithMetadata>[] => [
  {
    accessorKey: topicKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={StreamTopicColumn.topic} />
    ),
    cell: ({ row }) => (
      <div className="flex items-center gap-2 pl-4">
        <ActivityDot lastPublishedAt={row.original.lastPublishedAt} />
        <span>{row.original.topic}</span>
      </div>
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
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={cn(
          'rounded-sm bg-muted px-1.5 font-mono font-normal',
          !row.original.namespace && 'text-muted-foreground',
        )}
      >
        {row.original.namespace || 'default'}
      </Badge>
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
    accessorKey: lastPublishedAtKey,
    header: () => (
      <div
        className="flex items-center gap-1 text-xs"
        title="Sorted by last published, newest first"
      >
        {StreamTopicColumn.lastPublishedAt}
        <ArrowDownIcon className="size-3.5" aria-hidden="true" />
        <span className="sr-only">(newest first)</span>
      </div>
    ),
    cell: ({ row }) => <RelativeDate date={row.original.lastPublishedAt} />,
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
    accessorKey: startingOffsetKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={StreamTopicColumn.startingOffset}
      />
    ),
    cell: ({ row }) =>
      row.original.startingOffset > 0 ? (
        <div
          className="font-mono tabular-nums text-blue-700 dark:text-blue-400"
          title="The topic was reset: readers without a cursor start after this offset"
        >
          after #{row.original.startingOffset.toLocaleString()}
        </div>
      ) : (
        <div
          className="text-muted-foreground"
          title="Readers without a cursor start from the oldest retained message"
        >
          oldest
        </div>
      ),
    enableSorting: false,
  },
];
