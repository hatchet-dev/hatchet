import { Button } from '@/components/ui/button';
import { CodeEditor } from '@/components/ui/code-editor';
import { JsonForm } from '@/components/ui/json-form';
import { Loading } from '@/components/ui/loading';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { PlayIcon } from '@radix-ui/react-icons';
import { useEffect, useState } from 'react';
import { VscNote, VscJson } from 'react-icons/vsc';

export interface StepRunOutputProps {
  input: string;
  schema: string;
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
          {schema === '' ? (
            <>No Schema</>
          ) : (
            <JsonForm
              inputSchema={JSON.parse(schema)}
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
