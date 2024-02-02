import { StepRun } from '@/lib/api';
import { CodeEditor } from '@/components/ui/code-editor';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export function StepInputOutputSection({ stepRun }: { stepRun: StepRun }) {
  const input = stepRun.input || '{}';
  const output = stepRun.output || '{}';

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
        />
      </TabsContent>
      <TabsContent value="output">
        <CodeEditor
          language="json"
          className="my-4"
          height="400px"
          code={JSON.stringify(JSON.parse(output), null, 2)}
        />
      </TabsContent>
    </Tabs>
  );
}
