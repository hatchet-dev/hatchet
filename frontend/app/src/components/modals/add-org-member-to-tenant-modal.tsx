import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Label } from '@/components/v1/ui/label';
import { TenantMember } from '@/lib/api';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { UserPlusIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useState } from 'react';

type AddOrgMemberToTenantModalProps = {
  organizationId: string;
  tenantId: string;
  onClose: () => void;
};

export const AddOrgMemberToTenantModal = ({
  organizationId,
  tenantId,
  onClose,
}: AddOrgMemberToTenantModalProps) => {
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [apiError, setApiError] = useState<string | undefined>();

  const { handleApiError } = useApiError({
    setFieldErrors: () => {},
  });

  const orgApi = useOrganizationApi();
  const { tenantMemberListQuery } = useTenantApi();
  const queryClient = useQueryClient();

  const orgQuery = useQuery({
    ...orgApi.organizationGetQuery(organizationId),
  });

  const tenantMembersQuery = useQuery({
    ...tenantMemberListQuery(tenantId),
  });

  const tenantMemberEmails = new Set(
    (tenantMembersQuery.data?.rows ?? []).map(
      (m: TenantMember) => m.user.email,
    ),
  );

  const members = (orgQuery.data?.members ?? []).filter(
    (m) => !tenantMemberEmails.has(m.email),
  );

  const addMutation = useMutation({
    ...orgApi.organizationTenantMembersAddMutation(organizationId, tenantId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:get', organizationId],
      });
      onClose();
    },
    onError: (err) => {
      handleApiError(err as AxiosError);
      setApiError('Failed to add members. Please try again.');
    },
  });

  const toggleMember = (memberId: string) => {
    setSelectedMemberIds((prev) =>
      prev.includes(memberId)
        ? prev.filter((id) => id !== memberId)
        : [...prev, memberId],
    );
  };

  const handleSubmit = () => {
    if (selectedMemberIds.length === 0) {
      return;
    }
    addMutation.mutate({ memberIds: selectedMemberIds });
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlusIcon className="h-5 w-5" />
            Add Member to Tenant
          </DialogTitle>
          <DialogDescription>
            Select organization members to add directly to this tenant,
            bypassing tag matching.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {orgQuery.isLoading || tenantMembersQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading members...</p>
          ) : members.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              All organization members are already in this tenant.
            </p>
          ) : (
            <div className="max-h-64 overflow-y-auto space-y-2">
              {members.map((member) => (
                <div
                  key={member.metadata.id}
                  className="flex items-center gap-3 rounded-md border px-3 py-2"
                >
                  <Checkbox
                    id={member.metadata.id}
                    checked={selectedMemberIds.includes(member.metadata.id)}
                    onCheckedChange={() => toggleMember(member.metadata.id)}
                  />
                  <Label
                    htmlFor={member.metadata.id}
                    className="cursor-pointer text-sm"
                  >
                    {member.email}
                  </Label>
                </div>
              ))}
            </div>
          )}

          {apiError && <p className="text-sm text-red-500">{apiError}</p>}

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={selectedMemberIds.length === 0 || addMutation.isPending}
            >
              {addMutation.isPending ? 'Adding...' : 'Add Members'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
