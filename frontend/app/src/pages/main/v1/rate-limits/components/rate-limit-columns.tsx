import { LimitIndicator } from '../../tenant-settings/resource-limits/components/resource-limit-columns';
import { RateLimitWithMetadata } from '../hooks/use-rate-limits';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { capitalize } from '@/lib/utils';
import { ColumnDef } from '@tanstack/react-table';

export const RateLimitColumn = {
  key: 'Key',
  value: 'Value',
  limit: 'Limit',
  lastRefill: 'Last Refill',
  window: 'Window',
};

type RateLimitColumnKeys = keyof typeof RateLimitColumn;

export const keyKey: RateLimitColumnKeys = 'key';
const valueKey: RateLimitColumnKeys = 'value';
const limitKey: RateLimitColumnKeys = 'limit';
const lastRefillKey: RateLimitColumnKeys = 'lastRefill';
const windowKey: RateLimitColumnKeys = 'window';

export const columns: ColumnDef<RateLimitWithMetadata>[] = [
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
];
