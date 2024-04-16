import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground shadow',
        secondary:
          'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive:
          'border-transparent bg-destructive text-destructive-foreground shadow hover:bg-destructive/80',
        outline: 'text-foreground',
        successful:
          'border-transparent rounded-sm px-1 font-normal text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
        failed:
          'border-transparent rounded-sm px-1 font-normal text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500/30',
        inProgress:
          'border-transparent rounded-sm px-1 font-normal text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
);
export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
