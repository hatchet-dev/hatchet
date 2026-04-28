import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

export function WorkflowRunInputDialog({ input }: { input: object }) {
  return (
    <CodeHighlighter
      className="flex-1 min-h-0 overflow-hidden"
      maxHeight="100%"
      minHeight="100%"
      language="json"
      code={JSON.stringify(input, null, 2)}
    />
  );
}
