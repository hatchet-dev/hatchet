import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/next/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';

const badgeVariants = cva(
  'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default:
          'border-transparent bg-primary text-primary-foreground shadow hover:bg-primary/80',
        secondary:
          'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive:
          'border-transparent bg-destructive text-destructive-foreground shadow hover:bg-destructive/80',
        outline: 'text-foreground',
        xs: 'w-3 h-3 p-0 rounded-sm border-transparent',
        small: 'w-3 h-3 p-0 rounded-sm border-transparent',
        detail: 'text-foreground',
      },
      animated: {
        true: 'animate-pulse',
        false: '',
      },
    },
    defaultVariants: {
      variant: 'default',
      animated: false,
    },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  tooltipContent?: React.ReactNode;
}

function Badge({
  className,
  variant,
  animated,
  tooltipContent,
  ...props
}: BadgeProps) {
  // If it's the small variant and tooltip content is provided
  if (variant === 'xs' && tooltipContent) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              className={cn(badgeVariants({ variant, animated }), className)}
              {...props}
            />
          </TooltipTrigger>
          <TooltipContent>{tooltipContent}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <div
      className={cn(badgeVariants({ variant, animated }), className)}
      {...props}
    />
  );
}

export { Badge, badgeVariants };
