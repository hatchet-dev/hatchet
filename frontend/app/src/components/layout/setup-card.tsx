import { cn } from '@/lib/utils';
import { type ReactNode } from 'react';

// A centered, compact card: a small header (title + optional description) over
// a body, with a subtle border on a slightly-off background. Defaults to a
// narrow width that suits the short org/tenant creation forms; the onboarding
// flow, which holds a wider stepper, passes a larger max-width via className.
export function SetupCard({
  title,
  description,
  children,
  footer,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('w-full max-w-xl', className)}>
      <div className="rounded-xl border border-border bg-muted/20 shadow-sm">
        <div className="border-b border-border px-6 py-5">
          <h2 className="text-base font-medium tracking-tight">{title}</h2>
          {description ? (
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          ) : null}
        </div>
        <div className="px-6 py-5">{children}</div>
        {footer ? (
          <div className="flex items-center justify-end gap-2 border-t border-border px-6 py-4">
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}

// A full-screen centered surface for the creation flows: a scrollable
// background that centers a SetupCard, with an optional top-right slot for
// controls (e.g. sign out).
export function SetupScreen({
  topRight,
  children,
}: {
  topRight?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-background">
      {topRight ? (
        <div className="absolute right-4 top-4 z-10">{topRight}</div>
      ) : null}
      <div className="flex min-h-full items-center justify-center p-6">
        {children}
      </div>
    </div>
  );
}
