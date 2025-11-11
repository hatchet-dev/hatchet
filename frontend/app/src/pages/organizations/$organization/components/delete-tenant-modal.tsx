import { useState, useEffect } from 'react';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizations } from '@/hooks/use-organizations';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/loading';

interface DeleteTenantModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tenant: OrganizationTenant | null;
  tenantName?: string;
  organizationName: string;
  onSuccess: () => void;
}

export function DeleteTenantModal({
  open,
  onOpenChange,
  tenant,
  tenantName,
  organizationName,
  onSuccess,
}: DeleteTenantModalProps) {
  const { handleDeleteTenant, deleteTenantLoading } = useOrganizations();
  const [typedName, setTypedName] = useState('');

  const displayName = tenantName || tenant?.id || '';

  // Reset typed name when modal opens/closes
  useEffect(() => {
    if (!open) {
      setTypedName('');
    }
  }, [open]);

  if (!tenant) {
    return null;
  }

  const isNameMatch = typedName === displayName;

  const handleSubmit = () => {
    if (isNameMatch) {
      handleDeleteTenant(tenant.id, onSuccess, onOpenChange);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit max-w-[80%] min-w-[500px]">
        <DialogHeader>
          <DialogTitle>Archive Tenant</DialogTitle>
        </DialogHeader>
        <div>
          <div className="text-sm text-foreground mb-4 space-y-3">
            <p>
              Are you sure you want to archive <strong>{displayName}</strong>{' '}
              from {organizationName}?
            </p>
            <p className="text-sm text-muted-foreground">
              The tenant will be archived and kept for 30 days before being
              permanently deleted. During this period, you can contact{' '}
              <a
                href="mailto:support@hatchet.run"
                className="text-primary underline"
              >
                support@hatchet.run
              </a>{' '}
              to unarchive the tenant if needed. After 30 days, the tenant and
              all associated data will be permanently removed.
            </p>
            <div className="space-y-2 pt-2">
              <label className="text-sm font-medium">
                To confirm, type <strong>{displayName}</strong>:
              </label>
              <Input
                value={typedName}
                onChange={(e) => setTypedName(e.target.value)}
                placeholder={displayName}
                className="w-full"
                autoFocus
              />
            </div>
          </div>
          <div className="flex flex-row gap-4 justify-end">
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleSubmit}
              disabled={!isNameMatch || deleteTenantLoading}
            >
              {deleteTenantLoading && <Spinner />}
              Archive Tenant
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
