import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import api, { TenantMember, queries } from '@/lib/api';
import { capitalize } from '@/lib/utils';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useOutletContext } from 'react-router-dom';
import { UserContextType } from '@/lib/outlet';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { useMutation } from '@tanstack/react-query';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useState } from 'react';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useCurrentTenantId } from '@/hooks/use-tenant';

// Component for handling member actions
function MemberActions({
  member,
  onChangePasswordClick,
  onEditRoleClick,
}: {
  member: TenantMember;
  onChangePasswordClick: (member: TenantMember) => void;
  onEditRoleClick: (member: TenantMember) => void;
}) {
  const { user } = useOutletContext<UserContextType>();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const { handleApiError } = useApiError({});
  const { tenantId } = useCurrentTenantId();
  const meta = useApiMeta();

  const deleteMemberMutation = useMutation({
    mutationKey: ['tenant-member:delete', tenantId],
    mutationFn: async (data: { memberId: string }) => {
      await api.tenantMemberDelete(tenantId, data.memberId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.members.list(tenantId).queryKey,
      });
    },
    onError: handleApiError,
  });

  const canDeleteMember =
    member.user.email !== user?.email && meta.data?.allowInvites;

  const canChangePassword =
    member.user.email === user?.email && meta.data?.allowChangePassword;

  const canEditRole = member.user.email !== user?.email;

  return (
    <>
      <DataTableRowActions
        row={{ original: member } as any}
        actions={[
          ...(canEditRole
            ? [
                {
                  label: 'Edit role',
                  onClick: () => onEditRoleClick(member),
                },
              ]
            : []),
          ...(canChangePassword
            ? [
                {
                  label: 'Change password',
                  onClick: () => onChangePasswordClick(member),
                },
              ]
            : []),
          ...(canDeleteMember
            ? [
                {
                  label: 'Remove from tenant',
                  onClick: () => setShowDeleteDialog(true),
                },
              ]
            : []),
        ]}
      />
      <ConfirmDialog
        isOpen={showDeleteDialog}
        title="Remove member"
        description={`Are you sure you want to remove ${member.user.name} from this tenant?`}
        submitLabel="Remove"
        onSubmit={() => {
          deleteMemberMutation.mutate({
            memberId: member.metadata.id,
          });
        }}
        onCancel={() => setShowDeleteDialog(false)}
        isLoading={deleteMemberMutation.isPending}
      />
    </>
  );
}

export const columns = ({
  onChangePasswordClick,
  onEditRoleClick,
}: {
  onChangePasswordClick: (row: TenantMember) => void;
  onEditRoleClick: (row: TenantMember) => void;
}): ColumnDef<TenantMember>[] => {
  return [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => <div>{row.original.user.name}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'email',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Email" />
      ),
      cell: ({ row }) => <div>{row.original.user.email}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'role',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Role" />
      ),
      cell: ({ row }) => (
        <div className="font-medium">{capitalize(row.getValue('role'))}</div>
      ),
    },
    {
      accessorKey: 'joined',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Joined" />
      ),
      cell: ({ row }) => (
        <RelativeDate date={row.original.metadata.createdAt} />
      ),
    },
    {
      id: 'actions',
      cell: ({ row }) => (
        <MemberActions
          member={row.original}
          onChangePasswordClick={onChangePasswordClick}
          onEditRoleClick={onEditRoleClick}
        />
      ),
    },
  ];
};
