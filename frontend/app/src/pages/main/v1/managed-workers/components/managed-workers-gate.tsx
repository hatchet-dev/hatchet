import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import { ErrorPageLayout } from '@/pages/error/components/layout';
import { CloudIcon } from 'lucide-react';
import { Undo2 } from 'lucide-react';
import { PropsWithChildren } from 'react';

export function ManagedWorkersGate({ children }: PropsWithChildren) {
  const { cloud, featureFlags, isCloudEnabled } = useCloud();
  const managedWorkerEnabled = featureFlags?.['managed-worker'] === 'true';

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
        description="Managed Compute is a cloud-only feature and is not supported in self-hosted deployments."
        actions={
          <Button
            leftIcon={<Undo2 className="h-4 w-4" />}
            onClick={() => window.history.back()}
            variant="outline"
          >
            Go back
          </Button>
        }
      />
    );
  }

  // Cloud user but managed-worker feature not enabled for this account
  return (
    <ErrorPageLayout
      icon={<CloudIcon className="h-5 w-5" />}
      title="Managed Compute is not enabled"
      description="Managed Compute is not enabled for your account. Please contact support@hatchet.run to enable this feature."
      actions={
        <Button
          leftIcon={<Undo2 className="h-4 w-4" />}
          onClick={() => window.history.back()}
          variant="outline"
        >
          Go back
        </Button>
      }
    />
  );
}