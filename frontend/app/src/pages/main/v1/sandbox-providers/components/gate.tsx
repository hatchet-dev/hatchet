import { SUPPORT_URL } from '../lib/kinds';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantEntitlements } from '@/hooks/use-tenant-entitlements';
import { ReactNode } from 'react';

export const SandboxGate = ({ children }: { children: ReactNode }) => {
  const { isControlPlaneEnabled, isControlPlaneLoading } = useControlPlane();
  const { entitlements, isLoading } = useTenantEntitlements();

  if (isLoading || isControlPlaneLoading) {
    return <Loading />;
  }

  if (isControlPlaneEnabled && entitlements?.sandboxOperator) {
    return <>{children}</>;
  }

  return (
    <div className="flex h-full flex-1 items-center justify-center p-4">
      <div className="max-w-lg space-y-3 text-center">
        <h3 className="text-lg font-semibold">Sandboxes are not enabled</h3>
        <p className="text-sm text-muted-foreground">
          {isControlPlaneEnabled
            ? 'Hatchet sandboxes are not enabled for this tenant yet. Contact support to turn them on.'
            : 'Hatchet sandboxes are only available in Hatchet Cloud.'}
        </p>
        {isControlPlaneEnabled && (
          <Button asChild size="sm" variant="outline">
            <a href={SUPPORT_URL}>Contact support</a>
          </Button>
        )}
      </div>
    </div>
  );
};
