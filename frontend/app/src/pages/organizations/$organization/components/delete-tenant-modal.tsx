import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Spinner } from '@/components/v1/ui/loading';
import { useOrganizations } from '@/hooks/use-organizations';
import { OrganizationTenant } from '@/lib/api/generated/cloud/data-contracts';
import { useState, useEffect } from 'react';

interface DeleteTenantModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tenant: OrganizationTenant | null;
  tenantName: string;
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

  // Reset typed name when modal opens/closes
  useEffect(() => {
    if (!open) {
      setTypedName('');
    }
  }, [open]);

  if (!tenant) {
    return null;
  }

  const isNameMatch = typedName === tenantName;

  const handleSubmit = () => {
    if (isNameMatch) {
      handleDeleteTenant(tenant.id, onSuccess, onOpenChange);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Archive Tenant</DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 space-y-3 text-sm text-foreground">
            <p>
              Are you sure you want to archive <strong>{tenantName}</strong>{' '}
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
                To confirm, type <strong>{tenantName}</strong>:
              </label>
              <Input
                value={typedName}
                onChange={(e) => setTypedName(e.target.value)}
                placeholder={tenantName}
                className="w-full"
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
