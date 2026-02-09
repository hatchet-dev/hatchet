import { ChangePasswordDialog } from './components/change-password-dialog';
import { CreateInviteForm } from './components/create-invite-form';
import { DeleteInviteForm } from './components/delete-invite-form';
import { InviteActions } from './components/invites-columns';
import { MemberActions } from './components/members-columns';
import { UpdateInviteForm } from './components/update-invite-form';
import { UpdateMemberForm } from './components/update-member-form';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Dialog } from '@/components/v1/ui/dialog';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  CreateTenantInviteRequest,
  TenantInvite,
  TenantMember,
  TenantMemberRole,
  UpdateTenantInviteRequest,
  UserChangePasswordRequest,
  queries,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { capitalize } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { AxiosError } from 'axios';
import { useState, useEffect, useMemo } from 'react';

export default function Members() {
  const { meta } = useApiMeta();

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Members and Invites
        </h2>
        <Separator className="my-4" />
        <MembersList />
        {meta?.allowInvites && (
          <>
            <Separator className="my-4" />
            <InvitesList />
          </>
        )}
      </div>
    </div>
  );
}

function MembersList() {
  const { tenantId } = useCurrentTenantId();
  const { getOrganizationIdForTenant, isCloudEnabled } = useOrganizations();
  const [showChangePasswordDialog, setShowChangePasswordDialog] =
    useState(false);
  const [memberToEdit, setMemberToEdit] = useState<TenantMember | null>(null);

  const listMembersQuery = useQuery({
    ...queries.members.list(tenantId),
  });

  const organizationId = getOrganizationIdForTenant(tenantId);

  // Get current user
  const { currentUser } = useCurrentUser();

  // Check if current user is admin
  const currentUserMember = useMemo(() => {
    return listMembersQuery.data?.rows?.find(
      (member) => member.user.email === currentUser?.email,
    );
  }, [listMembersQuery.data?.rows, currentUser?.email]);

  const isCurrentUserOwner = currentUserMember?.role === 'OWNER';

  // Separate owners and non-owners (only in cloud mode)
  const owners = useMemo(() => {
    if (!isCloudEnabled) {
      return [];
    }
    return (
      listMembersQuery.data?.rows?.filter(
        (member) => member.role === 'OWNER',
      ) || []
    );
  }, [listMembersQuery.data?.rows, isCloudEnabled]);

  const nonOwners = useMemo(() => {
    if (!isCloudEnabled) {
      // In OSS, show all members in the members table
      return listMembersQuery.data?.rows || [];
    }
    return (
      listMembersQuery.data?.rows?.filter(
        (member) => member.role !== 'OWNER',
      ) || []
    );
  }, [listMembersQuery.data?.rows, isCloudEnabled]);

  const ownersColumns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (member: TenantMember) => <div>{member.user.name}</div>,
      },
      {
        columnLabel: 'Email',
        cellRenderer: (member: TenantMember) => <div>{member.user.email}</div>,
      },
      {
        columnLabel: 'Joined',
        cellRenderer: (member: TenantMember) => (
          <RelativeDate date={member.metadata.createdAt} />
        ),
      },
    ],
    [],
  );

  const membersColumns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (member: TenantMember) => <div>{member.user.name}</div>,
      },
      {
        columnLabel: 'Email',
        cellRenderer: (member: TenantMember) => <div>{member.user.email}</div>,
      },
      {
        columnLabel: 'Role',
        cellRenderer: (member: TenantMember) => (
          <div className="font-medium">{capitalize(member.role)}</div>
        ),
      },
      {
        columnLabel: 'Joined',
        cellRenderer: (member: TenantMember) => (
          <RelativeDate date={member.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (member: TenantMember) => (
          <MemberActions
            member={member}
            onChangePasswordClick={() => {
              setShowChangePasswordDialog(true);
            }}
            onEditRoleClick={(member) => {
              setMemberToEdit(member);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div>
      {/* Owners Section - Only show in cloud mode */}
      {isCloudEnabled && (
        <>
          <div className="flex flex-row items-center justify-between">
            <h3 className="text-xl font-semibold leading-tight text-foreground">
              Owners
            </h3>
            {organizationId && isCurrentUserOwner && (
              <a
                href={`/organizations/${organizationId}`}
                className="text-sm text-primary hover:underline"
              >
                Manage in Organization →
              </a>
            )}
          </div>
          <Separator className="my-4" />
          {owners.length > 0 ? (
            <SimpleTable columns={ownersColumns} data={owners} />
          ) : (
            <div className="py-8 text-center text-sm text-muted-foreground">
              No owners found.
            </div>
          )}
          <Separator className="my-8" />
        </>
      )}

      {/* Members Section */}
      <h3 className="text-xl font-semibold leading-tight text-foreground">
        Members
      </h3>
      <Separator className="my-4" />
      {nonOwners.length > 0 ? (
        <SimpleTable columns={membersColumns} data={nonOwners} />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No members found.
        </div>
      )}
      {showChangePasswordDialog && (
        <ChangePassword
          showChangePasswordDialog={showChangePasswordDialog}
          setShowChangePasswordDialog={setShowChangePasswordDialog}
          onSuccess={() => {}}
        />
      )}
      {memberToEdit && (
        <UpdateMember
          member={memberToEdit}
          onClose={() => setMemberToEdit(null)}
          onSuccess={() => {
            setMemberToEdit(null);
            listMembersQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function UpdateMember({
  member,
  onClose,
  onSuccess,
}: {
  member: TenantMember;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { getOrganizationIdForTenant, isCloudEnabled } = useOrganizations();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });
  const navigate = useNavigate();
  // Check if this is a cloud tenant and if we're trying to modify an OWNER
  const isOwnerRole = member.role === 'OWNER';

  const organizationId = getOrganizationIdForTenant(tenantId);

  // If it's cloud-enabled and the member is an OWNER, redirect to org settings
  useEffect(() => {
    if (isCloudEnabled && isOwnerRole && organizationId) {
      navigate({
        to: appRoutes.organizationsRoute.to,
        params: { organization: organizationId },
      });
      onClose();
    }
  }, [isCloudEnabled, isOwnerRole, organizationId, onClose, navigate]);

  const updateMutation = useMutation({
    mutationKey: ['tenant-member:update', tenantId, member.metadata.id],
    mutationFn: async (data: { role: TenantMemberRole }) => {
      // Don't allow OWNER role changes in cloud mode
      if (isCloudEnabled && data.role === 'OWNER') {
        throw new Error(
          'OWNER role management must be done through Organization Settings',
        );
      }
      await api.tenantMemberUpdate(tenantId, member.metadata.id, data);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  // Don't render the dialog if we're redirecting
  if (isCloudEnabled && isOwnerRole) {
    return null;
  }

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <UpdateMemberForm
        isLoading={updateMutation.isPending}
        onSubmit={(data) => {
          // Prevent OWNER role assignment in cloud mode
          if (isCloudEnabled && data.role === 'OWNER') {
            return;
          }
          updateMutation.mutate(data);
        }}
        fieldErrors={fieldErrors}
        member={member}
        isCloudEnabled={isCloudEnabled}
      />
    </Dialog>
  );
}

function InvitesList() {
  const { tenantId } = useCurrentTenantId();
  const [showCreateInviteModal, setShowCreateInviteModal] = useState(false);
  const [updateInvite, setUpdateInvite] = useState<TenantInvite | null>(null);
  const [deleteInvite, setDeleteInvite] = useState<TenantInvite | null>(null);

  const listInvitesQuery = useQuery({
    ...queries.invites.list(tenantId),
  });

  const invitesColumns = useMemo(
    () => [
      {
        columnLabel: 'Email',
        cellRenderer: (invite: TenantInvite) => <div>{invite.email}</div>,
      },
      {
        columnLabel: 'Role',
        cellRenderer: (invite: TenantInvite) => (
          <div>{capitalize(invite.role)}</div>
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (invite: TenantInvite) => (
          <RelativeDate date={invite.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Expires',
        cellRenderer: (invite: TenantInvite) => (
          <RelativeDate date={invite.expires} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (invite: TenantInvite) => (
          <InviteActions
            invite={invite}
            onEditClick={(invite) => {
              setUpdateInvite(invite);
            }}
            onDeleteClick={(invite) => {
              setDeleteInvite(invite);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
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
      {(listInvitesQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={invitesColumns}
          data={listInvitesQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No invites found. Create an invite to add new members to your tenant.
        </div>
      )}
      {showCreateInviteModal && (
        <CreateInvite
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
  showCreateInviteModal,
  setShowCreateInviteModal,
  onSuccess,
}: {
  showCreateInviteModal: boolean;
  setShowCreateInviteModal: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { getOrganizationIdForTenant, isCloudEnabled } = useOrganizations();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const organizationId = getOrganizationIdForTenant(tenantId);

  const createMutation = useMutation({
    mutationKey: ['tenant-invite:create', tenantId],
    mutationFn: async (data: CreateTenantInviteRequest) => {
      await api.tenantInviteCreate(tenantId, data);
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
        isCloudEnabled={isCloudEnabled}
        organizationId={organizationId}
      />
    </Dialog>
  );
}

function UpdateInvite({
  tenantInvite,
  setShowTenantInvite,
  onSuccess,
}: {
  tenantInvite: TenantInvite;
  setShowTenantInvite: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const updateMutation = useMutation({
    mutationKey: ['tenant-invite:update', tenantId, tenantInvite],
    mutationFn: async (data: UpdateTenantInviteRequest) => {
      await api.tenantInviteUpdate(tenantId, tenantInvite.metadata.id, data);
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
  tenantInvite,
  setShowTenantInviteDelete,
  onSuccess,
}: {
  tenantInvite: TenantInvite;
  setShowTenantInviteDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['tenant-invite:delete', tenantId, tenantInvite],
    mutationFn: async () => {
      await api.tenantInviteDelete(tenantId, tenantInvite.metadata.id);
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

function ChangePassword({
  showChangePasswordDialog,
  setShowChangePasswordDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showChangePasswordDialog: boolean;
  setShowChangePasswordDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const updatePasswordMutation = useMutation({
    mutationKey: ['user:update', tenantId],
    mutationFn: async (data: UserChangePasswordRequest) => {
      const res = await api.userUpdatePassword(data);
      return res.data;
    },
    onMutate: () => {
      setFieldErrors({});
    },
    onSuccess: () => {
      onSuccess();
      setShowChangePasswordDialog(false);
    },
    onError: (e: AxiosError<unknown, any>) => {
      return handleApiError(e);
    },
  });

  return (
    <Dialog
      open={showChangePasswordDialog}
      onOpenChange={setShowChangePasswordDialog}
    >
      <ChangePasswordDialog
        isLoading={updatePasswordMutation.isPending}
        onSubmit={(data) =>
          updatePasswordMutation.mutate({
            password: data.password,
            newPassword: data.newPassword,
          })
        }
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}
