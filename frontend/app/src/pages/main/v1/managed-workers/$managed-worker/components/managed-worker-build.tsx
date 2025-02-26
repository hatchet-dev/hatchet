import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { Spinner } from '@/components/v1/ui/loading';
import { ManagedWorkerBuildLogs } from './managed-worker-build-logs';
import RelativeDate from '@/components/v1/molecules/relative-date';

export function ManagedWorkerBuild({ buildId }: { buildId: string }) {
  const getBuildQuery = useQuery({
    ...queries.cloud.getBuild(buildId),
  });

  if (getBuildQuery.isLoading) {
    return <Spinner />;
  }

  const build = getBuildQuery.data;

  return (
    <div className="flex flex-col gap-4 w-full">
      <h4 className="text-lg font-semibold text-foreground">Build Overview</h4>
      <div className="flex flex-row gap-4">
        <div className="flex flex-col gap-2">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Build ID: {build?.metadata?.id}
          </div>
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Created: <RelativeDate date={build?.metadata?.createdAt} />
          </div>
          {build?.finishTime && (
            <div className="text-sm text-gray-700 dark:text-gray-300">
              Finished: <RelativeDate date={build?.finishTime} />
            </div>
          )}
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Status: {getBuildQuery.data?.status}
          </div>
        </div>
      </div>
      <h4 className="text-lg font-semibold text-foreground">Build Logs</h4>
      <ManagedWorkerBuildLogs buildId={buildId} />
    </div>
  );
}
