import {
  columns,
  StreamTopicColumn,
  topicKey,
} from './components/stream-topic-columns';
import { useStreamTopics } from './hooks/use-stream-topics';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { docsPages } from '@/lib/generated/docs';
import { VisibilityState } from '@tanstack/react-table';
import { isAxiosError } from 'axios';
import { useMemo } from 'react';

const streamsDocPage =
  docsPages.reference.typescript['feature-clients'].streams;

export default function Streams() {
  const [columnVisibility, setColumnVisibility] =
    useLocalStorageState<VisibilityState>('hatchet:columns:stream-topics', {});

  const {
    data,
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

  if (isAxiosError(error) && error.response?.status === 403) {
    return (
      <EmptyState
        title="Durable streams are not enabled"
        description="Durable streams are not enabled for this tenant."
        docPage={streamsDocPage}
        docLabel="Learn about durable streams"
      />
    );
  }

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
