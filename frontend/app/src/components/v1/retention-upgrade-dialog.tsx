import {
  setupCardDialogClassName,
  SetupCard,
} from '@/components/layout/setup-card';
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
        retentionPeriod={retentionPeriod}
      />
    );
  }

  return (
    <Dialog open={!!attempt} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className={`${setupCardDialogClassName} max-w-xl`}>
        <DialogTitle className="sr-only">Outside retention window</DialogTitle>
        <SetupCard
          className="max-w-none"
          title="Outside retention window"
          description={
            <DialogDescription>
              {tried ? `${tried} ` : ''}
              {`This instance keeps ${label} of data${
                attempt?.kind === 'since' && boundary
                  ? ` (since ${formatShortDate(boundary)})`
                  : ''
              }.`}
            </DialogDescription>
          }
        >
          <div className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              Raise SERVER_LIMITS_DEFAULT_TENANT_RETENTION_PERIOD in your config
              if you need a longer window.
            </p>
            <DocsButton
              doc={docsPages['self-hosting']['data-retention']}
              label="Retention docs"
            />
            <Button variant="ghost" className="w-full" onClick={onClose}>
              {`Keep last ${label}`}
            </Button>
          </div>
        </SetupCard>
      </DialogContent>
    </Dialog>
  );
}
