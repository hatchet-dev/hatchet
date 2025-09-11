import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import {
  LogLine,
  ManagedWorker,
} from '@/lib/api/generated/cloud/data-contracts';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useState, useEffect } from 'react';
import { Input } from '@/components/v1/ui/input';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { ListCloudLogsQuery } from '@/lib/api/queries';

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

  const getLogsQuery = useQuery({
    ...queries.cloud.getManagedWorkerLogs(
      managedWorker?.metadata.id || '',
      queryParams,
    ),
    enabled: !!managedWorker,
    refetchInterval: 15000,
  });

  useEffect(() => {
    const logs = getLogsQuery.data?.rows || [];

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
    if (
      getLogsQuery.isSuccess &&
      lastUpdatedAt &&
      // before input should be before the last log in the list
      (!beforeInput ||
        beforeInput?.toISOString() >
          mergedLogs[mergedLogs.length - 1]?.timestamp)
    ) {
      setQueryParams({
        ...queryParams,
        before: beforeInput?.toISOString(),
        after: mergedLogs[mergedLogs.length - 1]?.timestamp,
        direction: 'forward',
      });
    }
  };

  const handleTopReached = async () => {
    if (
      getLogsQuery.isSuccess &&
      lastUpdatedAt &&
      // after input should be before the first log in the list
      (!afterInput || afterInput?.toISOString() < mergedLogs[0]?.timestamp)
    ) {
      setQueryParams({
        ...queryParams,
        before: mergedLogs[0]?.timestamp,
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
      <div className="flex flex-row justify-between items-center">
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
              className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
          </Button>
        </div>
      </div>
      <LoggingComponent
        logs={mergedLogs}
        onBottomReached={handleBottomReached}
        onTopReached={handleTopReached}
      />
    </div>
  );
}

const mergeLogs = (existingLogs: LogLine[], newLogs: LogLine[]): LogLine[] => {
  const combinedLogs = [...existingLogs, ...newLogs];
  const uniqueLogs = Array.from(
    new Map(
      combinedLogs.map((log) => [log.timestamp + log.instance + log.line, log]),
    ).values(),
  );

  // sort logs by timestamp with collisions resolved by log line
  uniqueLogs.sort((a, b) => {
    if (a.timestamp === b.timestamp) {
      return a.line < b.line ? -1 : 1;
    }
    return a.timestamp < b.timestamp ? -1 : 1;
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
