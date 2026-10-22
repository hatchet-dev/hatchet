import { Skeleton } from '@/components/v1/ui/skeleton';
import { cn } from '@/lib/utils';
import { type ReactNode } from 'react';

export type StatCellProps = {
  label: string;
  hint?: ReactNode;
  loading: boolean;
  // A failed request must not render as an authoritative-looking zero.
  error?: boolean;
  className?: string;
  valueClassName?: string;
  children: ReactNode;
};

export function StatCell({
  label,
  hint,
  loading,
  error = false,
  className,
  valueClassName,
  children,
}: StatCellProps) {
  return (
    <div className={cn('flex min-w-0 flex-col gap-1 p-4', className)}>
      <span className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      {loading ? (
        <Skeleton className="h-8 w-16" />
      ) : (
        <span
          className={cn(
            'text-2xl font-semibold tabular-nums',
            !error && valueClassName,
          )}
        >
          {error ? 'n/a' : children}
        </span>
      )}
      {hint && <span className="text-xs text-muted-foreground/70">{hint}</span>}
    </div>
  );
}
