import { ColumnDef, Row } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { TenantMember } from '@/lib/api';
import { capitalize } from '@/lib/utils';
import { DataTableRowActions } from '@/components/molecules/data-table/data-table-row-actions';
import { useOutletContext } from 'react-router-dom';
import { UserContextType } from '@/lib/outlet';
import RelativeDate from '@/components/molecules/relative-date';

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
  const { user } = useOutletContext<UserContextType>();
  const actions = [];

  if (user.hasPassword) {
    actions.push({
      label: 'Change Password',
      onClick: () => onChangePasswordClick(row.original),
    });
  }

  if (user.metadata.id === row.original.metadata.id) {
    return <></>;
  }

  return <DataTableRowActions row={row} actions={actions} />;
}
