import { CodeEditor } from '@/components/v1/ui/code-editor';
import { Loading } from '@/components/v1/ui/loading';

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
        copy={true}
        code={JSON.stringify(
          errors.length > 0
            ? errors.map((error) => error.split('\\n')).flat()
            : JSON.parse(output),
          null,
          2,
        )}
      />
    </>
  );
};
