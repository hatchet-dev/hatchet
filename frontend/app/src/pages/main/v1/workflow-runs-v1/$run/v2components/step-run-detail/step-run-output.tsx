import { RestrictedPayloads } from '@/components/v1/shared/restricted-payloads';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { queries, V1TaskStatus } from '@/lib/api';
import { formatErrorForDisplay } from '@/lib/utils';
import { useQuery } from '@tanstack/react-query';

export const V1StepRunOutput = (props: { taskRunId: string }) => {
  const { isLoading, data } = useQuery({
    ...queries.v1Tasks.get(props.taskRunId),
  });

  if (isLoading || !data) {
    return null;
  }

  if (data.payloadsRestricted) {
    return <RestrictedPayloads />;
  }

  if (data.status === V1TaskStatus.FAILED) {
    const { code, language } = formatErrorForDisplay(data.errorMessage || '');

    return (
      <CodeHighlighter
        className="my-4"
        language={language}
        code={code}
        copyCode={data.errorMessage}
      />
    );
  }

  return (
    <CodeHighlighter
      className="my-4"
      language="json"
      code={JSON.stringify(data.output, null, 2) || ''}
    />
  );
};
