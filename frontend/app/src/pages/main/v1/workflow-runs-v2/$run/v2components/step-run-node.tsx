import { Label } from '@/components/v1/ui/label';
import { Step, StepRun, StepRunStatus } from '@/lib/api';
import { cn, formatDuration } from '@/lib/utils';
import { memo, useState } from 'react';
import { Handle, Position } from 'reactflow';
import { RunIndicator, RunStatus } from '../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { TabOption } from './step-run-detail/step-run-detail';

export interface StepRunNodeProps {
  stepRun: StepRun;
  step: Step;
  graphVariant: 'default' | 'input_only' | 'output_only' | 'none';
  //   selected: 'none' | 'selected' | 'not_selected';
  onClick: (defaultOpenTab?: TabOption) => void;
}

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: StepRunNodeProps }) => {
  const variant = data.graphVariant;
  const [isHovering, setIsHovering] = useState(false);

  //   const selected = data.selected;
  const step = data.step;
  const stepRun = data.stepRun;

  return (
    <div className="flex flex-col justify-start min-w-fit grow">
      {(variant == 'default' || variant == 'input_only') && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
      <div
        key={step.metadata.id}
        data-step-id={step.metadata.id}
        className={cn(
          `step-run-card shadow-md rounded-sm py-3 px-2 mb-1 w-full text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
          `transition-all duration-300 ease-in-out`,
          `cursor-pointer`,
          `flex flex-row items-center justify-between gap-4 border-2 dark:border-[1px]`,
          `bg-[#ffffff] dark:bg-[#050c1c]`,
        )}
        style={{
          height: '30px',
          opacity: isHovering ? 1 : 0.8,
        }}
        onClick={() => data.onClick()}
        onMouseEnter={() => setIsHovering(true)}
        onMouseLeave={() => setIsHovering(false)}
      >
        {data.stepRun.status == StepRunStatus.RUNNING && (
          <span className="spark mask-gradient animate-flip before:animate-rotate absolute inset-0 h-[100%] w-[100%] overflow-hidden [mask:linear-gradient(#ccc,_transparent_50%)] before:absolute before:aspect-square before:w-[200%] before:rotate-[-90deg] before:bg-[conic-gradient(from_0deg,transparent_0_340deg,#ccc_360deg)] before:content-[''] before:[inset:0_auto_auto_50%] before:[translate:-50%_-15%]" />
        )}
        <span className="step-run-backdrop absolute inset-[1px] bg-background transition-colors duration-200" />
        <div className="z-10 flex flex-row items-center justify-between gap-4 w-full">
          <div className="flex flex-row items-center justify-start gap-2 z-10">
            <RunIndicator status={data.stepRun.status} />
            <div className="truncate flex-grow">{step.readableId}</div>
          </div>
          {stepRun.finishedAtEpoch && stepRun.startedAtEpoch && (
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {formatDuration(stepRun.finishedAtEpoch - stepRun.startedAtEpoch)}
            </div>
          )}
        </div>

        {(variant == 'default' || variant == 'output_only') && (
          <Handle
            type="source"
            position={Position.Right}
            style={{ visibility: 'hidden' }}
            isConnectable={false}
          />
        )}
      </div>
      {stepRun?.childWorkflowsCount ? (
        <div
          key={`${step.metadata.id}-child-workflows`}
          className={cn(
            `w-[calc(100%-1rem)] box-border shadow-md ml-4 rounded-sm py-3 px-2 mb-1 text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
            `transition-all duration-300 ease-in-out`,
            `cursor-pointer`,
            `flex flex-row items-center justify-start border-2 dark:border-[1px]`,
            `bg-[#ffffff] dark:bg-[#050c1c]`,
          )}
          style={{
            height: '30px',
          }}
          onClick={() => data.onClick(TabOption.ChildWorkflowRuns)}
        >
          <div className="truncate flex-grow">
            {step.readableId}: {stepRun.childWorkflowsCount} children
          </div>
        </div>
      ) : null}
    </div>
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
