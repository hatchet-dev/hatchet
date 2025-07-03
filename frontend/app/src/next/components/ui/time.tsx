import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/next/lib/utils';
import { format } from 'date-fns';
import TimeAgo from 'timeago-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';

// Helper to check if a timestamp is valid (not empty or the special "0001-01-01" value)
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

const timeVariants = cva('text-sm', {
  variants: {
    variant: {
      // Live count of time since the event
      timeSince: 'text-muted-foreground',
      // Formatted ISO-like date in monospace font
      timestamp: 'font-mono text-xs bg-muted px-1.5 py-0.5 rounded',
      // Short form of the date-time
      short: 'text-xs',
      // Compact timestamp for table cells
      compact: 'font-mono text-xs',
    },
  },
  defaultVariants: {
    variant: 'timeSince',
  },
});

export interface TimeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof timeVariants> {
  date?: string | Date | null;
  /**
   * Update interval in milliseconds for timeSince variant
   * This is converted to seconds for the timeago-react component
   * Default: 0 (uses timeago.js default intervals)
   */
  updateInterval?: number;
  asChild?: boolean;
  /**
   * Optional tooltip variant to show when hovering over the time
   * If specified, the time will be wrapped in a tooltip showing the specified variant
   */
  tooltipVariant?: 'timestamp' | 'short' | 'compact';
}

export function Time({
  className,
  variant,
  date,
  updateInterval = 0,
  asChild,
  tooltipVariant,
  ...props
}: TimeProps) {
  const renderTime = (variant: TimeProps['variant']) => {
    // For timestamp and short variants, we'll format the date directly
    if (variant === 'timestamp' && isValidTimestamp(date)) {
      const dateObj = typeof date === 'string' ? new Date(date) : date;
      const formattedTime = format(dateObj, 'yyyy-MM-dd HH:mm:ss.SSS');

      return (
        <span
          className={cn(!asChild && timeVariants({ variant }), className)}
          {...props}
        >
          {formattedTime}
        </span>
      );
    }

    if (variant === 'short' && isValidTimestamp(date)) {
      const dateObj = typeof date === 'string' ? new Date(date) : date;
      const formattedTime = format(dateObj, 'MMM d, HH:mm');

      return (
        <span
          className={cn(!asChild && timeVariants({ variant }), className)}
          {...props}
        >
          {formattedTime}
        </span>
      );
    }

    // Add compact variant
    if (variant === 'compact' && isValidTimestamp(date)) {
      const dateObj = typeof date === 'string' ? new Date(date) : date;
      const formattedTime = format(dateObj, 'MM-dd HH:mm:ss.SSS');

      return (
        <span
          className={cn(!asChild && timeVariants({ variant }), className)}
          {...props}
        >
          {formattedTime}
        </span>
      );
    }

    // For timeSince variant or if no date is provided
    if (!isValidTimestamp(date)) {
      return (
        <span
          className={cn(!asChild && timeVariants({ variant }), className)}
          {...props}
        >
          -
        </span>
      );
    }

    // For timeSince variant, use TimeAgo component
    // Convert milliseconds to seconds for minInterval if specified
    const opts =
      updateInterval > 0
        ? { minInterval: Math.floor(updateInterval / 1000) }
        : undefined;

    return (
      <span
        className={cn(!asChild && timeVariants({ variant }), className)}
        {...props}
      >
        <TimeAgo datetime={date} live={true} opts={opts} />
      </span>
    );
  };

  if (tooltipVariant && isValidTimestamp(date)) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>{renderTime(variant)}</TooltipTrigger>
          <TooltipContent className="bg-muted">
            <span className="text-foreground">
              {renderTime(tooltipVariant)}
            </span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return renderTime(variant);
}
