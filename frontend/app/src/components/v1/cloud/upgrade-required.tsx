import { ReactNode } from 'react';

interface UpgradeRequiredProps {
  title: string;
  description: string;
  icon?: ReactNode;
  action?: ReactNode;
}

/**
 * Generic "upgrade required" surface for gating features behind a plan or
 * entitlement. Render a feature-specific icon, copy, and call-to-action via
 * props.
 */
export function UpgradeRequired({
  title,
  description,
  icon,
  action,
}: UpgradeRequiredProps) {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="rounded-lg border bg-card p-12 shadow-sm">
          <div className="mx-auto flex max-w-md flex-col items-center text-center">
            {icon && (
              <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
                {icon}
              </div>
            )}

            <h3 className="mb-2 text-2xl font-semibold">{title}</h3>

            <p className="mb-6 text-muted-foreground">{description}</p>

            {action}
          </div>
        </div>
      </div>
    </div>
  );
}
