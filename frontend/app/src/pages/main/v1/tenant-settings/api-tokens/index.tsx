import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, { APIToken, CreateAPITokenRequest, queries } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns as apiTokensColumns } from './components/api-tokens-columns';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';
import { Dialog } from '@/components/v1/ui/dialog';
import { useApiError } from '@/lib/hooks';

export default function APITokens() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const listTokensQuery = useQuery({
    ...queries.tokens.list(tenant.metadata.id),
  });

  const cols = apiTokensColumns({
    onRevokeClick: (row) => {
      setRevokeToken(row);
    },
  });

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
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
        <DataTable
          isLoading={listTokensQuery.isLoading}
          columns={cols}
          data={listTokensQuery.data?.rows || []}
          filters={[]}
          getRowId={(row) => row.metadata.id}
        />

        {showTokenDialog && (
          <CreateToken
            tenant={tenant.metadata.id}
            showTokenDialog={showTokenDialog}
            setShowTokenDialog={setShowTokenDialog}
            onSuccess={() => {
              listTokensQuery.refetch();
            }}
          />
        )}
        {revokeToken && (
          <RevokeToken
            tenant={tenant.metadata.id}
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
  tenant,
  showTokenDialog,
  setShowTokenDialog,
  onSuccess,
}: {
  tenant: string;
  onSuccess: () => void;
  showTokenDialog: boolean;
  setShowTokenDialog: (show: boolean) => void;
}) {
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenant],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenant, data);
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
  tenant,
  apiToken,
  setShowTokenRevoke,
  onSuccess,
}: {
  tenant: string;
  apiToken: APIToken;
  setShowTokenRevoke: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const revokeMutation = useMutation({
    mutationKey: ['api-token:revoke', tenant, apiToken],
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
