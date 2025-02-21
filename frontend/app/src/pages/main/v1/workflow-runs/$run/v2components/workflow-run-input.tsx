import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

export function WorkflowRunInputDialog({ input }: { input: object }) {
  return (
    <CodeHighlighter
      className="my-4"
      language="json"
      code={JSON.stringify(input, null, 2)}
    />
  );
}
