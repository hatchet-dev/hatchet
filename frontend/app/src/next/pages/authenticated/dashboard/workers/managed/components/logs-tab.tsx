import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { FC, useState, useEffect } from 'react';
import { Input } from '@/next/components/ui/input';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/next/components/ui/button';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { LogLine } from '@/lib/api/generated/cloud/data-contracts';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { ListCloudLogsQuery } from '@/lib/api/queries';

export const LogsTab: FC = () => {
  const { logs: activity } = useManagedComputeDetail();
  const [queryParams, setQueryParams] = useState<ListCloudLogsQuery>({});
  const [beforeInput, setBeforeInput] = useState<Date | undefined>();
  const [afterInput, setAfterInput] = useState<Date | undefined>();
  const [searchInput, setSearchInput] = useState<string>('');
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number | undefined>();
  const [mergedLogs, setMergedLogs] = useState<LogLine[]>([]);
  const [rotate, setRotate] = useState(false);

  useEffect(() => {
    const logs = activity?.data?.rows || [];

    if (!lastUpdatedAt && activity?.isSuccess) {
      setLastUpdatedAt(activity.dataUpdatedAt);
      setMergedLogs(logs);
    } else if (
      activity?.isSuccess &&
      activity.dataUpdatedAt !== lastUpdatedAt
    ) {
      setLastUpdatedAt(activity.dataUpdatedAt);

      setMergedLogs((prevLogs) => {
        return mergeLogs(prevLogs, logs);
      });
    }
  }, [
    lastUpdatedAt,
    activity?.isSuccess,
    activity?.data,
    activity?.dataUpdatedAt,
  ]);

  const handleBottomReached = async () => {
    if (
      activity?.isSuccess &&
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
      activity?.isSuccess &&
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
      before: beforeInput?.toISOString(),
      after: afterInput?.toISOString(),
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
};

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

  return uniqueLogs;
};
