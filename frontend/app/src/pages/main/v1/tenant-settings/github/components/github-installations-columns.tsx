import { ColumnDef } from '@tanstack/react-table';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/v1/ui/avatar';
import { Button } from '@/components/v1/ui/button';
import { GearIcon } from '@radix-ui/react-icons';
import { GithubAppInstallation } from '@/lib/api/generated/cloud/data-contracts';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { CheckCircleIcon, PlusCircleIcon } from '@heroicons/react/24/outline';

export const columns = (
  linkToTenant: (installationId: string) => void,
): ColumnDef<GithubAppInstallation>[] => {
  return [
    {
      accessorKey: 'repository',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Account name" />
      ),
      cell: ({ row }) => {
        return (
          <div className="flex flex-row gap-4 items-center">
            <Avatar className="w-6 h-6">
              <AvatarImage src={row.original.account_avatar_url} />
              <AvatarFallback />
            </Avatar>
            <div>{row.original.account_name}</div>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'link',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Link to tenant?" />
      ),
      cell: ({ row }) => {
        if (row.original.is_linked_to_tenant) {
          return (
            <Button
              variant="ghost"
              disabled
              className="flex flex-row gap-2 px-2"
            >
              <CheckCircleIcon className="h-4 w-4" />
              Linked
            </Button>
          );
        }
        return row.original.type == 'installation' ? (
          <Button
            variant="outline"
            className="flex flex-row gap-2 px-2"
            onClick={() => linkToTenant(row.original.metadata.id)}
          >
            <PlusCircleIcon className="h-4 w-4" />
            Link to tenant
          </Button>
        ) : (
          <a
            href={
              row.original.installation_settings_url +
              `&redirect_to=${encodeURIComponent(window.location.pathname)}`
            }
            target="_blank"
            rel="noreferrer"
          >
            <Button variant="outline" className="flex flex-row gap-2 px-2">
              Finish Setup
            </Button>
          </a>
        );
      },
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'settings',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Github Settings" />
      ),
      cell: ({ row }) => {
        return (
          <a
            href={
              row.original.installation_settings_url +
              `&redirect_to=${encodeURIComponent(window.location.pathname)}`
            }
            target="_blank"
            rel="noreferrer"
          >
            <Button variant="ghost" className="flex flex-row gap-2 px-2">
              <GearIcon className="h-4 w-4" />
              Configure
            </Button>
          </a>
        );
      },
      enableSorting: false,
      enableHiding: false,
    },
  ];
};
