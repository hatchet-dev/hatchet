import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { useLocation, useNavigate } from '@tanstack/react-router';
import type { NavigateOptions } from '@tanstack/react-router';
import { FileQuestion, Home, Undo2 } from 'lucide-react';
import { ReactNode } from 'react';

export function ResourceNotFound({
  resource,
  description,
  primaryAction,
}: {
  resource: string;
  description?: ReactNode;
  primaryAction?: {
    label: string;
    navigate: NavigateOptions;
  };
}) {
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <ErrorPageLayout
      icon={<FileQuestion className="h-5 w-5" />}
      title={`${resource} not found`}
      description={
        description ??
        `The ${resource.toLowerCase()} you're looking for doesn’t exist.`
      }
      actions={
        <>
          <Button
            leftIcon={<Home className="h-4 w-4" />}
            onClick={() =>
              primaryAction
                ? navigate({
                    ...primaryAction.navigate,
                    replace: primaryAction.navigate.replace ?? true,
                  })
                : navigate({
                    to: appRoutes.authenticatedRoute.to,
                    replace: true,
                  })
            }
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
