import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { useLocation, useNavigate } from '@tanstack/react-router';
import { RefreshCw, TriangleAlert } from 'lucide-react';

export function GenericError({
  status,
  statusText,
}: {
  status?: number;
  statusText?: string;
}) {
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <ErrorPageLayout
      icon={<TriangleAlert className="size-4" />}
      title="Something went wrong"
      description={statusText || 'An unexpected error occurred.'}
      actions={
        <>
          <Button
            size="sm"
            leftIcon={<RefreshCw className="h-4 w-4" />}
            onClick={() => window.location.reload()}
          >
            Reload
          </Button>
          <Button
            size="sm"
            onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
            variant="outline"
          >
            Dashboard
          </Button>
        </>
      }
    >
      <div className="flex">
        <Badge variant="secondary" className="font-mono">
          {status ?? 'error'}
        </Badge>
      </div>

      <div className="w-full rounded-md border bg-muted/20 p-3 text-left font-mono text-xs text-muted-foreground">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
          Path
        </div>
        <div className="break-all text-foreground/90">{location.pathname}</div>
      </div>
    </ErrorPageLayout>
  );
}
