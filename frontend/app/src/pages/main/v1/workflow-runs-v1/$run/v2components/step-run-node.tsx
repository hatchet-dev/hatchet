import { V1RunIndicator } from '../../components/run-statuses';
import { TabOption } from './step-run-detail/step-run-detail';
import {
  PortalTooltip,
  PortalTooltipContent,
  PortalTooltipProvider,
  PortalTooltipTrigger,
} from '@/components/v1/ui/portal-tooltip';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { cn, formatDuration } from '@/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';

export type NodeData = {
  taskRun: V1TaskSummary | undefined;
  graphVariant: 'default' | 'input_only' | 'output_only' | 'none';
  onClick: (defaultOpenTab?: TabOption) => void;
  childWorkflowsCount: number;
  taskName: string;
  isSkipped?: boolean;
};

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: NodeData }) => {
  const variant = data.graphVariant;

  const startedAtEpoch = data.taskRun?.startedAt
    ? new Date(data.taskRun.startedAt).getTime()
    : 0;
  const finishedAtEpoch = data.taskRun?.finishedAt
    ? new Date(data.taskRun.finishedAt).getTime()
    : 0;

  return (
    <div className="flex min-w-fit grow flex-col justify-start">
      {(variant == 'default' || variant == 'input_only') && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
      <PortalTooltipProvider>
        <PortalTooltip>
          <PortalTooltipTrigger>
            <div
              className={cn(
                `step-run-card mb-1 w-full rounded-sm px-2 py-3 font-mono text-xs font-semibold text-[#050c1c] shadow-md dark:text-[#ffffff]`,
                `transition-all duration-300 ease-in-out`,
                `cursor-pointer`,
                `flex flex-row items-center justify-between gap-4 border-2 dark:border-[1px]`,
                `bg-[#ffffff] dark:bg-[#050c1c]`,
                'opacity-80 hover:opacity-100',
                'h-[30px]',
              )}
              onClick={() => data.onClick()}
            >
              {data.taskRun?.status == V1TaskStatus.RUNNING && (
                <span className="spark mask-gradient absolute inset-0 h-[100%] w-[100%] animate-flip overflow-hidden [mask:linear-gradient(#ccc,_transparent_50%)] before:absolute before:aspect-square before:w-[200%] before:rotate-[-90deg] before:animate-rotate before:bg-[conic-gradient(from_0deg,transparent_0_340deg,#ccc_360deg)] before:content-[''] before:[inset:0_auto_auto_50%] before:[translate:-50%_-15%]" />
              )}
              <span className="step-run-backdrop absolute inset-[1px] bg-background transition-colors duration-200" />
              <div className="z-10 flex w-full flex-row items-center justify-between gap-4">
                <div className="z-10 flex flex-row items-center justify-start gap-2">
                  <V1RunIndicator
                    status={data.taskRun?.status}
                    isSkipped={data.isSkipped}
                  />
                  <div className="max-w-[160px] flex-grow truncate">
                    {data.taskName}
                  </div>
                </div>
                {data.taskRun?.finishedAt && data.taskRun?.startedAt && (
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {formatDuration(finishedAtEpoch - startedAtEpoch)}
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
          </PortalTooltipTrigger>
          <PortalTooltipContent>{data.taskName}</PortalTooltipContent>
        </PortalTooltip>
      </PortalTooltipProvider>
    </div>
  );
});
