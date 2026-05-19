import { Alert, AlertDescription, AlertTitle } from './alert';
import { Button } from './button';
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
    <Alert
      variant={variant === 'info' ? 'info' : 'default'}
      className={cn(
        variant === 'default' && 'border-border bg-card',
        '[&>svg]:hidden',
        className,
      )}
    >
      {dismissible && (
        <Button
          variant="ghost"
          size="icon"
          onClick={dismiss}
          className="absolute right-2 top-2 size-6 text-muted-foreground"
        >
          <X className="size-3.5" />
        </Button>
      )}

      <div className="flex items-start gap-3">
        {icon && (
          <div className="mt-0.5 shrink-0 text-muted-foreground">{icon}</div>
        )}
        <div className={cn('flex flex-col gap-1', dismissible && 'pr-6')}>
          <AlertTitle className="mb-0">{title}</AlertTitle>
          <AlertDescription className="text-xs leading-snug text-muted-foreground">
            {description}
          </AlertDescription>
          {actions && <div className="mt-2 flex gap-2">{actions}</div>}
        </div>
      </div>
    </Alert>
  );
}
