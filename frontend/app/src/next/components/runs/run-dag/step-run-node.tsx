import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { cn } from '@/next/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';
import { Link } from 'react-router-dom';
import { Duration } from '@/next/components/ui/duration';
import { RunsBadge } from '../runs-badge';
import { ROUTES } from '@/next/lib/routes';

enum TabOption {
  Output = 'output',
  ChildWorkflowRuns = 'child-workflow-runs',
  Input = 'input',
  Logs = 'logs',
}

export type NodeData = {
  taskRun: V1TaskSummary | undefined;
  graphVariant: 'default' | 'input_only' | 'output_only' | 'none';
  onClick: (defaultOpenTab?: TabOption) => void;
  childWorkflowsCount: number;
  taskName: string;
};

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: NodeData }) => {
  const variant = data.graphVariant;

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
        {data.taskRun?.status == V1TaskStatus.RUNNING && (
          <span className="spark mask-gradient animate-flip before:animate-rotate absolute inset-0 h-[100%] w-[100%] overflow-hidden [mask:linear-gradient(#ccc,_transparent_50%)] before:absolute before:aspect-square before:w-[200%] before:rotate-[-90deg] before:bg-[conic-gradient(from_0deg,transparent_0_340deg,#ccc_360deg)] before:content-[''] before:[inset:0_auto_auto_50%] before:[translate:-50%_-15%]" />
        )}
        <span className="step-run-backdrop absolute inset-[1px] bg-background transition-colors duration-200" />
        <div className="z-10 flex flex-row items-center justify-between gap-4 w-full">
          <div className="flex flex-row items-center justify-start gap-2 z-10">
            <RunsBadge status={data.taskRun?.status} variant="xs" />
            <div className="truncate flex-grow max-w-[160px]">
              {data.taskName}
            </div>
          </div>
          {data.taskRun?.finishedAt && data.taskRun?.startedAt && (
            <div className="text-xs text-gray-500 dark:text-gray-400">
              <Duration
                className="text-xs"
                start={data.taskRun?.startedAt}
                end={data.taskRun?.finishedAt}
                status={data.taskRun?.status}
              />
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
      {data.childWorkflowsCount && data.taskRun ? (
        <Link
          to={{
            pathname: ROUTES.runs.list,
            search: new URLSearchParams({
              ...Object.fromEntries(new URLSearchParams(location.search)),
              // [queryParamNames.parentTaskExternalId]: data.taskRun.metadata.id,
            }).toString(),
          }}
        >
          <div
            key={`${data.taskRun.metadata.id}-child-workflows`}
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
