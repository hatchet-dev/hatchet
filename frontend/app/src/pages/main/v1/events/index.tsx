import { useFilters } from '../filters/hooks/use-filters';
import { RunsTable } from '../workflow-runs-v1/components/runs-table';
import { RunsProvider } from '../workflow-runs-v1/hooks/runs-provider';
import {
  columns,
  EventColumn,
  idKey,
  keyKey,
  metadataKey,
  scopeKey,
  statusKey,
  workflowKey,
} from './components/event-columns';
import { useEvents } from './hooks/use-events';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Separator } from '@/components/v1/ui/separator';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1Event, V1Filter } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';
import { CheckIcon } from 'lucide-react';
import { useMemo, useState } from 'react';

export default function Events() {
  const [openMetadataPopover, setOpenMetadataPopover] = useState<string | null>(
    null,
  );
  const [openPayloadPopover, setOpenPayloadPopover] = useState<string | null>(
    null,
  );
  const { open } = useSidePanel();

  const {
    events,
    numEvents,
    isLoading,
    refetch,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    eventKeyFilters,
    workflowKeyFilters,
    workflowRunStatusFilters,
    isRefetching,
    resetFilters,
  } = useEvents({
    key: 'table',
  });

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    [idKey]: false,
    [EventColumn.payload]: false,
    [scopeKey]: false,
  });

  const tableColumns = columns({
    onRowClick: (row: V1Event) => {
      open({
        type: 'event-details',
        content: {
          event: row,
        },
      });
    },
    openMetadataPopover,
    setOpenMetadataPopover,
    openPayloadPopover,
    setOpenPayloadPopover,
  });

  return (
    <>
      <DataTable
        error={error}
        isLoading={isLoading}
        columns={tableColumns}
        data={events}
        filters={[
          {
            columnId: keyKey,
            title: EventColumn.key,
            options: eventKeyFilters,
            type: ToolbarType.Array,
          },
          {
            columnId: workflowKey,
            title: EventColumn.workflowId,
            options: workflowKeyFilters,
            type: ToolbarType.Checkbox,
          },
          {
            columnId: statusKey,
            title: EventColumn.status,
            options: workflowRunStatusFilters,
            type: ToolbarType.Checkbox,
          },
          {
            columnId: metadataKey,
            title: EventColumn.metadata,
            type: ToolbarType.KeyValue,
          },
          {
            columnId: idKey,
            title: EventColumn.id,
            type: ToolbarType.Array,
          },
          {
            columnId: scopeKey,
            title: EventColumn.scope,
            type: ToolbarType.Array,
          },
        ]}
        showColumnToggle={true}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={numEvents}
        getRowId={(row) => row.metadata.id}
        columnKeyToName={EventColumn}
        showSelectedRows={false}
        refetchProps={{
          isRefetching,
          onRefetch: refetch,
        }}
        onResetFilters={resetFilters}
        emptyState={
          <div className="flex h-full w-full flex-col items-center justify-center gap-y-4 py-8 text-foreground">
            <p className="text-lg font-semibold">No events found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['run-on-event']}
                label="Learn about pushing events to Hatchet"
              />
            </div>
          </div>
        }
      />
    </>
  );
}

export function ExpandedEventContent({ event }: { event: V1Event }) {
  const hasScope = Boolean(event.scope && event.scope.length > 0);
  const { filters, workflowIdToName } = useFilters({
    key: 'events-table',
    scopeOverrides: event.scope ? [event.scope] : undefined,
  });

  return (
    <div className="w-full">
      <div className="space-y-6">
        <div className="flex flex-col items-start justify-center gap-3 border-b pb-4 text-sm">
          <div className="flex w-full min-w-0 flex-row items-center gap-3">
            <span className="shrink-0 font-medium text-muted-foreground">
              Key
            </span>
            <div className="min-w-0 flex-1 overflow-x-auto px-2 py-1">
              <span className="whitespace-nowrap">{event.key}</span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <span className="font-medium text-muted-foreground">Seen</span>
            <span className="font-medium">
              <RelativeDate date={event.metadata.createdAt} />
            </span>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <h3 className="mb-2 text-sm font-semibold text-foreground">
              Payload
            </h3>
            <Separator className="mb-3" />
            <div className="max-h-96 overflow-y-auto rounded-lg">
              <EventDataSection event={event} />
            </div>
          </div>

          {hasScope && filters && filters.length > 0 && (
            <div>
              <h3 className="mb-2 text-sm font-semibold text-foreground">
                Filters
              </h3>
              <Separator className="mb-3" />
              <FiltersSection
                filters={filters}
                workflowIdToName={workflowIdToName}
              />
            </div>
          )}

          <div>
            <h3 className="mb-2 text-sm font-semibold text-foreground">Runs</h3>
            <Separator className="mb-3" />
            <EventWorkflowRunsList event={event} />
          </div>
        </div>
      </div>
    </div>
  );
}

function EventDataSection({ event }: { event: V1Event }) {
  const dataToDisplay = {
    id: event.metadata.id,
    seenAt: event.seenAt,
    key: event.key,
    additionalMetadata: event.additionalMetadata,
    scope: event.scope,
    payload: event.payload,
  };

  return (
    <CodeHighlighter
      language="json"
      className="text-xs"
      code={JSON.stringify(dataToDisplay, null, 2)}
    />
  );
}

function FiltersSection({
  filters,
  workflowIdToName,
}: {
  filters: V1Filter[];
  workflowIdToName: Record<string, string>;
}) {
  const filterColumns = useMemo(
    () => [
      {
        columnLabel: 'ID',
        cellRenderer: (filter: V1Filter) => (
          <div className="w-full">
            <Button className="w-fit pl-0" variant="link">
              {filter.metadata.id}
            </Button>
          </div>
        ),
      },
      {
        columnLabel: 'Workflow',
        cellRenderer: (filter: V1Filter) => (
          <div className="w-full">{workflowIdToName[filter.workflowId]}</div>
        ),
      },
      {
        columnLabel: 'Scope',
        cellRenderer: (filter: V1Filter) => (
          <div className="w-full">{filter.scope}</div>
        ),
      },
      {
        columnLabel: 'Expression',
        cellRenderer: (filter: V1Filter) => (
          <CodeHighlighter
            language="text"
            className="whitespace-pre-wrap break-words text-sm leading-relaxed"
            code={filter.expression}
            copy={false}
            maxHeight="10rem"
            minWidth="20rem"
          />
        ),
      },
      {
        columnLabel: 'Is Declarative',
        cellRenderer: (filter: V1Filter) =>
          filter.isDeclarative ? (
            <CheckIcon className="size-4 text-green-600" />
          ) : null,
      },
    ],
    [workflowIdToName],
  );

  return (
    <div className="w-full overflow-x-auto">
      <div className="min-w-[500px] [&_td:last-child]:w-[60px] [&_td:last-child]:min-w-[60px] [&_td:last-child]:max-w-[60px] [&_th:last-child]:w-[60px] [&_th:last-child]:min-w-[60px] [&_th:last-child]:max-w-[60px]">
        {filters.length > 0 ? (
          <SimpleTable columns={filterColumns} data={filters} />
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            No filters found for this event.
          </div>
        )}
      </div>
    </div>
  );
}

function EventWorkflowRunsList({ event }: { event: V1Event }) {
  return (
    <div className="w-full overflow-x-auto">
      <div className="min-w-[600px]">
        <RunsProvider
          tableKey={`event-workflow-runs-${event.metadata.id}`}
          display={{
            hideMetrics: true,
            hideCounts: true,
            hideDateFilter: true,
            hideTriggerRunButton: true,
            hideCancelAndReplayButtons: true,
          }}
          runFilters={{
            triggeringEventExternalId: event.metadata.id,
          }}
        >
          <RunsTable />
        </RunsProvider>
      </div>
    </div>
  );
}
