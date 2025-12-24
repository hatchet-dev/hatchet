import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useOrganizations } from '@/hooks/use-organizations';
import { ManagementToken } from '@/lib/api/generated/cloud/data-contracts';

interface DeleteTokenModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  token: ManagementToken | null;
  organizationName: string;
  onSuccess: () => void;
}

export function DeleteTokenModal({
  open,
  onOpenChange,
  token,
  organizationName,
  onSuccess,
}: DeleteTokenModalProps) {
  const { handleDeleteToken, deleteTokenLoading } = useOrganizations();

  if (!token) {
    return null;
  }

  return (
    <ConfirmDialog
      isOpen={open}
      title="Delete Management Token"
      description={
        <div className="space-y-3">
          <p>
            Are you sure you want to delete the management token{' '}
            <strong>"{token.name}"</strong> from {organizationName}?
          </p>
          <p className="text-sm text-muted-foreground">
            This action cannot be undone. Any applications or services using
            this token will immediately lose access and may break functionality.
          </p>
          <div className="text-sm text-muted-foreground">
            <p className="mb-1 font-medium">Potential consequences:</p>
            <ul className="list-inside list-disc space-y-1 text-xs">
              <li>API integrations may fail</li>
              <li>Automated deployments could break</li>
              <li>CI/CD pipelines might stop working</li>
              <li>Third-party tools will lose access</li>
            </ul>
          </div>
        </div>
      }
      submitLabel="Delete Token"
      submitVariant="destructive"
      cancelLabel="Cancel"
      onSubmit={() => handleDeleteToken(token.id, onSuccess, onOpenChange)}
      onCancel={() => onOpenChange(false)}
      isLoading={deleteTokenLoading}
    />
  );
}
