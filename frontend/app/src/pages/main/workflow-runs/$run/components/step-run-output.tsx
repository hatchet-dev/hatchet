import { CodeEditor } from '@/components/ui/code-editor';
import { Loading } from '@/components/ui/loading';
import {
  StepConfigurationSection,
  StepDurationSection,
  StepStatusSection,
} from '..';
import { StepRun } from '@/lib/api';

export interface StepRunOutputProps {
  stepRun: StepRun;
  output: string;
  isLoading: boolean;
  errors: string[];
}

export const StepRunOutput: React.FC<StepRunOutputProps> = ({
  stepRun,
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
      <StepStatusSection stepRun={stepRun} />
      <StepDurationSection stepRun={stepRun} />
      <StepConfigurationSection stepRun={stepRun} />
    </>
  );
};
