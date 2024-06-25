import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantResource, TenantResourceLimit } from '@/lib/api';
import RelativeDate from '@/components/molecules/relative-date';
import { cn } from '@/lib/utils';

const resources: Record<TenantResource, string> = {
  [TenantResource.WORKER]: 'Concurrent Workers',
  [TenantResource.EVENT]: 'Events',
  [TenantResource.WORKFLOW_RUN]: 'Workflow Runs',
  [TenantResource.CRON]: 'Cron Triggers',
  [TenantResource.SCHEDULE]: 'Schedule Triggers',
};

const indicatorVariants = {
  ok: 'border-transparent rounded-full bg-green-500',
  alarm: 'border-transparent rounded-full bg-yellow-500',
  exhausted: 'border-transparent rounded-full bg-red-500',
};

export function LimitIndicator({ limit }: { limit: TenantResourceLimit }) {
  let variant = indicatorVariants.ok;

  if (limit.alarmValue && limit.value >= limit.alarmValue) {
    variant = indicatorVariants.alarm;
  }

  if (limit.value >= limit.limitValue) {
    variant = indicatorVariants.exhausted;
  }

  return <div className={cn(variant, 'rounded-full h-[6px] w-[6px]')} />;
}

const durationMap: Record<string, string> = {
  '24h0m0s': 'Daily',
  '168h0m0s': 'Weekly',
  '720h0m0s': 'Monthly',
};

export const columns = (): ColumnDef<TenantResourceLimit>[] => {
  return [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Resource" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 pl-4">
          <LimitIndicator limit={row.original} />
          {resources[row.original.resource]}
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'current',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Current Value" />
      ),
      cell: ({ row }) => <div>{row.original.value.toLocaleString()}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'limit_value',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Limit Value" />
      ),
      cell: ({ row }) => <div>{row.original.limitValue.toLocaleString()}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'alarm_value',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Alarm Value" />
      ),
      cell: ({ row }) => (
        <div>
          {row.original.alarmValue
            ? row.original.alarmValue.toLocaleString()
            : 'N/A'}
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'window',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Meter Window" />
      ),
      cell: ({ row }) => (
        <div>
          {(row.original.window || '-') in durationMap
            ? durationMap[row.original.window || '-']
            : row.original.window}
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'alarm_value',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Last Refill" />
      ),
      cell: ({ row }) => (
        <div>
          {!row.original.window
            ? 'N/A'
            : row.original.lastRefill && (
                <RelativeDate date={row.original.lastRefill} />
              )}
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
  ];
};
