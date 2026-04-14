import { DocsButton } from '@/components/v1/docs/docs-button';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import { docsPages } from '@/lib/generated/docs';
import { formatRetentionPeriod } from '@/lib/utils/retention';
import { appRoutes } from '@/router';
import { Link, useParams } from '@tanstack/react-router';
import { Clock } from 'lucide-react';

interface RetentionBannerProps {
  retentionPeriod: string;
}

export function RetentionBanner({ retentionPeriod }: RetentionBannerProps) {
  const { isCloudEnabled } = useCloud();
  const label = formatRetentionPeriod(retentionPeriod);

  if (isCloudEnabled) {
    return <CloudRetentionBanner label={label} />;
  }

  return <OSSRetentionBanner label={label} />;
}

function CloudRetentionBanner({ label }: { label: string }) {
  const { tenant: tenantId } = useParams({ from: appRoutes.tenantRoute.to });

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
        <div>
          <Link
            to={appRoutes.tenantSettingsBillingRoute.to}
            params={{ tenant: tenantId }}
          >
            <Button size="sm" variant="default">
              View Plans
            </Button>
          </Link>
        </div>
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
