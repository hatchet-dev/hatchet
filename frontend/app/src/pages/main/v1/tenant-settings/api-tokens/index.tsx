import { SettingsPageHeader } from '../components/settings-page-header';
import { TokenActions } from './components/api-tokens-columns';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { Dialog } from '@/components/v1/ui/dialog';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, {
  APIToken,
  CreateAPITokenRequest,
  queries,
  TenantMemberRole,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { useState, useMemo } from 'react';

export default function APITokens() {
  const { tenantId } = useCurrentTenantId();
  const { membership } = useTenantDetails();
  const canManageApiTokens =
    membership === TenantMemberRole.OWNER ||
    membership === TenantMemberRole.ADMIN;
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [revokeToken, setRevokeToken] = useState<APIToken | null>(null);

  const listTokensQuery = useQuery({
    ...queries.tokens.list(tenantId),
    enabled: canManageApiTokens,
  });

  if (!canManageApiTokens) {
    return (
      <div className="h-full w-full flex-grow">
        <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
          <SettingsPageHeader
            title="API token settings"
            description="Create and revoke API tokens used by workers and external systems to authenticate with this tenant."
          />
          <Alert variant="warn">
            <ExclamationTriangleIcon className="size-4" />
            <AlertTitle className="font-semibold">
              You do not have permission to manage API tokens.
            </AlertTitle>
            <AlertDescription>
              Only tenant owners and admins can view or manage API tokens.
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

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
        <SettingsPageHeader
          title="API token settings"
          description="Create and revoke API tokens used by workers and external systems to authenticate with this tenant."
        />

        <div className="mb-4 flex flex-row items-baseline justify-end">
          <Button
            key="create-api-token"
            onClick={() => setShowTokenDialog(true)}
          >
            Create API Token
          </Button>
        </div>
        {(listTokensQuery.data?.rows || []).length > 0 ? (
          <SimpleTable
            columns={tokenColumns}
            data={listTokensQuery.data?.rows || []}
            rowKey={(row) => row.metadata.id}
          />
        ) : (
          <EmptyState
            title="No API tokens found"
            description="API tokens authenticate your workers and applications with the Hatchet API."
          />
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
