import { StepRun } from '@/lib/api';
import { CodeEditor } from '@/components/ui/code-editor';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loading } from '@/components/ui/loading';

export function StepInputOutputSection({
  stepRun,
  onInputChanged,
}: {
  stepRun: StepRun;
  onInputChanged?: (input: string) => void;
}) {
  const input = stepRun.input || '{}';
  const output = stepRun.output || '{}';

  const isLoading =
    stepRun?.status != 'SUCCEEDED' &&
    stepRun?.status != 'FAILED' &&
    stepRun?.status != 'CANCELLED';

  return (
    <Tabs defaultValue="input" className="w-full">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="input">Input</TabsTrigger>
        <TabsTrigger value="output">Output</TabsTrigger>
      </TabsList>
      <TabsContent value="input">
        <CodeEditor
          language="json"
          className="my-4"
          height="400px"
          code={JSON.stringify(JSON.parse(input), null, 2)}
          setCode={(code: string | undefined) => {
            if (onInputChanged && code) {
              onInputChanged(code);
            }
          }}
        />
      </TabsContent>
      <TabsContent value="output">
        {isLoading && <Loading />}
        {!isLoading && (
          <CodeEditor
            language="json"
            className="my-4"
            height="400px"
            code={JSON.stringify(JSON.parse(output), null, 2)}
          />
        )}
      </TabsContent>
    </Tabs>
  );
}
