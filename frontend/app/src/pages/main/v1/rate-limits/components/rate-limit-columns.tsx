import { LimitIndicator } from '../../tenant-settings/resource-limits/components/resource-limit-columns';
import { RateLimitWithMetadata } from '../hooks/use-rate-limits';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { capitalize } from '@/lib/utils';
import { ColumnDef } from '@tanstack/react-table';

export const RateLimitColumn = {
  key: 'Key',
  status: 'Status',
  value: 'Value',
  limit: 'Limit',
  lastRefill: 'Last Refill',
  window: 'Window',
  actions: 'Actions',
};

type RateLimitColumnKeys = keyof typeof RateLimitColumn;

export const keyKey: RateLimitColumnKeys = 'key';
const statusKey: RateLimitColumnKeys = 'status';
const valueKey: RateLimitColumnKeys = 'value';
const limitKey: RateLimitColumnKeys = 'limit';
const lastRefillKey: RateLimitColumnKeys = 'lastRefill';
const windowKey: RateLimitColumnKeys = 'window';
const actionsKey: RateLimitColumnKeys = 'actions';

export const columns = ({
  onDeleteClick,
  canWrite,
}: {
  onDeleteClick: (row: RateLimitWithMetadata) => void;
  canWrite: boolean;
}): ColumnDef<RateLimitWithMetadata>[] => [
  {
    accessorKey: keyKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.key} />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center gap-4 pl-4">
        <LimitIndicator
          value={row.original.limitValue - row.original.value}
          alarmValue={row.original.limitValue / 2}
          limitValue={row.original.limitValue}
        />
        {row.original.key}
      </div>
    ),
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: statusKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.status} />
    ),
    cell: ({ row }) =>
      row.original.pausedUntil ? (
        <div className="flex flex-row items-center gap-2">
          <Badge variant="inProgress">Paused</Badge>
          <span className="text-muted-foreground">
            for <RelativeDate date={row.original.pausedUntil} future />
          </span>
        </div>
      ) : (
        <Badge variant="successful">Active</Badge>
      ),
    enableSorting: false,
  },
  {
    accessorKey: valueKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.value} />
    ),
    cell: ({ row }) => {
      return <div>{row.original.value}</div>;
    },
    enableSorting: false,
  },
  {
    accessorKey: limitKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.limit} />
    ),
    cell: ({ row }) => {
      return <div>{row.original.limitValue}</div>;
    },
    enableSorting: false,
  },
  {
    accessorKey: lastRefillKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={RateLimitColumn.lastRefill}
      />
    ),
    cell: ({ row }) => {
      return (
        <div>
          <RelativeDate date={row.original.lastRefill} />
        </div>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: windowKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.window} />
    ),
    cell: ({ row }) => {
      return <div>{capitalize(row.original.window)}</div>;
    },
    enableSorting: false,
  },
  {
    accessorKey: actionsKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={RateLimitColumn.actions} />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row justify-center">
        <TableRowActions
          row={row.original}
          actions={
            canWrite
              ? [
                  {
                    label: 'Delete',
                    onClick: () => onDeleteClick(row.original),
                  },
                ]
              : []
          }
        />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
];
