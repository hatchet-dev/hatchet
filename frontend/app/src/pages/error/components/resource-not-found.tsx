import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { useLocation, useNavigate } from '@tanstack/react-router';
import type { NavigateOptions } from '@tanstack/react-router';
import { FileQuestion, Home, LucideIcon, Undo2 } from 'lucide-react';
import { ReactNode, useState } from 'react';

export function ResourceNotFound({
  resource,
  description,
  primaryAction,
}: {
  resource: string;
  description?: ReactNode;
  primaryAction?: {
    label: string;
    navigate?: NavigateOptions;
    icon?: LucideIcon;
    actionOverride?: () => void;
  };
}) {
  const navigate = useNavigate();
  const location = useLocation();
  const Icon = primaryAction?.icon ?? Home;
  const [spinning, setSpinning] = useState(false);

  const handleButtonClick = () => {
    if (!primaryAction) {
      navigate({
        to: appRoutes.authenticatedRoute.to,
        replace: true,
      });
      return;
    }

    const override = primaryAction.actionOverride;
    if (override) {
      setSpinning(true);
      override();
      return;
    }

    if (primaryAction.navigate) {
      navigate({
        ...primaryAction.navigate,
        replace: primaryAction.navigate.replace ?? true,
      });

      return;
    }

    throw new Error('unhandled action');
  };

  return (
    <ErrorPageLayout
      icon={<FileQuestion className="h-5 w-5" />}
      title={`${resource} not found`}
      description={
        description ??
        `The ${resource.toLowerCase()} you're looking for doesn't exist.`
      }
      actions={
        <>
          <Button
            leftIcon={
              <Icon
                className="h-4 w-4"
                style={
                  spinning ? { animation: 'spin 0.5s linear 1' } : undefined
                }
                onAnimationEnd={() => setSpinning(false)}
              />
            }
            onClick={handleButtonClick}
          >
            {primaryAction?.label ?? 'Dashboard'}
          </Button>
          <Button
            leftIcon={<Undo2 className="h-4 w-4" />}
            onClick={() => window.history.back()}
            variant="outline"
          >
            Go back
          </Button>
        </>
      }
    >
      <div className="flex justify-center">
        <Badge variant="secondary" className="font-mono">
          404
        </Badge>
      </div>
      <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-3 text-left font-mono text-xs text-muted-foreground">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
          Requested path
        </div>
        <div className="break-all text-foreground/90">{location.pathname}</div>
      </div>
    </ErrorPageLayout>
  );
}
