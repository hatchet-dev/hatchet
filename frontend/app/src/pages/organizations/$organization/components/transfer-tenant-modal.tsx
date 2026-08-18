import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

interface TransferTenantModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  organizationName: string;
  tenantId: string;
  tenantName: string;
  // Organizations the current user is an OWNER of, excluding the tenant's
  // current organization. The backend requires the caller to be an OWNER of
  // the destination org too (in addition to the source), so any org the user
  // doesn't own would just 403 -- only offering owned orgs here keeps the
  // picker honest about what will actually work.
  ownedDestinationOrganizations: OrganizationForUser[];
  onSuccess: () => void;
}

export function TransferTenantModal({
  open,
  onOpenChange,
  organizationId,
  organizationName,
  tenantId,
  tenantName,
  ownedDestinationOrganizations,
  onSuccess,
}: TransferTenantModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });
  const [destinationOrganizationId, setDestinationOrganizationId] =
    useState('');
  const [typedName, setTypedName] = useState('');

  useEffect(() => {
    if (!open) {
      setTypedName('');
      setDestinationOrganizationId('');
      setFormErrors([]);
    }
  }, [open, setFormErrors]);

  const selectedOrgName = ownedDestinationOrganizations.find(
    (org) => org.metadata.id === destinationOrganizationId,
  )?.name;

  const previewQueryDescriptor = orgApi.tenantTransferPreviewQuery(
    organizationId,
    tenantId,
    destinationOrganizationId,
  );

  const transferTenantMutation = useMutation({
    ...orgApi.tenantTransferMutation(organizationId, tenantId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:get', organizationId],
      });
      queryClient.invalidateQueries({ queryKey: ['tenant:get'] });
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  // Once the transfer has actually been kicked off, the tenant may already be
  // reparented to the destination org by the time this resolves (or retries),
  // which makes the preview's own organization/tenant path params stale and
  // the request 404 server-side. Stop asking the instant we submit.
  const previewQuery = useQuery({
    ...previewQueryDescriptor,
    enabled:
      open && !!destinationOrganizationId && transferTenantMutation.isIdle,
  });

  const isNameMatch = typedName === tenantName;
  const isPending = transferTenantMutation.isPending;

  const handleSubmit = () => {
    if (isNameMatch && destinationOrganizationId) {
      queryClient.cancelQueries({ queryKey: previewQueryDescriptor.queryKey });
      transferTenantMutation.mutate({ destinationOrganizationId });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Move Tenant to Another Organization</DialogTitle>
          <DialogDescription>
            Move <strong>{tenantName}</strong> out of {organizationName}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <InlineError errors={formErrors} />
          <p className="text-sm text-muted-foreground">
            Moving tenants requires OWNER role on both source and destination
            organizations. Tenant users will be automatically added to
            destination organization.
          </p>
          <div className="grid gap-2">
            <Label htmlFor="destination-organization-select">
              Destination organization
            </Label>
            {ownedDestinationOrganizations.length > 0 ? (
              <Select
                name="destinationOrganizationId"
                value={destinationOrganizationId}
                onValueChange={setDestinationOrganizationId}
                disabled={isPending}
                required
              >
                <SelectTrigger id="destination-organization-select">
                  <SelectValue placeholder="Select an organization" />
                </SelectTrigger>
                <SelectContent>
                  {ownedDestinationOrganizations.map((org) => (
                    <SelectItem key={org.metadata.id} value={org.metadata.id}>
                      {org.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <p className="text-xs text-muted-foreground">
                No destination organizations available.
              </p>
            )}
          </div>
          {destinationOrganizationId && (
            <div className="rounded-md border bg-muted/40 p-3 text-sm">
              {previewQuery.isLoading ? (
                <p className="text-muted-foreground">
                  Checking which members will be added...
                </p>
              ) : previewQuery.isError ? (
                <p className="text-muted-foreground">
                  Couldn&apos;t check which members will be added, but the move
                  itself will still work.
                </p>
              ) : previewQuery.data && previewQuery.data.rows.length > 0 ? (
                <>
                  <p className="mb-2">
                    {previewQuery.data.rows.length} user
                    {previewQuery.data.rows.length !== 1
                      ? 's'
                      : ''} currently{' '}
                    {previewQuery.data.rows.length !== 1 ? 'have' : 'has'}{' '}
                    access to <strong>{tenantName}</strong> and will be added to{' '}
                    <strong>{selectedOrgName}</strong>
                  </p>
                  <ul className="list-inside list-disc space-y-0.5">
                    {previewQuery.data.rows.map((member) => (
                      <li key={member.userId} className="text-muted-foreground">
                        {member.name ? `${member.name} — ` : ''}
                        {member.email}
                      </li>
                    ))}
                  </ul>
                </>
              ) : (
                <p className="text-muted-foreground">
                  No new members will be added — all users in this tenant are
                  already members of {selectedOrgName}.
                </p>
              )}
            </div>
          )}
          <div className="space-y-2 pt-2">
            <label className="text-sm font-medium">
              To confirm, type <strong>{tenantName}</strong>:
            </label>
            <Input
              value={typedName}
              onChange={(e) => setTypedName(e.target.value)}
              placeholder={tenantName}
              className="w-full"
              disabled={isPending}
            />
          </div>
          <div className="flex flex-row justify-end gap-4">
            <Button
              variant="ghost"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!isNameMatch || !destinationOrganizationId || isPending}
            >
              {isPending ? 'Moving...' : 'Move Tenant'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
