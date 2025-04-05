import { useState } from 'react';
import useApiTokens, { ApiTokensProvider } from '@/hooks/use-api-tokens';
import { TokensTable } from '@/components/tokens/tokens-table';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { APIToken } from '@/lib/api';
import { Dialog } from '@/components/ui/dialog/dialog';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';

export default function ApiTokensPage() {
  return (
    <ApiTokensProvider>
      <ApiTokensContent />
    </ApiTokensProvider>
  );
}

function ApiTokensContent() {
  const { data, isLoading, create, revoke } = useApiTokens();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const handleRevoke = (token: APIToken) => {
    revoke.mutate(token, {
      onSuccess: () => {
        setRevokeToken(null);
      },
    });
  };

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            API Tokens
          </h2>

          <Button
            key="create-api-token"
            onClick={() => setShowTokenDialog(true)}
          >
            Create API Token
          </Button>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          API tokens are used by workers to connect with the Hatchet API and
          engine.
        </p>
        <Separator className="my-4" />

        <TokensTable
          data={data || []}
          isLoading={isLoading}
          onRevokeClick={(token) => setRevokeToken(token)}
        />

        {showTokenDialog && (
          <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
            <CreateTokenDialog close={setShowTokenDialog} />
          </Dialog>
        )}

        {revokeToken && (
          <RevokeTokenForm
            apiToken={revokeToken}
            isLoading={revoke.isPending}
            onSubmit={() => handleRevoke(revokeToken)}
            onCancel={() => setRevokeToken(null)}
            open={!!revokeToken}
            onOpenChange={(open) => {
              if (!open) {
                setRevokeToken(null);
              }
            }}
          />
        )}
      </div>
    </div>
  );
}
