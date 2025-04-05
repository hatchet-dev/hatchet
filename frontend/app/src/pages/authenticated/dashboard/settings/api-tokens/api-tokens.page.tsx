import { useState } from 'react';
import useApiTokens, { ApiTokensProvider } from '@/hooks/use-api-tokens';
import { TokensTable } from '@/components/tokens/tokens-table';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { APIToken } from '@/lib/api';
import { Dialog } from '@/components/ui/dialog/dialog';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';
import { DocsButton } from '@/components/ui/docs-button';
import useCan from '@/hooks/use-can';
import docs from '@/docs-meta-data';
import { apiTokens } from '@/lib/can/features/api-tokens';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Lock } from 'lucide-react';
export default function ApiTokensPage() {
  return (
    <ApiTokensProvider>
      <ApiTokensContent />
    </ApiTokensProvider>
  );
}

function ApiTokensContent() {
  const { data, isLoading } = useApiTokens();
  const { canWithReason } = useCan();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const CreateTokenButton = () => (
    <Button key="create-api-token" onClick={() => setShowTokenDialog(true)}>
      Create API Token
    </Button>
  );

  const { allowed: canManage, message: canManageMessage } = canWithReason(
    apiTokens.manage(),
  );

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            API Tokens
          </h2>

          {canManage && <CreateTokenButton />}
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          API tokens are used by workers to connect with the Hatchet API and
          engine.
        </p>
        {canManageMessage && (
          <Alert variant="warning">
            <Lock className="w-4 h-4 mr-2" />
            <AlertTitle>Role required</AlertTitle>
            <AlertDescription>{canManageMessage}</AlertDescription>
          </Alert>
        )}
        <Separator className="my-4" />

        <TokensTable
          data={data || []}
          isLoading={isLoading}
          onRevokeClick={(token) => setRevokeToken(token)}
          emptyState={
            <div className="flex flex-col items-center justify-center gap-4 py-8">
              <p className="text-md">No API tokens found.</p>
              <p className="text-sm text-muted-foreground">
                Create a new API token to get started.
              </p>
              {canManage && <CreateTokenButton />}
              <DocsButton doc={docs.home.setup} />
            </div>
          }
        />

        {showTokenDialog && (
          <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
            <CreateTokenDialog close={() => setShowTokenDialog(false)} />
          </Dialog>
        )}

        {revokeToken && (
          <RevokeTokenForm
            apiToken={revokeToken}
            close={() => setRevokeToken(null)}
          />
        )}
      </div>
    </div>
  );
}
