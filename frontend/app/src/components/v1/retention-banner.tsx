import { DocsButton } from '@/components/v1/docs/docs-button';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantDetails } from '@/hooks/use-tenant';
import { docsPages } from '@/lib/generated/docs';
import { formatRetentionPeriod } from '@/lib/utils/retention';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { Clock } from 'lucide-react';

interface RetentionBannerProps {
  retentionPeriod: string;
}

export function RetentionBanner({ retentionPeriod }: RetentionBannerProps) {
  const { isControlPlaneEnabled } = useControlPlane();
  const label = formatRetentionPeriod(retentionPeriod);

  if (isControlPlaneEnabled) {
    return <CloudRetentionBanner label={label} />;
  }

  return <OSSRetentionBanner label={label} />;
}

function CloudRetentionBanner({ label }: { label: string }) {
  const { organizationId } = useTenantDetails();
  const { canBill } = useControlPlane();

  return (
    <Alert>
      <Clock className="size-4" />
      <AlertTitle>Data outside retention window</AlertTitle>
      <AlertDescription className="flex flex-col gap-3">
        <span>
          Your current plan retains data for {label}. Data outside this window
          is no longer available. Upgrade your plan to extend your retention
          period.
        </span>
        {canBill && organizationId ? (
          <div>
            <Link
              to={appRoutes.organizationBillingRoute.to}
              params={{ organization: organizationId }}
            >
              <Button size="sm" variant="default">
                View Plans
              </Button>
            </Link>
          </div>
        ) : null}
      </AlertDescription>
    </Alert>
  );
}

function OSSRetentionBanner({ label }: { label: string }) {
  return (
    <Alert>
      <Clock className="size-4" />
      <AlertTitle>Data outside retention window</AlertTitle>
      <AlertDescription className="flex flex-col gap-3">
        <span>
          Your instance retains data for {label}. Data outside this window has
          been pruned and is no longer available. You can adjust the retention
          period in your server configuration.
        </span>
        <div className="w-fit">
          <DocsButton
            doc={docsPages['self-hosting']['data-retention']}
            label="Data retention docs"
          />
        </div>
      </AlertDescription>
    </Alert>
  );
}
