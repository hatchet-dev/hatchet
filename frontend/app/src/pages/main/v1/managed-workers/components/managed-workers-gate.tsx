import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import { ErrorPageLayout } from '@/pages/error/components/layout';
import { appRoutes } from '@/router';
import { useParams } from '@tanstack/react-router';
import { Cloud, LifeBuoy, Mail, Undo2 } from 'lucide-react';
import { PropsWithChildren } from 'react';

export function ManagedWorkersGate({ children }: PropsWithChildren) {
  const params = useParams({ from: appRoutes.tenantRoute.to });
  const { cloud, featureFlags, isCloudEnabled } = useCloud(params.tenant);
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';

  if (isCloudEnabled && !cloud) {
    return null;
  }

  if (managedWorkerEnabled) {
    return <>{children}</>;
  }

  if (!isCloudEnabled) {
    return (
      <ErrorPageLayout
        icon={<Cloud className="h-5 w-5" />}
        title="Managed Workers is not available"
        description="Managed Workers are only available in Hatchet Cloud."
        actions={
          <Button
            leftIcon={<Undo2 className="h-4 w-4" />}
            variant="outline"
            onClick={() => window.history.back()}
          >
            Go back
          </Button>
        }
      />
    );
  }

  return (
    <ErrorPageLayout
      icon={<LifeBuoy className="h-5 w-5" />}
      title="Managed Workers not enabled"
      description="Managed Workers aren't enabled for your tenant. Contact support for more information."
      actions={
        <>
          <Button
            leftIcon={<Mail className="h-4 w-4" />}
            onClick={() => window.open('mailto:support@hatchet.run', '_blank')}
          >
            Contact us
          </Button>
          <Button
            leftIcon={<Undo2 className="h-4 w-4" />}
            variant="outline"
            onClick={() => window.history.back()}
          >
            Go back
          </Button>
        </>
      }
    />
  );
}
