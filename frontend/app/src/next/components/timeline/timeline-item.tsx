import { intervalToDuration, Duration } from 'date-fns';
import { TimelineItemProps } from './types';
import { V1TaskStatus } from '@/next/lib/api';
import useTimeline from '@/next/hooks/use-timeline-context';
import { cn } from '@/next/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { formatDuration } from '../runs/run-id';
import { RunStatusConfigs } from '../runs/runs-badge';

// Helper to check if a timestamp is valid (not empty or the special "0001-01-01" value)
const isValidTimestamp = (timestamp?: string): boolean => {
  if (!timestamp) {
    return false;
  }

  // Check for the special "0001-01-01" timestamp that represents a null value
  if (timestamp.startsWith('0001-01-01')) {
    return false;
  }

  const date = new Date(timestamp);
  // Check if the date is valid and not too far in the past
  return !isNaN(date.getTime()) && date.getFullYear() > 1970;
};

function TimelineTooltip({
  queueDuration,
  rawQueueTimeMs,
  itemStartTime,
  itemEndTime,
}: {
  queueDuration: Duration;
  rawQueueTimeMs: number;
  itemStartTime: number;
  itemEndTime: number;
}) {
  return (
    <div className="flex flex-col gap-1">
      <span>
        Time in queue: {formatDuration(queueDuration, rawQueueTimeMs)}
      </span>
      <span>
        Execution time:{' '}
        {formatDuration(
          intervalToDuration({
            start: itemStartTime,
            end: itemEndTime,
          }),
          itemEndTime - itemStartTime,
        )}
      </span>
    </div>
  );
}

export function TimelineItem({ item, onClick }: TimelineItemProps) {
  const { earliest, timeRange } = useTimeline();

  const handleClick = () => {
    if (onClick) {
      onClick();
    }
  };

  // Check if item has a valid createdAt timestamp
  if (!isValidTimestamp(item.metadata.createdAt)) {
    return undefined;
  }

  // At this point we know createdAt is valid and not undefined
  const itemCreatedAt = new Date(item.metadata.createdAt!).getTime();
  const createdAtDate = new Date(item.metadata.createdAt!);

  // Handle items with only createdAt (pending items)
  if (!isValidTimestamp(item.startedAt)) {
    // For pending items, show a dot at the creation time
    const createdOffset = Math.round(
      ((itemCreatedAt - earliest) / timeRange) * 100,
    );

    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              className="absolute h-full rounded cursor-pointer hover:brightness-110 z-10 flex items-center"
              style={{
                left: `${createdOffset}%`,
              }}
              onClick={handleClick}
            >
              <div className="flex flex-row items-center h-full">
                <div className="w-[10px] h-[10px] rounded-full bg-white/50"></div>
              </div>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <span>Created: {createdAtDate.toLocaleString()}</span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  // At this point we know startedAt is valid and not undefined
  const itemStartTime = new Date(item.startedAt!).getTime();
  const itemEndTime = isValidTimestamp(item.finishedAt)
    ? new Date(item.finishedAt!).getTime()
    : Date.now();

  // Calculate the total time range (ensure it's at least 1ms to prevent division by zero)
  const duration = Math.max(itemEndTime - itemStartTime, 1);

  const widthPercent = (duration / timeRange) * 100;

  const createdOffset = Math.round(
    ((itemCreatedAt - earliest) / timeRange) * 100,
  );

  const startedOffset = Math.round(
    ((itemStartTime - earliest) / timeRange) * 100,
  );

  const timeInQueue = Math.max(itemStartTime - itemCreatedAt, 1);

  const timeToStartWidth = Math.round((timeInQueue / timeRange) * 100);

  // Format the time in queue
  const queueDuration = intervalToDuration({
    start: itemCreatedAt,
    end: itemStartTime,
  });

  const rawQueueTimeMs = itemStartTime - itemCreatedAt;

  return (
    <>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              className="absolute h-full rounded cursor-pointer hover:brightness-110 z-10 flex items-center"
              style={{
                width: `${timeToStartWidth}%`,
                left: `${createdOffset}%`,
              }}
            >
              <EventDot />
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <TimelineTooltip
              queueDuration={queueDuration}
              rawQueueTimeMs={rawQueueTimeMs}
              itemStartTime={itemStartTime}
              itemEndTime={itemEndTime}
            />
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              className="absolute h-full rounded cursor-pointer hover:brightness-110 min-w-[5px]"
              style={{
                left: `${startedOffset}%`,
                width: `${widthPercent}%`,
              }}
              onClick={handleClick}
            >
              <RunBar status={item.status} />
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <TimelineTooltip
              queueDuration={queueDuration}
              rawQueueTimeMs={rawQueueTimeMs}
              itemStartTime={itemStartTime}
              itemEndTime={itemEndTime}
            />
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </>
  );
}

function RunBar({ status }: { status: V1TaskStatus }) {
  const statusColorClass = RunStatusConfigs[status].colors;

  return (
    <div className="flex flex-row items-center h-full rounded-md overflow-hidden">
      {status === 'RUNNING' ? (
        <div
          className={cn(
            'z-10 px-1 text-[10px] font-mono font-light text-muted-foreground/60 whitespace-nowrap w-full flex justify-between h-full items-center relative',
            statusColorClass,
          )}
        >
          {/* Inner glow pulsing effect */}
          <div className="absolute inset-0 bg-blue-500/30 animate-pulse rounded-md"></div>
          {/* Subtle moving dots effect */}
          <div className="absolute inset-y-0 left-0 w-full h-full overflow-hidden">
            <div className="absolute h-full w-1/5 bg-gradient-to-r from-transparent via-white/30 to-transparent animate-move"></div>
          </div>
        </div>
      ) : (
        <div
          className={cn(
            'z-10 px-1 text-[10px] font-mono font-light text-muted-foreground/60 whitespace-nowrap w-full flex justify-between h-full items-center',
            statusColorClass,
          )}
        ></div>
      )}
    </div>
  );
}

function EventDot() {
  return (
    <>
      <div className="flex flex-row items-center h-full ml-[-5px]">
        <div className="w-[10px] h-[10px] rounded-full bg-white/50"></div>
      </div>
      <div className="border-t border-white/50 w-full"></div>
    </>
  );
}
