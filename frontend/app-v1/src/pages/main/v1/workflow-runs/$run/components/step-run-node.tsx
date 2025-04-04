import { Card, CardContent } from '@/components/v1/ui/card';
import { Label } from '@/components/v1/ui/label';
import { StepRun, StepRunStatus } from '@/lib/api';
import { cn, formatDuration } from '@/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';
import { RunIndicator, RunStatus } from '../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';

export interface StepRunNodeProps {
  stepRun: StepRun;
  variant: 'default' | 'input_only' | 'output_only';
  selected: 'none' | 'selected' | 'not_selected';
  onClick: () => void;
}

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: StepRunNodeProps }) => {
  const variant = data.variant;

  const selected = data.selected;

  return (
    <Card
      className={cn(
        data.stepRun.status == StepRunStatus.RUNNING ? 'active' : '',
        selected === 'none' || selected === 'selected'
          ? 'opacity-100'
          : 'opacity-20',

        selected === 'selected' ? 'border-primary' : '',
        'step-run-card p-3 cursor-pointer bg-[#020817] shadow-[0_1000px_0_0_hsl(0_0%_20%)_inset] transition-colors duration-200',
      )}
      onClick={data.onClick}
    >
      {data.stepRun.status == StepRunStatus.RUNNING && (
        <span className="spark mask-gradient animate-flip before:animate-rotate absolute inset-0 h-[100%] w-[100%] overflow-hidden rounded-full [mask:linear-gradient(#4EB4D7,_transparent_50%)] before:absolute before:aspect-square before:w-[200%] before:rotate-[-90deg] before:bg-[conic-gradient(from_0deg,transparent_0_340deg,#4EB4D7_360deg)] before:content-[''] before:[inset:0_auto_auto_50%] before:[translate:-50%_-15%]" />
      )}
      <span className="step-run-backdrop absolute inset-[1px] rounded-full bg-background transition-colors duration-200" />
      {(variant == 'default' || variant == 'input_only') && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
      <CardContent className="p-0 z-10 bg-background">
        <div className="flex flex-row justify-between gap-2 items-center">
          <RunIndicator status={data.stepRun.status} />
          <div className="font-bold text-sm">
            {data.stepRun.step?.readableId || data.stepRun.metadata.id}
          </div>
        </div>
        <div className="flex flex-col mt-1">
          {getTiming({ stepRun: data.stepRun })}
          {getChildren({ stepRun: data.stepRun })}
        </div>
      </CardContent>
      {(variant == 'default' || variant == 'output_only') && (
        <Handle
          type="source"
          position={Position.Right}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
    </Card>
  );
});

export function getTiming({ stepRun }: { stepRun: StepRun }) {
  const start = stepRun.startedAtEpoch;
  const end = stepRun.finishedAtEpoch;

  // otherwise just return started at or created at time
  return (
    <Label className="cursor-pointer">
      <span className="font-bold mr-2 text-xs">
        <RunStatus status={stepRun?.status || StepRunStatus.PENDING} />
      </span>
      <span className="text-gray-500 font-medium text-xs">
        {stepRun.startedAt && !end && <RelativeDate date={stepRun.startedAt} />}
        {start && end && formatDuration(end - start)}
      </span>
    </Label>
  );
}

function getChildren({ stepRun }: { stepRun: StepRun }) {
  if (stepRun.childWorkflowRuns?.length === 0) {
    return null;
  }

  return (
    <Label className="cursor-pointer">
      <span className="font-bold mr-2 text-xs">Children</span>
      <span className="text-gray-500 font-medium text-xs">
        {stepRun.childWorkflowRuns?.length}
      </span>
    </Label>
  );
}
