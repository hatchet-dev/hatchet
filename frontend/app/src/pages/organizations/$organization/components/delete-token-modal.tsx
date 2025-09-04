import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { ManagementToken } from '@/lib/api/generated/cloud/data-contracts';
import { ConfirmDialog } from '@/components/molecules/confirm-dialog';

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
  const { handleApiError } = useApiError({});

  const deleteTokenMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        return;
      }
      await cloudApi.managementTokenDelete(token.id);
    },
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleDelete = () => {
    if (token) {
      deleteTokenMutation.mutate();
    }
  };

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
            <p className="font-medium mb-1">Potential consequences:</p>
            <ul className="list-disc list-inside space-y-1 text-xs">
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
      onSubmit={handleDelete}
      onCancel={() => onOpenChange(false)}
      isLoading={deleteTokenMutation.isPending}
    />
  );
}
