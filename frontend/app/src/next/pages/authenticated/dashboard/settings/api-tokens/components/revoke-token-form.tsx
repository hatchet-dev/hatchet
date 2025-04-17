import { APIToken } from '@/next/lib/api';
import { DestructiveDialog } from '@/next/components/ui/dialog/destructive-dialog';
import { Code } from '@/next/components/ui/code';
import useApiTokens from '@/next/hooks/use-api-tokens';

interface RevokeTokenFormProps {
  apiToken: APIToken;
  close: () => void;
}

export function RevokeTokenForm({ apiToken, close }: RevokeTokenFormProps) {
  const { revoke } = useApiTokens();

  // Create code representation of the token
  const tokenCode = `{
  "name": "${apiToken.name}",
  "createdAt": "${new Date(apiToken.metadata.createdAt).toISOString()}",
  "expiresAt": "${new Date(apiToken.expiresAt).toISOString()}"
}`;

  return (
    <DestructiveDialog
      open={true}
      onOpenChange={close}
      title="Revoke API Token"
      alertDescription="Are you sure you want to revoke this API token? Any workers that are using this token will no longer be assigned tasks and services will not be able to trigger runs."
      confirmationText={apiToken.name}
      confirmButtonText="Revoke Token"
      isLoading={revoke.isPending}
      onConfirm={async () => {
        await revoke.mutateAsync(apiToken);
        close();
      }}
      onCancel={close}
    >
      <div className="mt-4">
        <Code language="json" value={tokenCode} noHeader />
      </div>
    </DestructiveDialog>
  );
}
