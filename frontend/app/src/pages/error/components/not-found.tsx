import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { getOptionalStringParam } from '@/lib/router-helpers';
import { appRoutes } from '@/router';
import { useLocation, useNavigate, useParams } from '@tanstack/react-router';
import { FileQuestion, Home, Undo2 } from 'lucide-react';

export function NotFound() {
  const navigate = useNavigate();
  const location = useLocation();
  const params = useParams({ strict: false });
  const tenant = getOptionalStringParam(params, 'tenant');

  return (
    <ErrorPageLayout
      icon={<FileQuestion className="h-5 w-5" />}
      title="Page not found"
      description="This page doesn’t exist or may have moved."
      actions={
        <>
          <Button
            leftIcon={<Home className="h-4 w-4" />}
            onClick={() =>
              tenant
                ? navigate({
                    to: appRoutes.tenantRunsRoute.to,
                    params: { tenant },
                    replace: true,
                  })
                : navigate({
                    to: appRoutes.authenticatedRoute.to,
                    replace: true,
                  })
            }
          >
            Dashboard
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
