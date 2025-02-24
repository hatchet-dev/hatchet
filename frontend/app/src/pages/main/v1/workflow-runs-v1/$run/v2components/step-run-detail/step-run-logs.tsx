import {
  LogLineOrderByDirection,
  StepRun,
  StepRunStatus,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import LoggingComponent from '@/components/v1/cloud/logging/logs';

export function StepRunLogs({
  stepRun,
  readableId,
}: {
  stepRun: StepRun | undefined;
  readableId: string;
}) {
  const getLogsQuery = useQuery({
    ...queries.stepRuns.getLogs(stepRun?.metadata.id || '', {
      orderByDirection: LogLineOrderByDirection.Asc,
    }),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
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
            instance: readableId,
          })) || []
        }
        onTopReached={() => {}}
        onBottomReached={() => {}}
      />
    </div>
  );
}
