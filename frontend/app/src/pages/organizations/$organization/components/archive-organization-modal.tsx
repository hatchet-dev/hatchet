import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { SubscriptionPlanCode } from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useState, useEffect } from 'react';

interface ArchiveOrganizationModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  organizationName: string;
  activeTenantCount: number;
  subscriptionPlan?: SubscriptionPlanCode;
  onSuccess: () => void;
}

export function ArchiveOrganizationModal({
  open,
  onOpenChange,
  organizationId,
  organizationName,
  activeTenantCount,
  subscriptionPlan,
  onSuccess,
}: ArchiveOrganizationModalProps) {
  const orgApi = useOrganizationApi();
  const { handleApiError } = useApiError({});
  const [typedName, setTypedName] = useState('');

  const archiveOrganizationMutation = useMutation({
    ...orgApi.organizationArchiveMutation(organizationId),
    onSuccess: () => {
      onOpenChange(false);
      onSuccess();
    },
    onError: handleApiError,
  });

  // Reset typed name when modal opens/closes
  useEffect(() => {
    if (!open) {
      setTypedName('');
    }
  }, [open]);

  // Mirrors the rules enforced by the archive endpoint, so the user sees why
  // archiving is blocked instead of the request failing on submit.
  const hasActiveSubscription =
    !!subscriptionPlan && subscriptionPlan !== SubscriptionPlanCode.Free;

  const blockers: string[] = [];
  if (hasActiveSubscription) {
    blockers.push(
      'This organization has an active subscription. Cancel it before archiving the organization.',
    );
  }
  if (activeTenantCount > 0) {
    blockers.push(
      `This organization has ${activeTenantCount} active ${activeTenantCount === 1 ? 'tenant' : 'tenants'}. Archive ${activeTenantCount === 1 ? 'it' : 'them'} before archiving the organization.`,
    );
  }

  const isBlocked = blockers.length > 0;
  const isNameMatch = typedName === organizationName;

  const handleSubmit = () => {
    if (isNameMatch && !isBlocked) {
      archiveOrganizationMutation.mutate();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Archive Organization</DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 space-y-3 text-sm text-foreground">
            <p>
              Are you sure you want to archive{' '}
              <strong>{organizationName}</strong>?
            </p>
            <p className="text-sm text-muted-foreground">
              Members will lose access to this organization. Contact{' '}
              <a
                href="mailto:support@hatchet.run"
                className="text-primary underline"
              >
                support@hatchet.run
              </a>{' '}
              if you need it restored.
            </p>
            {isBlocked && (
              <Alert variant="warn">
                <ExclamationTriangleIcon className="size-4" />
                <AlertTitle>
                  This organization cannot be archived yet
                </AlertTitle>
                <AlertDescription className="flex flex-col gap-1">
                  {blockers.map((blocker) => (
                    <span key={blocker}>{blocker}</span>
                  ))}
                </AlertDescription>
              </Alert>
            )}
            <div className="space-y-2 pt-2">
              <label className="text-sm font-medium">
                To confirm, type <strong>{organizationName}</strong>:
              </label>
              <Input
                value={typedName}
                onChange={(e) => setTypedName(e.target.value)}
                placeholder={organizationName}
                className="w-full"
                disabled={isBlocked}
                autoFocus
              />
            </div>
          </div>
          <div className="flex flex-row justify-end gap-4">
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleSubmit}
              disabled={
                !isNameMatch ||
                isBlocked ||
                archiveOrganizationMutation.isPending
              }
            >
              {archiveOrganizationMutation.isPending && <Spinner />}
              Archive Organization
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
