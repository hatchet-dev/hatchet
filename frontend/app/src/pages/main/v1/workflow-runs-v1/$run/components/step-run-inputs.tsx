import { CodeEditor } from '@/components/v1/ui/code-editor';
import { JSONType, JsonForm } from '@/components/v1/ui/json-form';
import { useEffect, useState } from 'react';

export interface StepRunOutputProps {
  input: string;
  schema: object;
  setInput: React.Dispatch<React.SetStateAction<string>>;
  disabled: boolean;
  handleOnPlay: () => void;
  mode: 'json' | 'form';
}

const tryFormat = (input: string) => {
  try {
    return JSON.stringify(JSON.parse(input), null, 2);
  } catch (e) {
    return input;
  }
};

export const StepRunInputs: React.FC<StepRunOutputProps> = ({
  input,
  schema,
  disabled,
  handleOnPlay,
  setInput,
  mode,
}) => {
  const [currentInput, setCurrentInput] = useState(tryFormat(input));

  useEffect(() => {
    setCurrentInput(input);
  }, [input]);

  const handleCodeChange = (code: string | undefined) => {
    if (!code) {
      return;
    }
    setCurrentInput(code);
    setInput(code);
  };

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
            code={currentInput}
            setCode={handleCodeChange}
          />
        </div>
      )}
    </>
  );
};
