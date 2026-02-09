import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useOrganizations } from '@/hooks/use-organizations';
import { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';

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
  const { handleCancelInvite, cancelInviteLoading } = useOrganizations();

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
      onSubmit={() =>
        handleCancelInvite(invite.metadata.id, onSuccess, onOpenChange)
      }
      onCancel={() => onOpenChange(false)}
      isLoading={cancelInviteLoading}
    />
  );
}
