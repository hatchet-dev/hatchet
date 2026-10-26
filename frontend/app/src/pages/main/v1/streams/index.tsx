import {
  columns,
  StreamTopicColumn,
  topicKey,
} from './components/stream-topic-columns';
import { useStreamTopics } from './hooks/use-stream-topics';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { Loading } from '@/components/v1/ui/loading';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useTenantEntitlements } from '@/hooks/use-tenant-entitlements';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo } from 'react';

const streamsDocPage =
  docsPages.reference.typescript['feature-clients'].streams;

function formatRetention(hours: number) {
  if (hours % 24 === 0) {
    const days = hours / 24;
    return `${days} ${days === 1 ? 'day' : 'days'}`;
  }

  return `${hours} ${hours === 1 ? 'hour' : 'hours'}`;
}

export default function Streams() {
  const { entitlements, isLoading } = useTenantEntitlements();

  if (isLoading) {
    return <Loading />;
  }

  if (!entitlements?.durableStreams) {
    return (
      <EmptyState
        title="Durable streams are not enabled"
        description="Durable streams are not enabled for this tenant. Contact support to enable them."
        docPage={streamsDocPage}
        docLabel="Learn about durable streams"
      />
    );
  }

  return <StreamTopicsTable />;
}

function StreamTopicsTable() {
  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:stream-topics', {});

  const {
    data,
    retentionHours,
    error,
    isLoading,
    columnFilters,
    setColumnFilters,
    pagination,
    setPageSize,
    setPagination,
    numPages,
    isRefetching,
    isPlaceholderData,
    refetch,
    resetFilters,
  } = useStreamTopics({ key: 'stream-topics-table' });

  const tableColumns = useMemo(() => columns(), []);

  return (
    <DataTable
      error={error}
      isLoading={isLoading}
      isPlaceholderData={isPlaceholderData}
      columns={tableColumns}
      data={data}
      filters={[
        {
          columnId: topicKey,
          title: StreamTopicColumn.topic,
          type: ToolbarType.Search,
        },
      ]}
      rightActions={
        retentionHours !== undefined
          ? [
              <div
                key="retention"
                className="flex h-8 items-center whitespace-nowrap text-sm text-muted-foreground"
                title="Messages are kept for this long, and a topic is removed once nothing has been published to it for this long."
              >
                Retention:&nbsp;
                <span className="font-medium text-foreground">
                  {formatRetention(retentionHours)}
                </span>
              </div>,
            ]
          : []
      }
      showColumnToggle={true}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={numPages}
      getRowId={(row) => row.metadata.id}
      columnKeyToName={StreamTopicColumn}
      refetchProps={{
        isRefetching,
        onRefetch: refetch,
      }}
      onResetFilters={resetFilters}
      emptyState={
        <EmptyState
          title="No stream topics found"
          description="Topics are created the first time a message is published to them, and removed once no message has been published within the retention period."
          docPage={streamsDocPage}
          docLabel="Learn about durable streams"
        />
      }
    />
  );
}
