import { V1TaskStatus, V1TaskSummary, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
  const getLogsQuery = useQuery({
    ...queries.v1Tasks.getLogs(taskRun?.metadata.id || ''),
    enabled: !!taskRun,
    refetchInterval: () => {
      if (taskRun?.status === V1TaskStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  return (
    <div className="my-4">
      <LoggingComponent
        logs={
          getLogsQuery.data?.rows?.map((row) => ({
            timestamp: row.createdAt,
            line: row.message,
            instance: taskRun.displayName,
          })) || []
        }
        onTopReached={() => {}}
        onBottomReached={() => {}}
      />
    </div>
  );
}
