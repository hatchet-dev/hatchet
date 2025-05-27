import { V1Event, V1TaskStatus } from '@/lib/api';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { DataTableColumnHeader } from '@/next/components/runs/runs-table/data-table-column-header';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { EventsProvider, useEvents } from '@/next/hooks/use-events';
import { RunsProvider } from '@/next/hooks/use-runs';
import docs from '@/next/lib/docs';
import { AdditionalMetadata } from '@/pages/main/v1/events/components/additional-metadata';
import { ColumnDef } from '@tanstack/react-table';

function EventsContent() {
  const { data, isLoading } = useEvents();

  if (isLoading) {
    return (
      <div className="flex-grow h-full w-full flex items-center justify-center">
        <p>Loading events</p>
      </div>
    );
  }

  // const eventKeys = Array.from(new Set(data.map((e) => e.key)))
  //   .sort((a, b) => a.localeCompare(b))
  //   .map((k) => ({
  //     label: k,
  //     value: k,
  //   }));

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
      {/* <FilterGroup>
        <div className="flex flex-row gap-x-4">
          <FilterSelect<EventsFilters, string>
            name="keys"
            placeholder="Event Key"
          />
          <ClearFiltersButton />
        </div>
      </FilterGroup> */}
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
      accessorKey: 'EventId',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="ID" className="pl-4" />
      ),
      cell: ({ row }) => (
        <div className="w-full">{row.original.metadata.id} </div>
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
          <div className="flex flex-row gap-2 items-center justify-start w-max">
            <StatusBadgeWithTooltip
              count={queued}
              eventExternalId={row.original.metadata.id}
              status={V1TaskStatus.QUEUED}
            />
            <StatusBadgeWithTooltip
              count={running}
              eventExternalId={row.original.metadata.id}
              status={V1TaskStatus.RUNNING}
            />
            <StatusBadgeWithTooltip
              count={cancelled}
              eventExternalId={row.original.metadata.id}
              status={V1TaskStatus.CANCELLED}
            />
            <StatusBadgeWithTooltip
              count={succeeded}
              eventExternalId={row.original.metadata.id}
              status={V1TaskStatus.COMPLETED}
            />
            <StatusBadgeWithTooltip
              count={failed}
              eventExternalId={row.original.metadata.id}
              status={V1TaskStatus.FAILED}
            />
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

const StatusBadgeWithTooltip = ({
  count,
  eventExternalId,
  status,
}: {
  count: number | undefined;
  eventExternalId: string;
  status: V1TaskStatus;
}) => {
  if (!count || count === 0) {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div>
          <RunsBadge status={status} variant="default" />
        </div>
      </TooltipTrigger>
      <TooltipContent className="bg-[hsl(var(--background))] border-slate-700 border z-20 shadow-lg p-4 text-white">
        <RunsProvider
          initialFilters={{
            triggering_event_external_id: eventExternalId,
          }}
        >
          <RunsTable
            excludedFilters={[
              'additional_metadata',
              'only_tasks',
              'is_root_task',
              'parent_task_external_id',
              'statuses',
              'triggering_event_external_id',
              'worker_id',
              'workflow_ids',
            ]}
            showPagination={false}
            allowSelection={false}
            showActions={false}
          />
        </RunsProvider>
      </TooltipContent>
    </Tooltip>
  );
};
