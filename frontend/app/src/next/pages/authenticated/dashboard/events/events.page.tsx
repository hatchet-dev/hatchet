import { Checkbox } from '@/components/ui/checkbox';
import { Event } from '@/lib/api';
import { DataTableColumnHeader } from '@/next/components/runs/runs-table/data-table-column-header';
import { DataTable } from '@/next/components/ui/data-table';
import {
  PageSelector,
  PageSizeSelector,
  Pagination,
} from '@/next/components/ui/pagination';
import { useEvents } from '@/next/hooks/use-events';
import useTenant from '@/next/hooks/use-tenant';
import { cn } from '@/next/lib/utils';
import { ColumnDef } from '@tanstack/react-table';

export default function EventsPage() {
  const { tenant } = useTenant();

  const { data, isLoading } = useEvents();

  if (!tenant) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading tenant information...</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading events</p>
      </div>
    );
  }

  return (
    <>
      <DataTable
        columns={columns()}
        data={data || []}
        emptyState={
          <div className="flex flex-col items-center justify-center gap-4 py-8">
            No events found
          </div>
        }
        isLoading={isLoading}
      />
      <Pagination className="mt-4 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
    </>
  );
}

export const columns = (selectAll?: boolean): ColumnDef<Event>[] => [
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={selectAll || table.getIsAllPageRowsSelected()}
        onCheckedChange={(value: boolean) =>
          table.toggleAllPageRowsSelected(!!value)
        }
        aria-label="Select all"
        className="translate-y-[2px]"
        disabled={selectAll}
      />
    ),
    cell: ({ row }) => (
      <div
        className={cn(
          `pl-${row.depth * 4}`,
          'flex flex-row items-center justify-start gap-x-2 max-w-6 mr-2',
        )}
      >
        <Checkbox
          checked={selectAll || row.getIsSelected()}
          onCheckedChange={(value: boolean) => row.toggleSelected(!!value)}
          aria-label="Select row"
          disabled={selectAll}
        />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'key',
    header: ({ column }) => <DataTableColumnHeader column={column} title="" />,
    cell: ({ row }) => {
      const key = row.getValue('key') as string;
      return (
        <div className="flex items-center justify-center h-full">{key}</div>
      );
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
    enableSorting: false,
    enableHiding: false,
  },
];
