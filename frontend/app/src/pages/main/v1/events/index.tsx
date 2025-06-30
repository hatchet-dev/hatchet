import { columns } from './components/event-columns';
import { Separator } from '@/components/v1/ui/separator';
import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import api, {
  Event,
  EventOrderByDirection,
  EventOrderByField,
  V1TaskStatus,
  queries,
} from '@/lib/api';
import {
  FilterOption,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { Loading } from '@/components/v1/ui/loading.tsx';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { TaskRunsTable } from '../workflow-runs-v1/components/task-runs-table';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export default function Events() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
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
  const { tenantId } = useCurrentTenantId();

  const [searchParams, setSearchParams] = useSearchParams();
  const [rotate, setRotate] = useState(false);

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

  const [sorting, setSorting] = useState<SortingState>(() => {
    const sortParam = searchParams.get('sort');
    if (sortParam) {
      const [id, desc] = sortParam.split(':');
      return [{ id, desc: desc === 'desc' }];
    }
    return [];
  });
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    const filtersParam = searchParams.get('filters');
    if (filtersParam) {
      return JSON.parse(filtersParam);
    }
    return [];
  });

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    EventId: false,
    Payload: false,
    scope: false,
  });

  const [pagination, setPagination] = useState<PaginationState>(() => {
    const pageIndex = Number(searchParams.get('pageIndex')) || 0;
    const pageSize = Number(searchParams.get('pageSize')) || 50;
    return { pageIndex, pageSize };
  });
  const [pageSize, setPageSize] = useState<number>(
    Number(searchParams.get('pageSize')) || 50,
  );
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  useEffect(() => {
    const newSearchParams = new URLSearchParams(searchParams);

    newSearchParams.set(
      'sort',
      sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(','),
    );
    newSearchParams.set('filters', JSON.stringify(columnFilters));
    newSearchParams.set('pageIndex', pagination.pageIndex.toString());
    newSearchParams.set('pageSize', pagination.pageSize.toString());
    setSearchParams(newSearchParams);
  }, [sorting, columnFilters, pagination, setSearchParams, searchParams]);

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

  const workflows = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'workflows');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const scopes = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'scope');

    if (!filter) {
      return [];
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const statuses = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'status');

    if (!filter) {
      return;
    }

    return filter?.value as Array<V1TaskStatus>;
  }, [columnFilters]);

  const eventIds = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'EventId');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const AdditionalMetadataFilter = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Metadata');

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

  const {
    data,
    isLoading: eventsIsLoading,
    refetch,
    error: eventsError,
  } = useQuery({
    queryKey: [
      'v1:events:list',
      tenantId,
      {
        keys,
        workflows,
        orderByField,
        orderByDirection,
        offset,
        limit: pageSize,
        statuses,
        additionalMetadata: AdditionalMetadataFilter,
        eventIds,
      },
    ],
    queryFn: async () => {
      const response = await api.v1EventList(tenantId, {
        offset,
        limit: pageSize,
        keys,
        since: undefined,
        until: undefined,
        eventIds,
        workflowRunStatuses: statuses,
        additionalMetadata: AdditionalMetadataFilter,
        workflowIds: workflows,
        scopes,
      });

      return response.data;
    },
    refetchInterval: 2000,
  });

  const {
    data: eventKeys,
    isLoading: eventKeysIsLoading,
    error: eventKeysError,
  } = useQuery({
    queryKey: ['v1:events:listKeys', tenantId],
    queryFn: async () => {
      const response = await api.v1EventKeyList(tenantId);

      return response.data;
    },
  });

  const eventKeyFilters = useMemo((): FilterOption[] => {
    return (
      eventKeys?.rows?.map((key) => ({
        value: key,
        label: key,
      })) || []
    );
  }, [eventKeys]);

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowRunStatusFilters = useMemo((): FilterOption[] => {
    return [
      {
        value: V1TaskStatus.COMPLETED,
        label: 'Succeeded',
      },
      {
        value: V1TaskStatus.FAILED,
        label: 'Failed',
      },
      {
        value: V1TaskStatus.RUNNING,
        label: 'Running',
      },
      {
        value: V1TaskStatus.QUEUED,
        label: 'Queued',
      },
      {
        value: V1TaskStatus.CANCELLED,
        label: 'Cancelled',
      },
    ];
  }, []);

  const tableColumns = columns({
    onRowClick: (row: Event) => {
      setSelectedEvent(row);
    },
  });

  const actions = [
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        refetch();
        setRotate(!rotate);
      }}
      variant={'outline'}
      aria-label="Refresh events list"
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
        error={eventsError || eventKeysError || workflowKeysError}
        isLoading={
          eventsIsLoading || eventKeysIsLoading || workflowKeysIsLoading
        }
        columns={tableColumns}
        data={data?.rows || []}
        filters={[
          {
            columnId: 'key',
            title: 'Key',
            options: eventKeyFilters,
          },
          {
            columnId: 'workflows',
            title: 'Task',
            options: workflowKeyFilters,
          },
          {
            columnId: 'status',
            title: 'Status',
            options: workflowRunStatusFilters,
          },
          {
            columnId: 'Metadata',
            title: 'Metadata',
            type: ToolbarType.KeyValue,
          },
          {
            columnId: 'EventId',
            title: 'Event Id',
            type: ToolbarType.Array,
          },
          {
            columnId: 'scope',
            title: 'Scope',
            type: ToolbarType.Array,
          },
        ]}
        showColumnToggle={true}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        actions={actions}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={data?.pagination?.num_pages || 0}
        rowSelection={rowSelection}
        setRowSelection={setRowSelection}
        getRowId={(row) => row.metadata.id}
      />
    </>
  );
}

function ExpandedEventContent({ event }: { event: Event }) {
  return (
    <DialogContent className="w-fit max-w-[700px] max-h-[85%] overflow-auto">
      <DialogHeader>
        <DialogTitle>Event {event.key}</DialogTitle>
        <DialogDescription>
          Seen <RelativeDate date={event.metadata.createdAt} />
        </DialogDescription>
      </DialogHeader>

      <h3 className="text-lg font-bold leading-tight text-foreground">
        Event Data
      </h3>
      <Separator />
      <EventDataSection event={event} />
      <h3 className="text-lg font-bold leading-tight text-foreground">Runs</h3>
      <Separator />
      <EventWorkflowRunsList event={event} />
    </DialogContent>
  );
}

function EventDataSection({ event }: { event: Event }) {
  const { tenantId } = useCurrentTenantId();

  const { data, isLoading } = useQuery({
    queryKey: ['v1:events:list', tenantId, event.metadata.id],
    queryFn: async () => {
      const response = await api.v1EventList(tenantId, {
        eventIds: [event.metadata.id],
      });

      return response.data;
    },
    refetchInterval: 2000,
  });

  if (isLoading || !data) {
    return <Loading />;
  }

  const eventData = data.rows?.at(0);

  if (!eventData) {
    return <div className="text-red-500">Event data not found</div>;
  }

  const dataToDisplay = {
    id: eventData.metadata.id,
    seenAt: eventData.seenAt,
    key: eventData.key,
    additionalMetadata: eventData.additionalMetadata,
    scope: eventData.scope,
    payload: eventData.payload,
  };

  return (
    <CodeHighlighter
      language="json"
      className="my-4"
      code={JSON.stringify(dataToDisplay, null, 2)}
    />
  );
}

function EventWorkflowRunsList({ event }: { event: Event }) {
  return (
    <div className="w-full overflow-x-auto max-w-full">
      <TaskRunsTable
        triggeringEventExternalId={event.metadata.id}
        showMetrics={false}
        showCounts={false}
      />
    </div>
  );
}
