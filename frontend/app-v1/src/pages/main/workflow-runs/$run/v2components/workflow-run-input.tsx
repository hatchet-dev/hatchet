import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { Loading } from '@/components/ui/loading';
import { WorkflowRun, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export function WorkflowRunInputDialog({ run }: { run: WorkflowRun }) {
  const getInputQuery = useQuery({
    ...queries.workflowRuns.getInput(run.tenantId, run.metadata.id),
  });

  if (getInputQuery.isLoading) {
    return <Loading />;
  }

  if (!getInputQuery.data) {
    return null;
  }

  const input = getInputQuery.data;

  return (
    <CodeHighlighter
      className="my-4"
      language="json"
      code={JSON.stringify(input, null, 2)}
    />
  );
}
