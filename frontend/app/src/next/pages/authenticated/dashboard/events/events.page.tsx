import { Checkbox } from '@/components/ui/checkbox';
import { V1Event } from '@/lib/api';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DataTableColumnHeader } from '@/next/components/runs/runs-table/data-table-column-header';
import { Badge } from '@/next/components/ui/badge';
import { Button } from '@/next/components/ui/button';
import { DataTable } from '@/next/components/ui/data-table';
import { DocsButton } from '@/next/components/ui/docs-button';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import {
  PageSelector,
  PageSizeSelector,
  Pagination,
} from '@/next/components/ui/pagination';
import RelativeDate from '@/next/components/ui/relative-date';
import { Separator } from '@/next/components/ui/separator';
import { EventsProvider, useEvents } from '@/next/hooks/use-events';
import useTenant from '@/next/hooks/use-tenant';
import docs from '@/next/lib/docs';
import { ROUTES } from '@/next/lib/routes';
import { AdditionalMetadata } from '@/pages/main/v1/events/components/additional-metadata';
import { ColumnDef } from '@tanstack/react-table';
import { Link } from 'react-router-dom';

function EventsContent() {
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
    <BasicLayout>
      <Headline>
        <PageTitle description="View events pushed to Hatchet">
          Events
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.run_on_event} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>
      <Separator className="my-4" />
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
    </BasicLayout>
  );
}

export const columns = (): ColumnDef<V1Event>[] => {
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
      accessorKey: 'EventId',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="ID" />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Link to={ROUTES.events.detail(row.original.metadata.id)}>
            <Button variant="link">{row.original.metadata.id}</Button>
          </Link>
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'key',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Key" />
      ),
      cell: ({ row }) => <div className="w-full">{row.getValue('key')}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'Seen at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Seen at" />
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
    {
      accessorKey: 'Runs',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Runs" />
      ),
      cell: ({ row }) => {
        if (!row.original.workflowRunSummary) {
          return <div>None</div>;
        }

        const { cancelled, failed, queued, succeeded, running } = row.original
          .workflowRunSummary || {
          cancelled: 0,
          failed: 0,
          succeeded: 0,
          running: 0,
          queued: 0,
        };

        return (
          <div className="flex flex-row gap-2 items-center justify-start">
            {!!queued && <Badge variant="outline">{queued} Queued</Badge>}
            {!!running && (
              <Badge className="bg-amber-400">{running} Running</Badge>
            )}
            {!!cancelled && (
              <Badge className="bg-black border border-red-500 text-white">
                {cancelled} Cancelled
              </Badge>
            )}
            {!!succeeded && (
              <Badge variant="successful">{succeeded} Succeeded</Badge>
            )}
            {!!failed && <Badge variant="destructive">{failed} Failed</Badge>}
          </div>
        );
      },
      enableSorting: false,
    },
    {
      accessorKey: 'Metadata',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Metadata" />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={Object.keys(row.original.additionalMetadata)
              .filter(
                (k) => !['hatchet__event_id', 'hatchet__event_key'].includes(k),
              )
              .reduce<Record<string, unknown>>((acc, k) => {
                const m = row.original.additionalMetadata as Record<
                  string,
                  unknown
                >;

                if (!m) {
                  return acc;
                }

                acc[k] = m[k];
                return acc;
              }, {})}
          />
        );
      },
      enableSorting: false,
    },
  ];
};

export default function EventsPage() {
  return (
    <EventsProvider>
      <EventsContent />
    </EventsProvider>
  );
}
