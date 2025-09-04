import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { OrganizationMember } from '@/lib/api/generated/cloud/data-contracts';
import { ConfirmDialog } from '@/components/molecules/confirm-dialog';

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
  const { handleApiError } = useApiError({});

  const deleteMemberMutation = useMutation({
    mutationFn: async () => {
      if (!member) {
        return;
      }
      await cloudApi.organizationMemberDelete(member.metadata.id, {
        emails: [member.email],
      });
    },
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleDelete = () => {
    if (member) {
      deleteMemberMutation.mutate();
    }
  };

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
      onSubmit={handleDelete}
      onCancel={() => onOpenChange(false)}
      isLoading={deleteMemberMutation.isPending}
    />
  );
}
