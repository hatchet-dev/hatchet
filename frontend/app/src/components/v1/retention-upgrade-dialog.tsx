import { DocsButton } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
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

  return (
    <Dialog open={!!attempt} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isControlPlaneEnabled
              ? 'Need more history?'
              : 'Outside retention window'}
          </DialogTitle>
          <DialogDescription asChild>
            <div className="space-y-2 text-left text-sm text-gray-700 dark:text-gray-300">
              <p>{tried}</p>
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
            </div>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2 sm:justify-end">
          <Button variant="ghost" onClick={onClose}>
            {keepLabel}
          </Button>
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
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
