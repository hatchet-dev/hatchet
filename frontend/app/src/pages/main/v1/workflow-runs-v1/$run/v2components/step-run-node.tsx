import { Label } from '@/components/v1/ui/label';
import { StepRun, StepRunStatus, V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { cn, formatDuration } from '@/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';
import { RunStatus, V1RunIndicator } from '../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { TabOption } from './step-run-detail/step-run-detail';
import { Link } from 'react-router-dom';
import { useColumnFilters } from '../../hooks/column-filters';

export type NodeData = {
  task: V1TaskSummary;
  graphVariant: 'default' | 'input_only' | 'output_only' | 'none';
  onClick: (defaultOpenTab?: TabOption) => void;
  childWorkflowsCount: number;
  taskName: string;
};

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: NodeData }) => {
  const variant = data.graphVariant;

  const startedAtEpoch = data.task.startedAt
    ? new Date(data.task.startedAt).getTime()
    : 0;
  const finishedAtEpoch = data.task.finishedAt
    ? new Date(data.task.finishedAt).getTime()
    : 0;

  const { queryParamNames } = useColumnFilters();

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
        key={data.task.metadata.id}
        data-step-id={data.task.metadata.id}
        className={cn(
          `step-run-card shadow-md rounded-sm py-3 px-2 mb-1 w-full text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
          `transition-all duration-300 ease-in-out`,
          `cursor-pointer`,
          `flex flex-row items-center justify-between gap-4 border-2 dark:border-[1px]`,
          `bg-[#ffffff] dark:bg-[#050c1c]`,
          'hover:opacity-100 opacity-80',
          'h-[30px]',
        )}
        onClick={() => data.onClick()}
      >
        {data.task.status == V1TaskStatus.RUNNING && (
          <span className="spark mask-gradient animate-flip before:animate-rotate absolute inset-0 h-[100%] w-[100%] overflow-hidden [mask:linear-gradient(#ccc,_transparent_50%)] before:absolute before:aspect-square before:w-[200%] before:rotate-[-90deg] before:bg-[conic-gradient(from_0deg,transparent_0_340deg,#ccc_360deg)] before:content-[''] before:[inset:0_auto_auto_50%] before:[translate:-50%_-15%]" />
        )}
        <span className="step-run-backdrop absolute inset-[1px] bg-background transition-colors duration-200" />
        <div className="z-10 flex flex-row items-center justify-between gap-4 w-full">
          <div className="flex flex-row items-center justify-start gap-2 z-10">
            <V1RunIndicator status={data.task.status} />
            <div className="truncate flex-grow">{data.taskName}</div>
          </div>
          {data.task.finishedAt && data.task.startedAt && (
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
      {data.childWorkflowsCount ? (
        <Link
          to={{
            pathname: '/v1/workflow-runs',
            search: new URLSearchParams({
              ...Object.fromEntries(new URLSearchParams(location.search)),
              [queryParamNames.parentTaskExternalId]: data.task.metadata.id,
            }).toString(),
          }}
        >
          <div
            key={`${data.task.metadata.id}-child-workflows`}
            className={cn(
              `w-[calc(100%-1rem)] box-border shadow-md ml-4 rounded-sm py-3 px-2 mb-1 text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
              `transition-all duration-300 ease-in-out`,
              `cursor-pointer`,
              `flex flex-row items-center justify-start border-2 dark:border-[1px]`,
              `bg-[#ffffff] dark:bg-[#050c1c]`,
              'h-[30px]',
            )}
          >
            <div className="truncate flex-grow">
              {data.taskName}: {data.childWorkflowsCount} children
            </div>
          </div>{' '}
        </Link>
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
