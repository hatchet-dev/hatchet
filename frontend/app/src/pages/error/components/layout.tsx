import { SetupCard } from '@/components/layout/setup-card';
import { PropsWithChildren, ReactNode } from 'react';

// Error states render both full-screen (root error boundary) and inside the
// app shell next to the sidebar (resource not found), so this centers a
// SetupCard in flow instead of using the fixed, shell-covering SetupScreen.
export function ErrorPageLayout({
  title,
  description,
  icon,
  actions,
  children,
  className,
}: PropsWithChildren<{
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  actions?: ReactNode;
  className?: string;
}>) {
  return (
    <div className="h-full w-full flex-1 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-6">
        <SetupCard
          title={
            <span className="flex items-center gap-2">
              {icon && (
                <span className="shrink-0 text-muted-foreground">{icon}</span>
              )}
              {title}
            </span>
          }
          description={children ? description : undefined}
          footer={
            actions ? (
              // Callers list the primary action first. Reversing keeps it
              // first in tab order while placing it at the right edge.
              <div className="flex flex-row-reverse flex-wrap gap-2">
                {actions}
              </div>
            ) : undefined
          }
          className={className}
        >
          {children ? (
            <div className="space-y-3">{children}</div>
          ) : (
            // Without detail content the description is the body, so the card
            // does not render an empty padded section.
            <div className="text-sm text-muted-foreground">{description}</div>
          )}
        </SetupCard>
      </div>
    </div>
  );
}
