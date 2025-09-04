import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';
import { ConfirmDialog } from '@/components/molecules/confirm-dialog';

interface CancelInviteModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invite: OrganizationInvite | null;
  organizationName: string;
  onSuccess: () => void;
}

export function CancelInviteModal({
  open,
  onOpenChange,
  invite,
  organizationName,
  onSuccess,
}: CancelInviteModalProps) {
  const { handleApiError } = useApiError({});

  const cancelInviteMutation = useMutation({
    mutationFn: async () => {
      if (!invite) {
        return;
      }
      await cloudApi.organizationInviteDelete(invite.metadata.id);
    },
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleCancel = () => {
    if (invite) {
      cancelInviteMutation.mutate();
    }
  };

  if (!invite) {
    return null;
  }

  return (
    <ConfirmDialog
      isOpen={open}
      title="Cancel Invitation"
      description={
        <div className="space-y-3">
          <p>
            Are you sure you want to cancel the invitation for{' '}
            <strong>{invite.inviteeEmail}</strong> to join {organizationName}?
          </p>
          <p className="text-sm text-muted-foreground">
            This action cannot be undone. The invitation will be permanently
            deleted and the invited user will no longer be able to accept it.
          </p>
        </div>
      }
      submitLabel="Cancel Invitation"
      submitVariant="destructive"
      cancelLabel="Keep Invitation"
      onSubmit={handleCancel}
      onCancel={() => onOpenChange(false)}
      isLoading={cancelInviteMutation.isPending}
    />
  );
}
