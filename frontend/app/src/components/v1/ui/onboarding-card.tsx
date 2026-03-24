import { cn } from '@/lib/utils';
import { X } from 'lucide-react';
import { type ReactNode, useCallback, useState } from 'react';

type OnboardingCardProps = {
  title: string;
  description: ReactNode;
  icon?: ReactNode;
  actions?: ReactNode;
  dismissible?: boolean;
  /** localStorage key used to persist dismissal; required when dismissible */
  dismissKey?: string;
  variant?: 'default' | 'info';
  className?: string;
};

function useDismiss(key?: string) {
  const [dismissed, setDismissed] = useState(() => {
    if (!key) {
      return false;
    }
    try {
      return localStorage.getItem(key) === '1';
    } catch {
      return false;
    }
  });

  const dismiss = useCallback(() => {
    if (key) {
      try {
        localStorage.setItem(key, '1');
      } catch {
        // noop
      }
    }
    setDismissed(true);
  }, [key]);

  return { dismissed, dismiss };
}

export function OnboardingCard({
  title,
  description,
  icon,
  actions,
  dismissible = false,
  dismissKey,
  variant = 'default',
  className,
}: OnboardingCardProps) {
  const { dismissed, dismiss } = useDismiss(
    dismissible ? dismissKey : undefined,
  );

  if (dismissed) {
    return null;
  }

  return (
    <div
      className={cn(
        'relative rounded-lg border px-4 py-3 text-sm',
        variant === 'info'
          ? 'border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30'
          : 'border-border bg-card',
        className,
      )}
    >
      {dismissible && (
        <button
          type="button"
          onClick={dismiss}
          className="absolute right-2 top-2 rounded-md p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <X className="size-3.5" />
        </button>
      )}

      <div className="flex items-start gap-3">
        {icon && (
          <div className="mt-0.5 shrink-0 text-muted-foreground">{icon}</div>
        )}
        <div className={cn('flex flex-col gap-1', dismissible && 'pr-6')}>
          <p className="font-medium leading-tight">{title}</p>
          <div className="text-xs leading-snug text-muted-foreground">
            {description}
          </div>
          {actions && <div className="mt-2 flex gap-2">{actions}</div>}
        </div>
      </div>
    </div>
  );
}
