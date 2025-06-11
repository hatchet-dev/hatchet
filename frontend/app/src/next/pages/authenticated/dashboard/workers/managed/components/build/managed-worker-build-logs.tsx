import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import LoggingComponent from '@/components/v1/cloud/logging/logs';

export function ManagedWorkerBuildLogs({ buildId }: { buildId: string }) {
  const getBuildLogsQuery = useQuery({
    ...queries.cloud.getBuildLogs(buildId),
    refetchInterval: 5000,
  });

  const fallbackLogs = useMemo(
    () => [
      {
        line: 'Loading...',
        timestamp: new Date().toISOString(),
        instance: 'Hatchet',
      },
    ],
    [],
  );

  const logs = getBuildLogsQuery.data?.rows || fallbackLogs;

  return (
    <div className="w-full">
      <LoggingComponent
        logs={logs}
        onBottomReached={() => {}}
        onTopReached={() => {}}
      />
    </div>
  );
}
