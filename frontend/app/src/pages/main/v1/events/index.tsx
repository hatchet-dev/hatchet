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
import { Separator } from '@/components/v1/ui/separator';
import { useMemo, useState } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { V1Event, V1Filter } from '@/lib/api';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { RunsTable } from '../workflow-runs-v1/components/runs-table';
import { RunsProvider } from '../workflow-runs-v1/hooks/runs-provider';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

import {
  FilterColumn,
  filterColumns,
} from '../filters/components/filter-columns';
import { useFilters } from '../filters/hooks/use-filters';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useEvents } from './hooks/use-events';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';

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
          <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
            <p className="text-lg font-semibold">No events found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['run-on-event']}
                size="full"
                variant="outline"
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
        <div className="flex flex-col justify-center items-start gap-3 pb-4 border-b text-sm">
          <div className="flex flex-row items-center gap-3 min-w-0 w-full">
            <span className="text-muted-foreground font-medium shrink-0">
              Key
            </span>
            <div className="px-2 py-1 overflow-x-auto min-w-0 flex-1">
              <span className="whitespace-nowrap">{event.key}</span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground font-medium">Seen</span>
            <span className="font-medium">
              <RelativeDate date={event.metadata.createdAt} />
            </span>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground mb-2">
              Payload
            </h3>
            <Separator className="mb-3" />
            <div className="max-h-96 overflow-y-auto rounded-lg">
              <EventDataSection event={event} />
            </div>
          </div>

          {hasScope && filters && filters.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-2">
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
            <h3 className="text-sm font-semibold text-foreground mb-2">Runs</h3>
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
  const columns = useMemo(
    () => filterColumns(workflowIdToName),
    [workflowIdToName],
  );

  return (
    <div className="w-full overflow-x-auto">
      <div className="min-w-[500px] [&_th:last-child]:w-[60px] [&_th:last-child]:min-w-[60px] [&_th:last-child]:max-w-[60px] [&_td:last-child]:w-[60px] [&_td:last-child]:min-w-[60px] [&_td:last-child]:max-w-[60px]">
        <DataTable
          columns={columns}
          data={filters}
          columnKeyToName={FilterColumn}
        />
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
