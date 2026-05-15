import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { ErrorPageLayout } from '@/pages/error/components/layout';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import { CloudIcon } from 'lucide-react';
import { PropsWithChildren } from 'react';

export function ManagedWorkersGate({ children }: PropsWithChildren) {
  const { cloud, featureFlags, isCloudEnabled } = useCloud();
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';
  const navigate = useNavigate();
  const { tenantId } = useCurrentTenantId();

  // Cloud enabled but metadata not yet loaded
  if (isCloudEnabled && !cloud) {
    return null;
  }

  // Feature flag explicitly enables managed workers
  if (managedWorkerEnabled) {
    return <>{children}</>;
  }

  // Self-hosted mode — show informative message
  if (!isCloudEnabled) {
    return (
      <ErrorPageLayout
        icon={<CloudIcon className="h-5 w-5" />}
        title="Managed Compute is not available"
        description="Managed Compute is a cloud-only feature and is not supported in self-hosted deployments. Sign up for Hatchet Cloud to access this feature."
        actions={
          <Button
            onClick={() =>
              navigate({
                to: appRoutes.tenantRunsRoute.to,
                params: { tenant: tenantId },
              })
            }
          >
            Back to Dashboard
          </Button>
        }
      />
    );
  }

  return null;
}
