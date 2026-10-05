import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { cn } from '@/lib/utils';
import { type ReactNode } from 'react';

// Shared chrome for every overview dashboard panel: a light card with an
// icon + uppercase mono title, an optional subtitle, an optional header
// action (expand / manage link), a subtle border, and a body. Matches the
// existing SupportSection card treatment so the dashboard reads as one system.
export function PanelCard({
  icon,
  title,
  subtitle,
  action,
  className,
  bodyClassName,
  children,
}: {
  icon?: ReactNode;
  title: string;
  subtitle?: string;
  action?: ReactNode;
  className?: string;
  bodyClassName?: string;
  children: ReactNode;
}) {
  return (
    <Card
      variant="light"
      className={cn(
        'flex flex-col bg-transparent ring-1 ring-border/50 border-none',
        className,
      )}
    >
      <CardHeader className="flex flex-row items-start justify-between gap-2 border-b border-border/50 p-4">
        <div className="min-w-0 space-y-0.5">
          <CardTitle className="flex items-center gap-2 whitespace-nowrap font-mono text-xs font-normal uppercase tracking-wider text-muted-foreground">
            {icon}
            {title}
          </CardTitle>
          {subtitle && (
            <p className="text-xs text-muted-foreground/70">{subtitle}</p>
          )}
        </div>
        {action}
      </CardHeader>
      <CardContent className={cn('flex-1 p-4', bodyClassName)}>
        {children}
      </CardContent>
    </Card>
  );
}

// Renders the loading / error / empty branches shared by the live panels, or
// the panel body once data is present. Keeps every panel's state handling
// identical: a skeleton while loading, a retry affordance on error, and a
// per-panel empty message.
export function PanelState({
  loading,
  error,
  onRetry,
  isEmpty,
  emptyText,
  children,
}: {
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  isEmpty: boolean;
  emptyText: string;
  children: ReactNode;
}) {
  if (loading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-2/3" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-start gap-2 text-sm text-muted-foreground">
        <span>Could not load this panel.</span>
        {onRetry && (
          <Button variant="outline" size="sm" onClick={onRetry}>
            Retry
          </Button>
        )}
      </div>
    );
  }

  if (isEmpty) {
    return <p className="text-sm text-muted-foreground">{emptyText}</p>;
  }

  return <>{children}</>;
}
