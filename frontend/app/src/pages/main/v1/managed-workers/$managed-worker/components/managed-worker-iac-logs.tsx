import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';

export function ManagedWorkerIaCLogs({
  managedWorkerId,
  deployKey,
}: {
  managedWorkerId: string;
  deployKey: string;
}) {
  const getBuildLogsQuery = useQuery({
    ...queries.cloud.getIacLogs(managedWorkerId, deployKey),
    refetchInterval: 5000,
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
