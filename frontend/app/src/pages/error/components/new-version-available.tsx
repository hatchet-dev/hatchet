import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { useLocation, useNavigate } from '@tanstack/react-router';
import { CloudDownload, Home } from 'lucide-react';

export function NewVersionAvailable() {
  const navigate = useNavigate();
  const location = useLocation();

  const queryParams = new URLSearchParams(location.search);

  if (!queryParams.has('updated')) {
    queryParams.set('updated', 'true');
    const updatedUrl = `${location.pathname}?${queryParams.toString()}`;
    window.location.href = updatedUrl;
  }

  return (
    <ErrorPageLayout
      icon={<CloudDownload className="h-6 w-6" />}
      title="Update available"
      description="A new version of the app is available."
      actions={
        <>
          <Button onClick={() => window.location.reload()}>Reload</Button>
          <Button
            leftIcon={<Home className="h-4 w-4" />}
            onClick={() => navigate({ to: appRoutes.authenticatedRoute.to })}
            variant="outline"
          >
            Dashboard
          </Button>
        </>
      }
    >
      <div className="flex justify-center">
        <Badge variant="secondary" className="font-mono">
          update
        </Badge>
      </div>

      <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-3 text-left font-mono text-xs text-muted-foreground">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
          Path
        </div>
        <div className="break-all text-foreground/90">{location.pathname}</div>
      </div>
    </ErrorPageLayout>
  );
}
