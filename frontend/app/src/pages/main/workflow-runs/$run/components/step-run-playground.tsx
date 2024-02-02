import { Code } from '@/components/ui/code';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { StepRun, StepRunStatus } from '@/lib/api';
import { useEffect, useState } from 'react';
import { RunStatus } from '../../components/run-statuses';
import { getTiming } from './step-run-node';

export function StepRunPlayground({
  stepRun,
  setStepRun,
}: {
  stepRun: StepRun | null;
  setStepRun: (stepRun: StepRun | null) => void;
}) {
  const originalInput = JSON.stringify(
    JSON.parse(stepRun?.input || '{}'),
    null,
    2,
  );

  const output = JSON.stringify(JSON.parse(stepRun?.output || '{}'), null, 2);

  const [stepInput, setStepInput] = useState(originalInput);

  useEffect(() => {
    setStepInput(JSON.stringify(JSON.parse(stepRun?.input || '{}'), null, 2));
  }, [stepRun]);

  return (
    <Dialog
      open={!!stepRun}
      onOpenChange={(open) => {
        if (!open) {
          setStepRun(null);
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12">
        <DialogHeader>
          <div className="flex flex-row justify-between items-center">
            <DialogTitle>
              {'Coding Assistant' ||
                stepRun?.step?.readableId ||
                stepRun?.metadata.id}
            </DialogTitle>
            <RunStatus status={stepRun?.status || StepRunStatus.PENDING} />
          </div>
          {stepRun && getTiming({ stepRun })}
          <DialogDescription>
            You can change the input to your step and see the output here. By
            default, this will trigger all child steps.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-row justify-between items-center">
          <div className="font-bold">Input</div>
          {/* <Button
            className="w-fit"
            disabled={stepInput === originalInput || isLoading}
            onClick={() => {
              setIsLoading(true);
              setStatus(StepRunStatus.RUNNING);
            }}
          >
            <ArrowPathIcon
              className={cn(isLoading ? 'rotate-180' : '', 'h-4 w-4 mr-2')}
            />
            Rerun Step
          </Button> */}
        </div>
        {stepInput && (
          <Code
            language="json"
            copy={true}
            code={stepInput}
            setCode={setStepInput}
            wrapLines={false}
            className="max-w-full"
          />
        )}
        <div className="font-bold">Output</div>
        <Code
          language="json"
          code={output}
          copy={true}
          maxHeight="300px"
          wrapLines={true}
          className="max-w-full"
        />
      </DialogContent>
    </Dialog>
  );
}
