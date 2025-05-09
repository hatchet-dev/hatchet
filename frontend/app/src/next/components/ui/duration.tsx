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
import { formatDuration } from '@/next/lib/utils/formatDuration';
import { V1TaskStatus } from '@/lib/api';
import { isValidTimestamp } from './time';

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

  const startDate = new Date(start!);
  const endDate = isValidTimestamp(end) ? new Date(end!) : new Date();
  const duration = intervalToDuration({ start: startDate, end: endDate });
  const rawDuration = endDate.getTime() - startDate.getTime();
  const isRunning = status === 'RUNNING';

  const content = (
    <span className="flex items-center gap-1">
      {showIcon && <Clock className="h-3.5 w-3.5" />}
      <span className={isRunning ? 'animate-pulse' : ''}>
        {formatDuration(duration, rawDuration)}
        {isRunning && '...'}
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
