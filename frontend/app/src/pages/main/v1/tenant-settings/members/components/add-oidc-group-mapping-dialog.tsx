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
import { Spinner } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { TenantMemberRole } from '@/lib/api';
import { useTenantApi } from '@/lib/api/tenant-wrapper';
import { useApiError } from '@/lib/hooks';
import { PlusIcon } from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { FormEvent, useState } from 'react';

type AddOIDCGroupMappingDialogProps = Readonly<{
  tenantId: string;
  onCreated: () => Promise<unknown>;
}>;

export function AddOIDCGroupMappingDialog({
  tenantId,
  onCreated,
}: AddOIDCGroupMappingDialogProps) {
  const { tenantOIDCGroupMappingCreateMutation } = useTenantApi();
  const { handleApiError } = useApiError({});
  const [open, setOpen] = useState(false);
  const [group, setGroup] = useState('');
  const [role, setRole] = useState<TenantMemberRole>(TenantMemberRole.MEMBER);
  const createMapping = useMutation({
    ...tenantOIDCGroupMappingCreateMutation(tenantId),
    onSuccess: async () => {
      await onCreated();
      setGroup('');
      setRole(TenantMemberRole.MEMBER);
      setOpen(false);
    },
    onError: handleApiError,
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmedGroup = group.trim();
    if (trimmedGroup) {
      createMapping.mutate({ group: trimmedGroup, role });
    }
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (createMapping.isPending) {
      return;
    }
    setOpen(nextOpen);
    if (!nextOpen) {
      setGroup('');
      setRole(TenantMemberRole.MEMBER);
    }
  };

  return (
    <>
      <Button
        type="button"
        variant="link"
        className="h-auto p-0"
        leftIcon={<PlusIcon className="size-4" />}
        onClick={() => setOpen(true)}
      >
        Add group
      </Button>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add OIDC group</DialogTitle>
            <DialogDescription>
              Grant an identity provider group access to this tenant on the next
              sign-in.
            </DialogDescription>
          </DialogHeader>
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <Label htmlFor="oidc-group">Group</Label>
              <Input
                id="oidc-group"
                value={group}
                onChange={(event) => setGroup(event.target.value)}
                placeholder="engineering"
                maxLength={255}
                disabled={createMapping.isPending}
                autoFocus
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="oidc-role">Tenant role</Label>
              <Select
                value={role}
                onValueChange={(value) => setRole(value as TenantMemberRole)}
                disabled={createMapping.isPending}
              >
                <SelectTrigger id="oidc-role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={TenantMemberRole.ADMIN}>Admin</SelectItem>
                  <SelectItem value={TenantMemberRole.MEMBER}>
                    Member
                  </SelectItem>
                  <SelectItem value={TenantMemberRole.VIEWER}>
                    Viewer
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={createMapping.isPending}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={!group.trim() || createMapping.isPending}
              >
                {createMapping.isPending && <Spinner />}
                Add group
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
