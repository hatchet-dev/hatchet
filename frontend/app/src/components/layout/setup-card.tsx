import { cn } from '@/lib/utils';
import { type ReactNode } from 'react';

// Narrow by default because most callers hold a short form; the onboarding
// flow passes a wider max-width for its stepper.
export function SetupCard({
  title,
  description,
  children,
  footer,
  className,
  bodyClassName,
}: {
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  className?: string;
  bodyClassName?: string;
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
        <div className={cn('px-6 py-5', bodyClassName)}>{children}</div>
        {footer ? (
          <div className="flex items-center justify-end gap-2 border-t border-border px-6 py-4">
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}

// `fixed` so the screen covers the app shell (nav and sidebar) on routes that
// render inside it.
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
