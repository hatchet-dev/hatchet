import { APIToken } from '@/lib/api';
import { DestructiveDialog } from '@/components/ui/dialog/destructive-dialog';
import { Code } from '@/components/ui/code';

interface RevokeTokenFormProps {
  apiToken: APIToken;
  isLoading: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RevokeTokenForm({
  apiToken,
  isLoading,
  onSubmit,
  onCancel,
  open,
  onOpenChange,
}: RevokeTokenFormProps) {
  // Create code representation of the token
  const tokenCode = `{
  "name": "${apiToken.name}",
  "createdAt": "${new Date(apiToken.metadata.createdAt).toISOString()}",
  "expiresAt": "${new Date(apiToken.expiresAt).toISOString()}"
}`;

  return (
    <DestructiveDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Revoke API Token"
      alertDescription="Are you sure you want to revoke this API token? Any workers that are using this token will no longer be assigned tasks and services will not be able to trigger runs."
      confirmationText={apiToken.name}
      confirmButtonText="Revoke Token"
      isLoading={isLoading}
      onConfirm={onSubmit}
      onCancel={onCancel}
    >
      <div className="mt-4">
        <Code language="json" value={tokenCode} noHeader />
      </div>
    </DestructiveDialog>
  );
}
