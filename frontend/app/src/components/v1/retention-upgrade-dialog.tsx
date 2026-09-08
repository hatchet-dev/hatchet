import { UpgradeGateDialog } from '@/components/v1/cloud/billing/upgrade-gate-dialog';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import useControlPlane from '@/hooks/use-control-plane';
import type { RetentionAttempt } from '@/hooks/use-retention-gate';
import { useTenantDetails } from '@/hooks/use-tenant';
import { docsPages } from '@/lib/generated/docs';
import {
  TIME_WINDOW_LABELS,
  formatRetentionPeriod,
  formatShortDate,
  getRetentionBoundary,
} from '@/lib/utils/retention';

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
  const { isControlPlaneEnabled } = useControlPlane();
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

  if (isControlPlaneEnabled && organizationId) {
    return (
      <UpgradeGateDialog
        open={!!attempt}
        gate="retention"
        organizationId={organizationId}
        onDismiss={onClose}
        retentionAttempt={attempt}
        retentionPeriod={retentionPeriod}
      />
    );
  }

  return (
    <Dialog open={!!attempt} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogTitle>Outside retention window</DialogTitle>
        <DialogDescription className="space-y-2">
          {tried ? <p>{tried}</p> : null}
          <p>
            {`This instance keeps ${label} of data${
              attempt?.kind === 'since' && boundary
                ? ` (since ${formatShortDate(boundary)})`
                : ''
            }.`}
          </p>
          <p>
            Raise SERVER_LIMITS_DEFAULT_TENANT_RETENTION_PERIOD in your config
            if you need a longer window.
          </p>
        </DialogDescription>
        <DocsButton
          doc={docsPages['self-hosting']['data-retention']}
          label="Retention docs"
        />
        <Button variant="ghost" className="w-full" onClick={onClose}>
          {`Keep last ${label}`}
        </Button>
      </DialogContent>
    </Dialog>
  );
}
