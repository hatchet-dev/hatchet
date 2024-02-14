import { CodeEditor } from '@/components/ui/code-editor';
import { Loading } from '@/components/ui/loading';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StepConfigurationSection, StepStatusSection } from '..';
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
    <Tabs defaultValue="output" className="w-full">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="output">Output</TabsTrigger>
        <TabsTrigger value="details">Details</TabsTrigger>
        {/* <TabsTrigger value="eval">Eval</TabsTrigger> */}
        {/* <TabsTrigger value="timing">Timing</TabsTrigger> */}
      </TabsList>
      <TabsContent value="output">
        <CodeEditor
          language="json"
          className="my-4"
          height="400px"
          code={JSON.stringify(
            errors.length > 0 ? errors : JSON.parse(output),
            null,
            2,
          )}
        />
      </TabsContent>
      <TabsContent value="details">
        <StepStatusSection stepRun={stepRun} />
        <StepConfigurationSection stepRun={stepRun} />
      </TabsContent>
      <TabsContent value="logs">Logs Coming Soon!</TabsContent>
      <TabsContent value="timing">Execution Timing Coming Soon!</TabsContent>
    </Tabs>
  );
};
