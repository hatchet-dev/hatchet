import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export function ManagedWorkerBuildLogs({ buildId }: { buildId: string }) {
  const { refetchInterval } = useRefetchInterval();

  const getBuildLogsQuery = useQuery({
    ...queries.cloud.getBuildLogs(buildId),
    refetchInterval,
  });

  const logs = getBuildLogsQuery.data?.rows || [
    {
      line: 'Loading...',
      timestamp: new Date().toISOString(),
      instance: 'Hatchet',
    },
  ];

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
