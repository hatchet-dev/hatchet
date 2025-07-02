import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/next/lib/utils';
import { intervalToDuration } from 'date-fns';
import { Clock } from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { V1TaskStatus } from '@/lib/api';
import { Duration as DateFnsDuration } from 'date-fns';

export function formatDuration(
  duration: DateFnsDuration,
  rawTimeMs: number,
): string {
  const parts = [];

  if (duration.days) {
    parts.push(`${duration.days}d`);
  }

  if (duration.hours) {
    parts.push(`${duration.hours}h`);
  }

  if (duration.minutes) {
    parts.push(`${duration.minutes}m`);
  }

  if (rawTimeMs < 10000 && duration.seconds) {
    const ms = Math.floor((rawTimeMs % 1000) / 10);
    parts.push(`${duration.seconds}.${ms.toString().padStart(2, '0')}s`);
    return parts.join(' ');
  }

  if (duration.seconds) {
    parts.push(`${duration.seconds}s`);
  }

  if (rawTimeMs < 1000) {
    const ms = rawTimeMs % 1000;
    parts.push(`${ms}ms`);
  }

  return parts.join(' ');
}

export const isValidTimestamp = (
  timestamp?: string | Date | null,
): timestamp is string | Date => {
  if (!timestamp) {
    return false;
  }

  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;

  // Check for the special "0001-01-01" timestamp that represents a null value
  if (date.getFullYear() <= 1) {
    return false;
  }

  // Check if the date is valid and not too far in the past
  return !isNaN(date.getTime()) && date.getFullYear() > 1970;
};

const durationVariants = cva('text-sm', {
  variants: {
    variant: {
      default: 'text-muted-foreground',
      compact: 'font-mono text-xs',
    },
  },
  defaultVariants: {
    variant: 'default',
  },
});

export interface DurationProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof durationVariants> {
  start?: string | Date | null;
  end?: string | Date | null;
  status?: V1TaskStatus;
  showIcon?: boolean;
  asChild?: boolean;
}

export function Duration({
  className,
  variant,
  start,
  end,
  status,
  showIcon = true,
  asChild,
  ...props
}: DurationProps) {
  if (!isValidTimestamp(start)) {
    return (
      <span
        className={cn(!asChild && durationVariants({ variant }), className)}
        {...props}
      >
        -
      </span>
    );
  }

  const startDate = typeof start === 'string' ? new Date(start) : start;
  const endDate = isValidTimestamp(end) ? new Date(end) : new Date();
  const duration = intervalToDuration({ start: startDate, end: endDate });
  const rawDuration = endDate.getTime() - startDate.getTime();
  const isRunning = status === 'RUNNING';

  const content = (
    <span className="flex items-center gap-1">
      {showIcon ? <Clock className="h-3.5 w-3.5" /> : null}
      <span className={isRunning ? 'animate-pulse' : ''}>
        {formatDuration(duration, rawDuration)}
        {isRunning ? '...' : null}
      </span>
    </span>
  );

  if (variant === 'compact') {
    return (
      <span
        className={cn(!asChild && durationVariants({ variant }), className)}
        {...props}
      >
        {content}
      </span>
    );
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            className={cn(!asChild && durationVariants({ variant }), className)}
            {...props}
          >
            {content}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          {isRunning ? 'Running for ' : ''}
          {formatDuration(duration, rawDuration)}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
