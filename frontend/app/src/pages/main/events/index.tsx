import { DataTable } from '../../../components/molecules/data-table/data-table';
import { columns } from './components/event-columns';
import { columns as workflowRunsColumns } from '../workflow-runs/components/workflow-runs-columns';
import { Separator } from '@/components/ui/separator';
import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
} from '@tanstack/react-table';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, {
  Event,
  EventOrderByDirection,
  EventOrderByField,
  ReplayEventRequest,
  queries,
} from '@/lib/api';
import invariant from 'tiny-invariant';
import { FilterOption } from '@/components/molecules/data-table/data-table-toolbar';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { relativeDate } from '@/lib/utils';
import { Code } from '@/components/ui/code';
import { useOutletContext, useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import {
  ArrowPathIcon,
  ArrowPathRoundedSquareIcon,
} from '@heroicons/react/24/outline';
import { useApiError } from '@/lib/hooks';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';

export default function Events() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Events
        </h2>
        <Separator className="my-4" />
        <EventsTable />
      </div>
    </div>
  );
}

function EventsTable() {
  const [selectedEvent, setSelectedEvent] = useState<Event | null>(null);
  const { tenant } = useOutletContext<TenantContextType>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [rotate, setRotate] = useState(false);
  const { handleApiError } = useApiError({});

  invariant(tenant);

  useEffect(() => {
    if (
      selectedEvent &&
      (!searchParams.get('event') ||
        searchParams.get('event') !== selectedEvent.metadata.id)
    ) {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.set('event', selectedEvent.metadata.id);
      setSearchParams(newSearchParams);
    } else if (
      !selectedEvent &&
      searchParams.get('event') &&
      searchParams.get('event') !== ''
    ) {
      const newSearchParams = new URLSearchParams(searchParams);
      newSearchParams.delete('event');
      setSearchParams(newSearchParams);
    }
  }, [selectedEvent, searchParams, setSearchParams]);

  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 50,
  });
  const [pageSize, setPageSize] = useState<number>(50);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const orderByDirection = useMemo((): EventOrderByDirection | undefined => {
    if (!sorting.length) {
      return;
    }

    return sorting[0]?.desc
      ? EventOrderByDirection.Desc
      : EventOrderByDirection.Asc;
  }, [sorting]);

  const orderByField = useMemo((): EventOrderByField | undefined => {
    if (!sorting.length) {
      return;
    }

    switch (sorting[0]?.id) {
      case 'Seen at':
      default:
        return EventOrderByField.CreatedAt;
    }
  }, [sorting]);

  const keys = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'key');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const listEventsQuery = useQuery({
    ...queries.events.list(tenant.metadata.id, {
      keys,
      orderByField,
      orderByDirection,
      offset,
      limit: pageSize,
    }),
  });

  const replayEventsMutation = useMutation({
    mutationKey: ['event:update:replay', tenant.metadata.id],
    mutationFn: async (data: ReplayEventRequest) => {
      await api.eventUpdateReplay(tenant.metadata.id, data);
    },
    onSuccess: () => {
      listEventsQuery.refetch();
    },
    onError: handleApiError,
  });

  const listEventKeysQuery = useQuery({
    ...queries.events.listKeys(tenant.metadata.id),
  });

  const eventKeyFilters = useMemo((): FilterOption[] => {
    return (
      listEventKeysQuery.data?.rows?.map((key) => ({
        value: key,
        label: key,
      })) || []
    );
  }, [listEventKeysQuery.data?.rows]);

  // useEffect(() => {
  //   if (listEventsQuery.data?.pagination) {
  //     setPagination({
  //       pageIndex: (listEventsQuery.data.pagination.current_page || 1) - 1,
  //       pageSize: listEventsQuery.data.pagination.num_pages || 0,
  //     });
  //   }
  // }, [listEventsQuery.data?.pagination]);

  if (listEventsQuery.isLoading) {
    return <Loading />;
  }

  const tableColumns = columns({
    onRowClick: (row: Event) => {
      setSelectedEvent(row);
    },
  });

  const actions = [
    <Button
      key="replay"
      disabled={Object.keys(rowSelection).length === 0}
      variant={Object.keys(rowSelection).length === 0 ? 'outline' : 'default'}
      size="sm"
      className="h-8 px-2 lg:px-3 gap-2"
      onClick={() => {
        replayEventsMutation.mutate({
          eventIds: Object.keys(rowSelection),
        });
      }}
    >
      <ArrowPathRoundedSquareIcon className="h-4 w-4" />
      Replay
    </Button>,
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        listEventsQuery.refetch();
        setRotate(!rotate);
      }}
      variant={'outline'}
    >
      <ArrowPathIcon
        className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
      />
    </Button>,
  ];

  return (
    <>
      <Dialog
        open={!!selectedEvent}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedEvent(null);
          }
        }}
      >
        {selectedEvent && <ExpandedEventContent event={selectedEvent} />}
      </Dialog>
      <DataTable
        columns={tableColumns}
        data={listEventsQuery.data?.rows || []}
        filters={[
          {
            columnId: 'key',
            title: 'Key',
            options: eventKeyFilters,
          },
        ]}
        actions={actions}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={listEventsQuery.data?.pagination?.num_pages || 0}
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        getRowId={(row) => row.metadata.id}
      />
    </>
  );
}

function ExpandedEventContent({ event }: { event: Event }) {
  return (
    <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
      <DialogHeader>
        <DialogTitle>Event {event.key}</DialogTitle>
        <DialogDescription>
          Seen {relativeDate(event.metadata.createdAt)}
        </DialogDescription>
      </DialogHeader>

      <h3 className="text-lg font-bold leading-tight text-foreground">
        Event Data
      </h3>
      <Separator />
      <EventDataSection event={event} />
      <h3 className="text-lg font-bold leading-tight text-foreground">
        Workflow Runs
      </h3>
      <Separator />
      <EventWorkflowRunsList event={event} />
    </DialogContent>
  );
}

function EventDataSection({ event }: { event: Event }) {
  const getEventDataQuery = useQuery({
    ...queries.events.getData(event.metadata.id),
  });

  if (getEventDataQuery.isLoading || !getEventDataQuery.data) {
    return <Loading />;
  }

  const eventData = getEventDataQuery.data;

  return (
    <>
      <Code language="json" className="my-4" maxHeight="400px">
        {JSON.stringify(JSON.parse(eventData.data), null, 2)}
      </Code>
    </>
  );
}

function EventWorkflowRunsList({ event }: { event: Event }) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset: 0,
      limit: 10,
      eventId: event.metadata.id,
    }),
  });

  return (
    <DataTable
      columns={workflowRunsColumns}
      data={listWorkflowRunsQuery.data?.rows || []}
      filters={[]}
      pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
      columnVisibility={{
        'Triggered by': false,
      }}
      isLoading={listWorkflowRunsQuery.isLoading}
    />
  );
}
