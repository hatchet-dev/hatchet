import { CodeEditor } from '@/components/ui/code-editor';
import { JSONType, JsonForm } from '@/components/ui/json-form';

export interface StepRunOutputProps {
  input: string;
  schema: object;
  setInput: React.Dispatch<React.SetStateAction<string>>;
  disabled: boolean;
  handleOnPlay: () => void;
  mode: 'json' | 'form';
}

export const StepRunInputs: React.FC<StepRunOutputProps> = ({
  input,
  schema,
  disabled,
  handleOnPlay,
  setInput,
  mode,
}) => {
  return (
    <>
      {mode === 'form' && (
        <div>
          {!schema ? (
            <>No Schema</>
          ) : (
            <JsonForm
              inputSchema={schema as JSONType}
              setInput={setInput}
              inputData={JSON.parse(input)}
              onSubmit={handleOnPlay}
              disabled={disabled}
            />
          )}
        </div>
      )}

      {mode === 'json' && (
        <div>
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
        </div>
      )}
    </>
  );
};
