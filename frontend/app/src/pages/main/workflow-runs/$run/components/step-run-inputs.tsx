import { CodeEditor } from '@/components/ui/code-editor';
import { JsonForm } from '@/components/ui/json-form';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export interface StepRunOutputProps {
  input: string;
  setInput: (input: string) => void;
  disabled: boolean;
  handleOnPlay: () => void;
}

export const StepRunInputs: React.FC<StepRunOutputProps> = ({
  input,
  disabled,
  handleOnPlay,
  setInput,
}) => {
  return (
    <Tabs defaultValue="output" className="w-full">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="form">Form</TabsTrigger>
        <TabsTrigger value="code">Code</TabsTrigger>
      </TabsList>
      <TabsContent value="form">
        <JsonForm
          json={JSON.parse(input)}
          setInput={setInput}
          onSubmit={handleOnPlay}
          disabled={disabled}
        />
      </TabsContent>
      <TabsContent value="code">
        <CodeEditor
          language="json"
          className="my-4"
          height="400px"
          code={JSON.stringify(JSON.parse(input), null, 2)}
          setCode={(code: string | undefined) => {
            if (!code) return;
            setInput(code);
          }}
        />
      </TabsContent>
    </Tabs>
  );
};
