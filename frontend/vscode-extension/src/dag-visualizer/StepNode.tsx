import { clsx } from 'clsx';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';
import { twMerge } from 'tailwind-merge';
import { NodeDisplayData } from './types';

function cn(...inputs: Parameters<typeof clsx>) {
  return twMerge(clsx(inputs));
}

function formatDuration(ms: number): string {
  if (ms < 0) return '0s';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000)
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  const h = Math.floor(ms / 3600000);
  const m = Math.floor((ms % 3600000) / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return `${h}h ${m}m ${s}s`;
}

function StatusIndicator({
  status,
  isSkipped,
}: {
  status?: NodeDisplayData['status'];
  isSkipped?: boolean;
}) {
  if (isSkipped) {
    return (
      <svg
        className="h-3 w-3 text-gray-400"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        aria-label="Skipped"
      >
        <circle cx="12" cy="12" r="10" />
        <line x1="4" y1="4" x2="20" y2="20" />
      </svg>
    );
  }

  const colorMap: Record<string, string> = {
    running: 'bg-yellow-500',
    completed: 'bg-green-500',
    failed: 'bg-red-500',
    cancelled: 'bg-orange-500',
    pending: 'bg-slate-500',
  };

  const color = status ? (colorMap[status] ?? 'bg-gray-400') : 'bg-gray-400';
  return <div className={`h-[6px] w-[6px] rounded-full ${color}`} />;
}

const StepNode = memo(({ data }: { data: NodeDisplayData }) => {
  const variant = data.graphVariant;

  return (
    <div className="flex min-w-fit grow flex-col justify-start">
      {(variant === 'default' || variant === 'input_only') && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
      <div
        title={data.taskName}
        className={cn(
          'step-run-card mb-1 w-full rounded-sm px-2 py-3 font-mono text-xs font-semibold text-[#050c1c] shadow-md dark:text-[#ffffff]',
          'transition-all duration-300 ease-in-out',
          'cursor-pointer',
          'flex flex-row items-center justify-between gap-4 border-2 dark:border-[1px]',
          'bg-[#ffffff] dark:bg-[#050c1c]',
          'opacity-80 hover:opacity-100',
          'h-[30px]',
        )}
        onClick={() => data.onClick?.()}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && data.onClick?.()}
      >
        <div className="z-10 flex w-full flex-row items-center justify-between gap-4">
          <div className="z-10 flex flex-row items-center justify-start gap-2">
            <StatusIndicator status={data.status} isSkipped={data.isSkipped} />
            <div className="max-w-[160px] flex-grow truncate">
              {data.taskName}
            </div>
          </div>
          {data.durationMs !== undefined && (
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {formatDuration(data.durationMs)}
            </div>
          )}
        </div>
      </div>
      {(variant === 'default' || variant === 'output_only') && (
        <Handle
          type="source"
          position={Position.Right}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
    </div>
  );
});

StepNode.displayName = 'StepNode';

export default StepNode;
