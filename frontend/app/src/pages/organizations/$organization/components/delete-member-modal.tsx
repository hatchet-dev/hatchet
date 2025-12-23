import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useOrganizations } from '@/hooks/use-organizations';
import { OrganizationMember } from '@/lib/api/generated/cloud/data-contracts';

interface DeleteMemberModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  member: OrganizationMember | null;
  organizationName: string;
  onSuccess: () => void;
}

export function DeleteMemberModal({
  open,
  onOpenChange,
  member,
  organizationName,
  onSuccess,
}: DeleteMemberModalProps) {
  const { handleDeleteMember, deleteMemberLoading } = useOrganizations();

  if (!member) {
    return null;
  }

  return (
    <ConfirmDialog
      isOpen={open}
      title="Remove Member"
      description={
        <div className="space-y-3">
          <p>
            Are you sure you want to remove <strong>{member.email}</strong> from{' '}
            {organizationName}?
          </p>
          <p className="text-sm text-muted-foreground">
            This action cannot be undone. The member will lose access to this
            organization and all its tenants immediately.
          </p>
        </div>
      }
      submitLabel="Remove Member"
      submitVariant="destructive"
      cancelLabel="Cancel"
      onSubmit={() => handleDeleteMember(member, onSuccess, onOpenChange)}
      onCancel={() => onOpenChange(false)}
      isLoading={deleteMemberLoading}
    />
  );
}
