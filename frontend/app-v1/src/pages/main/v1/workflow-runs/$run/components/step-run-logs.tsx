import {
  LogLineOrderByDirection,
  StepRun,
  StepRunStatus,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { CodeEditor } from '@/components/v1/ui/code-editor';

export function StepRunLogs({ stepRun }: { stepRun: StepRun | undefined }) {
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

  const logLines =
    getLogsQuery.data?.rows?.reduce(
      (acc, row) => acc + row.createdAt + ': ' + row.message + '\n',
      '',
    ) || 'No logs found';

  return (
    <CodeEditor
      language="txt"
      height="400px"
      code={logLines}
      lineNumbers
      className="mb-4"
    />
  );
}
