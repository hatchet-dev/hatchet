import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { TenantMemberRoleType } from '@/lib/api/generated/control-plane/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

interface CreateUserGroupModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
}

export function CreateUserGroupModal({
  open,
  onOpenChange,
  organizationId,
}: CreateUserGroupModalProps) {
  const orgApi = useOrganizationApi();
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const [name, setName] = useState('');
  const [role, setRole] = useState<TenantMemberRoleType>(
    TenantMemberRoleType.MEMBER,
  );

  const createMutation = useMutation({
    ...orgApi.userGroupCreateMutation(organizationId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['organization:user-groups:list', organizationId],
      });
      setName('');
      setRole(TenantMemberRoleType.MEMBER);
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Create User Group</DialogTitle>
          <DialogDescription>
            User groups associate tags and a tenant role with a set of org
            members.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="group-name">Name</Label>
            <Input
              id="group-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Engineers"
              disabled={createMutation.isPending}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-role">Tenant role</Label>
            <Select
              value={role}
              onValueChange={(v) => setRole(v as TenantMemberRoleType)}
              disabled={createMutation.isPending}
            >
              <SelectTrigger id="group-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={TenantMemberRoleType.MEMBER}>
                  MEMBER
                </SelectItem>
                <SelectItem value={TenantMemberRoleType.ADMIN}>
                  ADMIN
                </SelectItem>
                <SelectItem value={TenantMemberRoleType.OWNER}>
                  OWNER
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex justify-end gap-3">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={createMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate({ name, role })}
              disabled={createMutation.isPending || !name.trim()}
            >
              {createMutation.isPending ? 'Creating...' : 'Create Group'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
