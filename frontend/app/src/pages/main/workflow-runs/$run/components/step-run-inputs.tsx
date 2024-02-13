import { CodeEditor } from '@/components/ui/code-editor';
import { JsonForm } from '@/components/ui/json-form';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { VscNote, VscJson } from 'react-icons/vsc';

export interface StepRunOutputProps {
  input: string;
  schema: string;
  setInput: (input: string) => void;
  disabled: boolean;
  handleOnPlay: () => void;
}

export const StepRunInputs: React.FC<StepRunOutputProps> = ({
  input,
  schema,
  disabled,
  handleOnPlay,
  setInput,
}) => {
  return (
    <Tabs defaultValue="form" className="w-full">
      <TabsList className="grid w-1/3 grid-cols-2">
        <TabsTrigger value="form" aria-label="Form Editor">
          <VscNote />
        </TabsTrigger>
        <TabsTrigger value="json" aria-label="JSON Editor">
          <VscJson />
        </TabsTrigger>
      </TabsList>
      <TabsContent value="form">
        {schema === '' ? (
          <>No Schema</>
        ) : (
          <JsonForm
            json={JSON.parse(schema)}
            setInput={setInput}
            onSubmit={handleOnPlay}
            disabled={disabled}
          />
        )}
      </TabsContent>
      <TabsContent value="json">
        <CodeEditor
          language="json"
          className="my-4"
          height="400px"
          code={JSON.stringify(JSON.parse(input), null, 2)}
          setCode={(code: string | undefined) => {
            if (!code) {
              return;
            }
            setInput(code);
          }}
        />
      </TabsContent>
    </Tabs>
  );
};
