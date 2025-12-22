import { cn } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';
import * as React from 'react';

const badgeVariants = cva(
  'inline-flex items-center rounded-md border px-3 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
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
          'border-transparent rounded-sm font-normal text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
        failed:
          'border-transparent rounded-sm font-normal text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500/30',
        inProgress:
          'border-transparent rounded-sm font-normal text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
        outlineDestructive:
          'border border-destructive rounded-sm font-normal text-red-800 dark:text-red-300 bg-transparent',
        queued:
          'border-transparent rounded-sm font-normal text-slate-800 dark:text-slate-300 bg-slate-500/20 ring-slate-500/30',
        cancelled:
          'border-transparent rounded-sm font-normal text-orange-800 dark:text-orange-300 bg-orange-500/20 ring-orange-500/30',
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

export { Badge };
