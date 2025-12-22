import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { queries, V1TaskStatus } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export const V1StepRunOutput = (props: { taskRunId: string }) => {
  const { isLoading, data } = useQuery({
    ...queries.v1Tasks.get(props.taskRunId),
  });

  if (isLoading || !data) {
    return null;
  }

  const outputData =
    (data.status === V1TaskStatus.FAILED
      ? data.errorMessage
      : JSON.stringify(data.output, null, 2)) || '';

  return (
    <CodeHighlighter
      className="my-4 h-[400px] max-h-[400px] overflow-y-auto"
      language="json"
      maxHeight="400px"
      minHeight="400px"
      code={outputData}
    />
  );
};
