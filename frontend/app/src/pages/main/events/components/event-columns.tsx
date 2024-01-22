import { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { columns as workflowRunsColumns } from '../../workflow-runs/components/workflow-runs-columns';
import { Event, queries } from '@/lib/api';
import { relativeDate } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';

export const columns = ({
  onRowClick,
}: {
  onRowClick?: (row: Event) => void;
}): ColumnDef<Event>[] => {
  return [
    {
      id: 'select',
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && 'indeterminate')
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
          className="translate-y-[2px]"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          className="translate-y-[2px]"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'key',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Event" />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="w-fit cursor-pointer pl-0"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            {row.getValue('key')}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'Seen at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Seen at" />
      ),
      cell: ({ row }) => {
        return <div>{relativeDate(row.original.metadata.createdAt)}</div>;
      },
    },
    {
      accessorKey: 'workflows',
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: 'Workflow Runs',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Workflow Runs" />
      ),
      cell: ({ row }) => {
        if (!row.original.workflowRunSummary) {
          return <div>None</div>;
        }

        return <WorkflowRunSummary event={row.original} />;
      },
    },
    // {
    //   id: "actions",
    //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
    // },
  ];
};

// eslint-disable-next-line react-refresh/only-export-components
function WorkflowRunSummary({ event }: { event: Event }) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [hoverCardOpen, setPopoverOpen] = useState<
    'failed' | 'succeeded' | 'running'
  >();

  const numFailed = event.workflowRunSummary?.failed || 0;
  const numSucceeded = event.workflowRunSummary?.succeeded || 0;
  const numRunning =
    (event.workflowRunSummary?.pending || 0) +
    (event.workflowRunSummary?.running || 0);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset: 0,
      limit: 10,
      eventId: event.metadata.id,
    }),
    enabled: !!hoverCardOpen,
  });

  const workflowRuns = useMemo(() => {
    return (
      listWorkflowRunsQuery.data?.rows?.filter((run) => {
        if (hoverCardOpen) {
          if (hoverCardOpen == 'failed') {
            return run.status == 'FAILED';
          }
          if (hoverCardOpen == 'succeeded') {
            return run.status == 'SUCCEEDED';
          }
          if (hoverCardOpen == 'running') {
            return run.status == 'RUNNING' || run.status == 'PENDING';
          }
        }

        return false;
      }) || []
    );
  }, [listWorkflowRunsQuery, hoverCardOpen]);

  const hoverCardContent = (
    <div className="min-w-fit z-40 bg-white/10 rounded">
      <DataTable
        columns={workflowRunsColumns}
        data={workflowRuns}
        filters={[]}
        pageCount={0}
        columnVisibility={{
          select: false,
          'Triggered by': false,
          actions: false,
        }}
        isLoading={listWorkflowRunsQuery.isLoading}
      />
    </div>
  );

  return (
    <div className="flex flex-row gap-2 items-center justify-start">
      {numFailed > 0 && (
        <Popover
          open={hoverCardOpen == 'failed'}
          // open={true}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="failed"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('failed')}
            >
              {numFailed} Failed
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numSucceeded > 0 && (
        <Popover
          open={hoverCardOpen == 'succeeded'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="successful"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('succeeded')}
            >
              {numSucceeded} Succeeded
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numRunning > 0 && (
        <Popover
          open={hoverCardOpen == 'running'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="inProgress"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('running')}
            >
              {numRunning} Running
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}
