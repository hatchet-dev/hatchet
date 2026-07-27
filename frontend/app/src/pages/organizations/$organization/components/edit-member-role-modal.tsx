import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { InlineError } from '@/components/v1/ui/inline-error';
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
import {
  OrganizationMember,
  OrganizationMemberRoleType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { useState } from 'react';

interface EditMemberRoleModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  member: OrganizationMember;
  organizationName: string;
  /** When the member is the org's only owner, demotion is blocked client-side
   * (the server enforces this too). */
  isLastOwner: boolean;
  onSuccess: () => void;
}

export function EditMemberRoleModal({
  open,
  onOpenChange,
  member,
  organizationName,
  isLastOwner,
  onSuccess,
}: EditMemberRoleModalProps) {
  const [role, setRole] = useState<OrganizationMemberRoleType>(member.role);
  const [formErrors, setFormErrors] = useState<string[]>([]);
  const { handleApiError } = useApiError({ setErrors: setFormErrors });
  const orgApi = useOrganizationApi();

  const memberUpdate = orgApi.organizationMemberUpdateMutation(
    member.metadata.id,
  );
  const updateMutation = useMutation({
    ...memberUpdate,
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    // Keep the dialog open so the inline error is visible (a toast would
    // render behind the overlay).
    onError: (error: AxiosError) => handleApiError(error),
  });

  const handleSubmit = () => {
    setFormErrors([]);
    updateMutation.mutate({ role });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-fit min-w-[500px] max-w-[80%]">
        <DialogHeader>
          <DialogTitle>Change member role</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4">
          <InlineError errors={formErrors} />
          <p className="text-sm text-muted-foreground">
            Change the role of this member in {organizationName}.
          </p>
          <div className="grid gap-2">
            <Label htmlFor="member-email">Email</Label>
            <Input
              readOnly
              id="member-email"
              type="email"
              value={member.email}
              disabled={true}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="member-role">Role</Label>
            <Select
              value={role}
              onValueChange={(value) =>
                setRole(value as OrganizationMemberRoleType)
              }
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue id="member-role" placeholder="Role..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={OrganizationMemberRoleType.OWNER}>
                  Owner
                </SelectItem>
                <SelectItem
                  value={OrganizationMemberRoleType.MEMBER}
                  disabled={isLastOwner}
                >
                  Member
                </SelectItem>
              </SelectContent>
            </Select>
            {isLastOwner && (
              <p className="text-xs text-muted-foreground">
                An organization must have at least one owner.
              </p>
            )}
          </div>
          <Button
            onClick={handleSubmit}
            disabled={updateMutation.isPending || role === member.role}
          >
            {updateMutation.isPending && <Spinner />}
            Update member
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
