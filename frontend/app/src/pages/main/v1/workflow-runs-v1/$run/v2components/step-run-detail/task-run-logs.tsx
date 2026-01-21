import { LogSearchInput } from '@/components/v1/cloud/logging/log-search/log-search-input';
import { useLogs } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { V1TaskSummary } from '@/lib/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';

export function TaskRunLogs({
  taskRun,
  resetTrigger,
}: {
  taskRun: V1TaskSummary;
  resetTrigger?: number;
}) {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });
  const { logs, queryString, setQueryString, handleScroll } = useLogs({
    taskRun,
    resetTrigger,
  });
  const { featureFlags } = useCloud(tenantId);
  const isLogSearchEnabled = featureFlags?.enable_log_search === 'true';

  return (
    <div className="my-4 flex flex-col gap-y-2">
      {isLogSearchEnabled && (
        <LogSearchInput value={queryString} onChange={setQueryString} />
      )}
      <LogViewer logs={logs} onScroll={handleScroll} />
    </div>
  );
}
