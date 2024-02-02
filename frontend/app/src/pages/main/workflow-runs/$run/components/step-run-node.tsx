import { Card, CardContent } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { StepRun, StepRunStatus } from '@/lib/api';
import { cn, relativeDate } from '@/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';
import { RunIndicator } from '../../components/run-statuses';

// eslint-disable-next-line react/display-name
export default memo(
  ({
    data,
  }: {
    data: {
      stepRun: StepRun;
      variant: 'default' | 'input_only' | 'output_only';
      onClick: () => void;
    };
  }) => {
    const variant = data.variant;

    return (
      <Card
        className={cn(
          data.stepRun.status == StepRunStatus.RUNNING ? 'active' : '',
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
          {getTiming({ stepRun: data.stepRun })}
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
  },
);

export function getTiming({ stepRun }: { stepRun: StepRun }) {
  const start = stepRun.startedAtEpoch;
  const end = stepRun.finishedAtEpoch;

  if (start && end) {
    return (
      <Label className="cursor-pointer">
        <span className="font-bold mr-2 text-xs">Duration</span>
        <span className="text-gray-500 font-medium text-xs">
          {formatDuration(end - start)}
        </span>
      </Label>
    );
  }

  if (!stepRun.startedAt) {
    return (
      <Label className="cursor-pointer">
        <span className="font-bold mr-2 text-xs">Pending</span>
      </Label>
    );
  }

  // otherwise just return started at or created at time
  return (
    <Label className="cursor-pointer">
      <span className="font-bold mr-2 text-xs">Started</span>
      <span className="text-gray-500 font-medium text-xs">
        {relativeDate(stepRun.startedAt)}
      </span>
    </Label>
  );
}

export function formatDuration(duration: number) {
  const milliseconds = duration % 1000;
  const seconds = Math.round(duration / 1000);
  const minutes = Math.round(seconds / 60);
  const hours = Math.round(minutes / 60);

  if (seconds == 0 && hours == 0 && minutes == 0) {
    return `${milliseconds}ms`;
  }

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }

  return `${seconds}s`;
}
