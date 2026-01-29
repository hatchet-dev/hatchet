import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

export function ManagedWorkerBuildLogs({ buildId }: { buildId: string }) {
  const { refetchInterval } = useRefetchInterval();

  const getBuildLogsQuery = useQuery({
    ...queries.cloud.getBuildLogs(buildId),
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
      level: log.level,
      metadata: log.metadata as Record<string, unknown> | undefined,
    }));
  }, [getBuildLogsQuery.data?.rows]);

  return (
    <div className="w-full">
      <LogViewer logs={logs} />
    </div>
  );
}
