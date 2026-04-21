import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries, V1LogLineLevel } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export function ManagedWorkerIaCLogs({
  managedWorkerId,
  deployKey,
}: {
  managedWorkerId: string;
  deployKey: string;
}) {
  const { refetchInterval } = useRefetchInterval();

  const getBuildLogsQuery = useQuery({
    ...queries.cloud.getIacLogs(managedWorkerId, deployKey),
    refetchInterval,
  });

  const logs: LogLine[] = useMemo(() => {
    const cloudLogs = getBuildLogsQuery.data?.rows || [];
    if (cloudLogs.length === 0) {
      return [
        {
          line: 'Loading...',
          timestamp: new Date().toISOString(),
          instance: 'Hatchet',
        },
      ];
    }
    return cloudLogs.map((log) => ({
      timestamp: log.timestamp,
      line: log.line,
      instance: log.instance,
      // fixme: this should use the v1 log type, not the v0 one
      level: log.level as V1LogLineLevel | undefined,
      metadata: log.metadata as Record<string, unknown> | undefined,
    }));
  }, [getBuildLogsQuery.data?.rows]);

  return (
    <div className="w-full flex flex-col max-h-[25rem] min-h-[25rem]">
      <LogViewer logs={logs} />
    </div>
  );
}
