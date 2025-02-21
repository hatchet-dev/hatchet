import { ColumnDef, Row } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import api, { TenantMember, queries } from '@/lib/api';
import { capitalize } from '@/lib/utils';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType, UserContextType } from '@/lib/outlet';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { useMutation } from '@tanstack/react-query';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { useState } from 'react';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';

export const columns = ({
  onChangePasswordClick,
}: {
  onChangePasswordClick: (row: TenantMember) => void;
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
      cell: ({ row }) => <div>{capitalize(row.getValue('role'))}</div>,
    },
    {
      accessorKey: 'joined',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Joined" />
      ),
      cell: ({ row }) => (
        <div>
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
    },
    {
      id: 'actions',
      cell: ({ row }) => (
        <MemberActions
          row={row}
          onChangePasswordClick={onChangePasswordClick}
        />
      ),
    },
  ];
};

function MemberActions({
  row,
  onChangePasswordClick,
}: {
  row: Row<TenantMember>;
  onChangePasswordClick: (row: TenantMember) => void;
}) {
  const meta = useApiMeta();
  const { user } = useOutletContext<UserContextType>();
  const { tenant } = useOutletContext<TenantContextType>();

  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isDeleteLoading, setIsDeleteLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const isCurrent = row.original.user.email !== user.email;

  const deleteUserMutation = useMutation({
    mutationKey: ['tenant-member:delete'],
    mutationFn: async (data: TenantMember) => {
      await api.tenantMemberDelete(tenant.metadata.id, data.metadata.id);
    },
    onMutate: () => {
      setIsDeleteLoading(true);
    },
    onSuccess: async () => {
      setIsDeleteLoading(false);
      await queryClient.invalidateQueries({
        queryKey: queries.members.list(tenant.metadata.id).queryKey,
      });
      setIsDeleteConfirmOpen(false);
    },
    onError: handleApiError,
  });

  const actions = [];

  if (user.hasPassword && !isCurrent && meta.data?.allowChangePassword) {
    actions.push({
      label: 'Change Password',
      onClick: () => onChangePasswordClick(row.original),
    });
  }

  if (isCurrent) {
    actions.push({
      label: 'Remove',
      onClick: async () => {
        setIsDeleteConfirmOpen(true);
      },
    });
  }

  if (actions.length === 0) {
    return <></>;
  }

  return (
    <>
      <ConfirmDialog
        isOpen={isDeleteConfirmOpen}
        title={'Remove member'}
        description={`Are you sure you want to remove ${row.original.user.name} <${row.original.user.email}> from the tenant?`}
        submitLabel={'Remove'}
        onSubmit={() => {
          deleteUserMutation.mutate(row.original);
        }}
        onCancel={function (): void {
          setIsDeleteConfirmOpen(false);
        }}
        isLoading={isDeleteLoading}
      />
      <DataTableRowActions row={row} actions={actions} />
    </>
  );
}
