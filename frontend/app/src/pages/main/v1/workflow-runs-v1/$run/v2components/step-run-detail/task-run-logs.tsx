import { LogSearchInput } from '@/components/v1/cloud/logging/log-search/log-search-input';
import { useLogs } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { V1TaskSummary } from '@/lib/api';

export function TaskRunLogs({
  taskRun,
  resetTrigger,
}: {
  taskRun: V1TaskSummary;
  resetTrigger?: number;
}) {
  const { logs, queryString, setQueryString, handleScroll } = useLogs({
    taskRun,
    resetTrigger,
  });

  return (
    <div className="my-4 flex flex-col gap-y-2">
      <LogSearchInput value={queryString} onChange={setQueryString} />
      <LogViewer logs={logs} onScroll={handleScroll} />
    </div>
  );
}
