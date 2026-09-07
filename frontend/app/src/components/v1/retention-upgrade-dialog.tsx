import {
  UpgradeRequiredLayout,
  formatPlanTier,
  useCurrentPlanName,
} from '@/components/v1/cloud/billing/upgrade-required';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
import { Dialog, DialogContent } from '@/components/v1/ui/dialog';
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantDetails } from '@/hooks/use-tenant';
import type { RetentionAttempt } from '@/hooks/use-retention-gate';
import { docsPages } from '@/lib/generated/docs';
import {
  TIME_WINDOW_LABELS,
  formatRetentionPeriod,
  formatShortDate,
  getRetentionBoundary,
} from '@/lib/utils/retention';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';

type RetentionUpgradeDialogProps = {
  attempt: RetentionAttempt | null;
  retentionPeriod?: string;
  onClose: () => void;
};

export function RetentionUpgradeDialog({
  attempt,
  retentionPeriod,
  onClose,
}: RetentionUpgradeDialogProps) {
  const { isControlPlaneEnabled, canBill } = useControlPlane();
  const { organizationId } = useTenantDetails();
  const currentPlanName = useCurrentPlanName(organizationId);
  const tier = formatPlanTier(currentPlanName);
  const label = retentionPeriod
    ? formatRetentionPeriod(retentionPeriod)
    : 'your current window';
  const boundary = retentionPeriod
    ? getRetentionBoundary(retentionPeriod)
    : null;

  const tried = attempt
    ? attempt.kind === 'preset'
      ? `You tried to view the last ${TIME_WINDOW_LABELS[attempt.window]}.`
      : `You tried to look back to ${formatShortDate(attempt.date)}.`
    : '';

  const keepLabel = `Keep last ${label}`;
  const canUpgrade = isControlPlaneEnabled && canBill && !!organizationId;

  return (
    <Dialog open={!!attempt} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <UpgradeRequiredLayout
          title={
            isControlPlaneEnabled
              ? `You've reached the ${tier}'s retention limit`
              : 'Outside retention window'
          }
          description={
            <>
              {tried ? <p>{tried}</p> : null}
              <p>
                {`${isControlPlaneEnabled ? 'This tenant' : 'This instance'} keeps ${label} of data${
                  attempt?.kind === 'since' && boundary
                    ? ` (since ${formatShortDate(boundary)})`
                    : ''
                }.`}
              </p>
              {isControlPlaneEnabled ? (
                <p>Upgrade to search further back.</p>
              ) : (
                <p>
                  Raise SERVER_LIMITS_DEFAULT_TENANT_RETENTION_PERIOD in your
                  config if you need a longer window.
                </p>
              )}
            </>
          }
          summary={
            isControlPlaneEnabled && (currentPlanName || retentionPeriod) ? (
              <>
                {currentPlanName ? (
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Current plan</span>
                    <span className="font-medium text-foreground">
                      {currentPlanName}
                    </span>
                  </div>
                ) : null}
                {retentionPeriod ? (
                  <div
                    className={
                      currentPlanName
                        ? 'mt-2 flex items-center justify-between'
                        : 'flex items-center justify-between'
                    }
                  >
                    <span className="text-muted-foreground">Retention</span>
                    <span className="font-medium text-foreground">
                      {label}
                    </span>
                  </div>
                ) : null}
              </>
            ) : undefined
          }
        >
          {canUpgrade ? (
            <Link
              to={appRoutes.organizationBillingRoute.to}
              params={{ organization: organizationId }}
              className="w-full"
              onClick={onClose}
            >
              <Button size="lg" className="w-full">
                View plans &amp; upgrade
              </Button>
            </Link>
          ) : !isControlPlaneEnabled ? (
            <DocsButton
              doc={docsPages['self-hosting']['data-retention']}
              label="Retention docs"
            />
          ) : null}
          <Button variant="ghost" className="mt-2 w-full" onClick={onClose}>
            {keepLabel}
          </Button>
        </UpgradeRequiredLayout>
      </DialogContent>
    </Dialog>
  );
}
