import { Copy, t } from './copy';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantEntitlements } from '@/hooks/use-tenant-entitlements';
import { ReactNode } from 'react';

/**
 * Renders the Endpoints pages only for a tenant with the serverless operator entitlement. The
 * side nav hides the tab otherwise; this covers links straight to an Endpoints page.
 */
export const EntitlementGate = ({ children }: { children: ReactNode }) => {
  const { isSelfHosted } = useControlPlane();
  const { entitlements, isLoading } = useTenantEntitlements();

  if (isLoading) {
    return <Loading />;
  }

  if (entitlements?.serverlessOperator) {
    return <>{children}</>;
  }

  return (
    <div className="flex h-full flex-1 items-center justify-center p-4">
      <div className="max-w-lg space-y-3 text-center">
        <h3 className="text-lg font-semibold">
          {t('create.notEntitled.title')}
        </h3>
        <p className="text-sm text-muted-foreground">
          <Copy
            k={
              isSelfHosted
                ? 'create.notEntitled.oss'
                : 'create.notEntitled.cloud'
            }
          />
        </p>
        {!isSelfHosted && (
          <Button asChild size="sm" variant="outline">
            <a href={t('create.notEntitled.supportUrl')}>
              {t('create.notEntitled.contactSupport')}
            </a>
          </Button>
        )}
      </div>
    </div>
  );
};
