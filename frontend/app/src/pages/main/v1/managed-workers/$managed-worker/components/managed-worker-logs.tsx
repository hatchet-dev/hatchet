import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import { ManagedWorker } from '@/lib/api/generated/cloud/data-contracts';
import { ListCloudLogsQuery } from '@/lib/api/queries';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useState, useEffect } from 'react';

export function ManagedWorkerLogs({
  managedWorker,
}: {
  managedWorker: ManagedWorker;
}) {
  const [queryParams, setQueryParams] = useState<ListCloudLogsQuery>({});
  const [beforeInput, setBeforeInput] = useState<Date | undefined>();
  const [afterInput, setAfterInput] = useState<Date | undefined>();
  const [searchInput, setSearchInput] = useState<string>('');
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number | undefined>();
  const [mergedLogs, setMergedLogs] = useState<LogLine[]>([]);
  const [rotate, setRotate] = useState(false);
  const { refetchInterval } = useRefetchInterval();

  const getLogsQuery = useQuery({
    ...queries.cloud.getManagedWorkerLogs(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval,
  });

  useEffect(() => {
    const cloudLogs = getLogsQuery.data?.rows || [];
    const logs: LogLine[] = cloudLogs.map((log) => ({
      timestamp: log.timestamp,
      line: log.line,
      instance: log.instance,
      level: log.level,
      metadata: log.metadata as Record<string, unknown> | undefined,
    }));

    if (!lastUpdatedAt && getLogsQuery.isSuccess) {
      setLastUpdatedAt(getLogsQuery.dataUpdatedAt);
      setMergedLogs(logs);
    } else if (
      getLogsQuery.isSuccess &&
      getLogsQuery.dataUpdatedAt !== lastUpdatedAt
    ) {
      setLastUpdatedAt(getLogsQuery.dataUpdatedAt);

      setMergedLogs((prevLogs) => {
        return mergeLogs(prevLogs, logs);
      });
    }
  }, [
    lastUpdatedAt,
    getLogsQuery.isSuccess,
    getLogsQuery.data,
    getLogsQuery.dataUpdatedAt,
  ]);

  const handleBottomReached = async () => {
    const lastLog = mergedLogs[mergedLogs.length - 1];
    if (
      getLogsQuery.isSuccess &&
      lastUpdatedAt &&
      lastLog?.timestamp &&
      // before input should be before the last log in the list
      (!beforeInput || beforeInput?.toISOString() > lastLog.timestamp)
    ) {
      setQueryParams({
        ...queryParams,
        before: beforeInput?.toISOString(),
        after: lastLog.timestamp,
        direction: 'forward',
      });
    }
  };

  const handleTopReached = async () => {
    const firstLog = mergedLogs[0];
    if (
      getLogsQuery.isSuccess &&
      lastUpdatedAt &&
      firstLog?.timestamp &&
      // after input should be before the first log in the list
      (!afterInput || afterInput?.toISOString() < firstLog.timestamp)
    ) {
      setQueryParams({
        ...queryParams,
        before: firstLog.timestamp,
        after: afterInput?.toISOString(),
        direction: 'backward',
      });
    }
  };

  const refreshLogs = () => {
    setMergedLogs([]);
    setQueryParams({
      ...queryParams,
      before: beforeInput && beforeInput.toISOString(),
      after: afterInput && afterInput.toISOString(),
    });
    setRotate(!rotate);
  };

  const datesMatchSearch =
    beforeInput?.toISOString() === queryParams?.before &&
    afterInput?.toISOString() === queryParams?.after;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-row items-center justify-between">
        <form
          onSubmit={() =>
            setQueryParams({
              ...queryParams,
              search: searchInput,
            })
          }
        >
          <Input
            id="search-input"
            placeholder="Search..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="h-8 w-[150px] lg:w-[250px]"
          />
          {/* hidden button for submitting input */}
          <button type="submit" className="hidden" formTarget="search-input" />
        </form>
        <div className="flex flex-row gap-4">
          <DateTimePicker
            date={afterInput}
            setDate={setAfterInput}
            label="After"
          />
          <DateTimePicker
            date={beforeInput}
            setDate={setBeforeInput}
            label="Before"
          />
          <Button
            key="refresh"
            className="h-8 px-2 lg:px-3"
            size="sm"
            onClick={refreshLogs}
            variant={datesMatchSearch ? 'outline' : 'default'}
            aria-label="Refresh logs"
          >
            <ArrowPathIcon
              className={`size-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
          </Button>
        </div>
      </div>
      <LogViewer
        logs={mergedLogs}
        onScrollToBottom={handleBottomReached}
        onScrollToTop={handleTopReached}
      />
    </div>
  );
}

const mergeLogs = (existingLogs: LogLine[], newLogs: LogLine[]): LogLine[] => {
  const combinedLogs = [...existingLogs, ...newLogs];
  const uniqueLogs = Array.from(
    new Map(
      combinedLogs.map((log) => [
        (log.timestamp || '') + (log.instance || '') + (log.line || ''),
        log,
      ]),
    ).values(),
  );

  // sort logs by timestamp with collisions resolved by log line
  uniqueLogs.sort((a, b) => {
    if (a.timestamp === b.timestamp) {
      return (a.line || '') < (b.line || '') ? -1 : 1;
    }
    return (a.timestamp || '') < (b.timestamp || '') ? -1 : 1;
  });

  // NOTE: this was used to truncate log lines to 300, but was causing issues with the scroll position
  // in the LoggingComponent. I've left this here in case we want to revisit this in the future.
  //   if (uniqueLogs.length > 300) {
  //     // if favoring forward, truncate the newest logs
  //     if (favoredDirection === 'forward') {
  //       return uniqueLogs.slice(-300);
  //     }

  //     return uniqueLogs.slice(0, 300);
  //   }

  return uniqueLogs;
};
