import { CodeEditor } from '@/components/ui/code-editor';
import { Loading } from '@/components/ui/loading';

export interface StepRunOutputProps {
  output: string;
  isLoading: boolean;
  errors: string[];
}

export const StepRunOutput: React.FC<StepRunOutputProps> = ({
  output,
  isLoading,
  errors,
}) => {
  if (isLoading) {
    return <Loading />;
  }

  return (
    <>
      <CodeEditor
        language="json"
        className="mb-4"
        height="400px"
        code={JSON.stringify(
          errors.length > 0 ? errors : JSON.parse(output),
          null,
          2,
        )}
      />
    </>
  );
};
