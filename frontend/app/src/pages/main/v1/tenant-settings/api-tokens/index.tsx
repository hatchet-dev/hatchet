import { TokenActions } from './components/api-tokens-columns';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Dialog } from '@/components/v1/ui/dialog';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { APIToken, CreateAPITokenRequest, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState, useMemo } from 'react';

export default function APITokens() {
  const { tenantId } = useCurrentTenantId();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const listTokensQuery = useQuery({
    ...queries.tokens.list(tenantId),
  });

  const tokenColumns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (token: APIToken) => <div>{token.name}</div>,
      },
      {
        columnLabel: 'Created',
        cellRenderer: (token: APIToken) => (
          <RelativeDate date={token.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Expires',
        cellRenderer: (token: APIToken) => (
          <div>{new Date(token.expiresAt).toLocaleDateString()}</div>
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (token: APIToken) => (
          <TokenActions
            token={token}
            onRevokeClick={(token) => {
              setRevokeToken(token);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
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
        <p className="my-4 text-gray-700 dark:text-gray-300">
          API tokens are used by workers to connect with the Hatchet API and
          engine.
        </p>
        <Separator className="my-4" />
        {(listTokensQuery.data?.rows || []).length > 0 ? (
          <SimpleTable
            columns={tokenColumns}
            data={listTokensQuery.data?.rows || []}
          />
        ) : (
          <div className="py-8 text-center text-sm text-muted-foreground">
            No API tokens found. Create a token to allow workers to connect.
          </div>
        )}

        {showTokenDialog && (
          <CreateToken
            showTokenDialog={showTokenDialog}
            setShowTokenDialog={setShowTokenDialog}
            onSuccess={() => {
              listTokensQuery.refetch();
            }}
          />
        )}
        {revokeToken && (
          <RevokeToken
            apiToken={revokeToken}
            setShowTokenRevoke={() => setRevokeToken(null)}
            onSuccess={() => {
              setRevokeToken(null);
              listTokensQuery.refetch();
            }}
          />
        )}
      </div>
    </div>
  );
}

function CreateToken({
  showTokenDialog,
  setShowTokenDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showTokenDialog: boolean;
  setShowTokenDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
      <CreateTokenDialog
        isLoading={createTokenMutation.isPending}
        onSubmit={createTokenMutation.mutate}
        token={generatedToken}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function RevokeToken({
  apiToken,
  setShowTokenRevoke,
  onSuccess,
}: {
  apiToken: APIToken;
  setShowTokenRevoke: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const revokeMutation = useMutation({
    mutationKey: ['api-token:revoke', tenantId, apiToken],
    mutationFn: async () => {
      await api.apiTokenUpdateRevoke(apiToken.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!apiToken} onOpenChange={setShowTokenRevoke}>
      <RevokeTokenForm
        apiToken={apiToken}
        isLoading={revokeMutation.isPending}
        onSubmit={() => revokeMutation.mutate()}
        onCancel={() => setShowTokenRevoke(false)}
      />
    </Dialog>
  );
}
