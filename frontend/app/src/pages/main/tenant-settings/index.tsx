import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { CreateInviteForm } from './components/create-invite-form';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, {
  APIToken,
  CreateAPITokenRequest,
  CreateTenantInviteRequest,
  TenantInvite,
  UpdateTenantInviteRequest,
  queries,
} from '@/lib/api';
import { Dialog } from '@/components/ui/dialog';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './components/invites-columns';
import { columns as membersColumns } from './components/members-columns';
import { columns as apiTokensColumns } from './components/api-tokens-columns';
import { UpdateInviteForm } from './components/update-invite-form';
import { DeleteInviteForm } from './components/delete-invite-form';
import { CreateTokenDialog } from './components/create-token-dialog';
import { RevokeTokenForm } from './components/revoke-token-form';

export default function TenantSettings() {
  const { tenant } = useOutletContext<TenantContextType>();

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Settings for {tenant.name}
        </h2>
        <Separator className="my-4" />
        <MembersList />
        <Separator className="my-4" />
        <InvitesList />
        <Separator className="my-4" />
        <TokensList />
      </div>
    </div>
  );
}

function MembersList() {
  const { tenant } = useOutletContext<TenantContextType>();

  const listMembersQuery = useQuery({
    ...queries.members.list(tenant.metadata.id),
  });

  return (
    <div>
      <h3 className="text-xl font-semibold leading-tight text-foreground">
        Members
      </h3>
      <Separator className="my-4" />
      <DataTable
        columns={membersColumns()}
        data={listMembersQuery.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
        isLoading={listMembersQuery.isLoading}
      />
    </div>
  );
}

function InvitesList() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showCreateInviteModal, setShowCreateInviteModal] = useState(false);
  const [updateInvite, setUpdateInvite] = useState<TenantInvite | null>(null);
  const [deleteInvite, setDeleteInvite] = useState<TenantInvite | null>(null);

  const listInvitesQuery = useQuery({
    ...queries.invites.list(tenant.metadata.id),
  });

  const cols = columns({
    onEditClick: (row) => {
      setUpdateInvite(row);
    },
    onDeleteClick: (row) => {
      setDeleteInvite(row);
    },
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Invites
        </h3>
        <Button
          key="create-invite"
          onClick={() => setShowCreateInviteModal(true)}
        >
          Create Invite
        </Button>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={listInvitesQuery.isLoading}
        columns={cols}
        data={listInvitesQuery.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {showCreateInviteModal && (
        <CreateInvite
          tenant={tenant.metadata.id}
          showCreateInviteModal={showCreateInviteModal}
          setShowCreateInviteModal={setShowCreateInviteModal}
          onSuccess={() => {
            setShowCreateInviteModal(false);
            listInvitesQuery.refetch();
          }}
        />
      )}
      {updateInvite && (
        <UpdateInvite
          tenant={tenant.metadata.id}
          tenantInvite={updateInvite}
          setShowTenantInvite={() => setUpdateInvite(null)}
          onSuccess={() => {
            setUpdateInvite(null);
            listInvitesQuery.refetch();
          }}
        />
      )}
      {deleteInvite && (
        <DeleteInvite
          tenant={tenant.metadata.id}
          tenantInvite={deleteInvite}
          setShowTenantInviteDelete={() => setDeleteInvite(null)}
          onSuccess={() => {
            setDeleteInvite(null);
            listInvitesQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function CreateInvite({
  tenant,
  showCreateInviteModal,
  setShowCreateInviteModal,
  onSuccess,
}: {
  tenant: string;
  showCreateInviteModal: boolean;
  setShowCreateInviteModal: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createMutation = useMutation({
    mutationKey: ['tenant-invite:create', tenant],
    mutationFn: async (data: CreateTenantInviteRequest) => {
      await api.tenantInviteCreate(tenant, data);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog
      open={showCreateInviteModal}
      onOpenChange={setShowCreateInviteModal}
    >
      <CreateInviteForm
        isLoading={createMutation.isPending}
        onSubmit={createMutation.mutate}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function UpdateInvite({
  tenant,
  tenantInvite,
  setShowTenantInvite,
  onSuccess,
}: {
  tenant: string;
  tenantInvite: TenantInvite;
  setShowTenantInvite: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const updateMutation = useMutation({
    mutationKey: ['tenant-invite:update', tenant, tenantInvite],
    mutationFn: async (data: UpdateTenantInviteRequest) => {
      await api.tenantInviteUpdate(tenant, tenantInvite.metadata.id, data);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!tenantInvite} onOpenChange={setShowTenantInvite}>
      <UpdateInviteForm
        invite={tenantInvite}
        isLoading={updateMutation.isPending}
        onSubmit={updateMutation.mutate}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteInvite({
  tenant,
  tenantInvite,
  setShowTenantInviteDelete,
  onSuccess,
}: {
  tenant: string;
  tenantInvite: TenantInvite;
  setShowTenantInviteDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['tenant-invite:delete', tenant, tenantInvite],
    mutationFn: async () => {
      await api.tenantInviteDelete(tenant, tenantInvite.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!tenantInvite} onOpenChange={setShowTenantInviteDelete}>
      <DeleteInviteForm
        invite={tenantInvite}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowTenantInviteDelete(false)}
      />
    </Dialog>
  );
}

function TokensList() {
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
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          API Tokens
        </h3>
        <Button key="create-api-token" onClick={() => setShowTokenDialog(true)}>
          Create API Token
        </Button>
      </div>
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
