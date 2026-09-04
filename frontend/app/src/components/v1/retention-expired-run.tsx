import { DocsButton } from '@/components/v1/docs/docs-button';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/v1/ui/breadcrumb';
import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { docsPages } from '@/lib/generated/docs';
import { formatRetentionPeriod, formatShortDate } from '@/lib/utils/retention';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';

type RetentionExpiredRunProps = {
  retentionPeriod: string;
  createdAt?: string | Date;
  runId: string;
};

export function RetentionExpiredRun({
  retentionPeriod,
  createdAt,
  runId,
}: RetentionExpiredRunProps) {
  const { tenantId } = useCurrentTenantId();
  const { organizationId } = useTenantDetails();
  const { isControlPlaneEnabled, canBill } = useControlPlane();
  const label = formatRetentionPeriod(retentionPeriod);

  return (
    <div className="flex flex-col gap-6 p-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link
                to={appRoutes.tenantRoute.to}
                params={{ tenant: tenantId }}
              >
                Home
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link
                to={appRoutes.tenantRunsRoute.to}
                params={{ tenant: tenantId }}
              >
                Runs
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage className="font-mono text-xs">{runId}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="mx-auto w-full max-w-lg rounded-lg border bg-background p-6 shadow-sm">
        <h2 className="text-lg font-semibold">
          This run is outside your {label} retention.
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          {createdAt
            ? `It was created ${formatShortDate(createdAt)}. Data from that window is no longer available.`
            : 'Data from that window is no longer available.'}
        </p>
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <Link
            to={appRoutes.tenantRunsRoute.to}
            params={{ tenant: tenantId }}
          >
            <Button variant="ghost">Back to Runs</Button>
          </Link>
          {isControlPlaneEnabled && canBill && organizationId ? (
            <Link
              to={appRoutes.organizationBillingRoute.to}
              params={{ organization: organizationId }}
            >
              <Button>View plans</Button>
            </Link>
          ) : !isControlPlaneEnabled ? (
            <DocsButton
              doc={docsPages['self-hosting']['data-retention']}
              label="Retention docs"
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}
