import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/v1/ui/avatar';
import { Button } from '@/components/v1/ui/button';
import { GithubAppInstallation } from '@/lib/api/generated/cloud/data-contracts';
import { CheckCircleIcon, PlusCircleIcon } from '@heroicons/react/24/outline';
import { GearIcon } from '@radix-ui/react-icons';
import { ColumnDef } from '@tanstack/react-table';

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
          <div className="flex flex-row items-center gap-4">
            <Avatar className="h-6 w-6">
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
              leftIcon={<CheckCircleIcon className="size-4" />}
            >
              Linked
            </Button>
          );
        }
        return row.original.type == 'installation' ? (
          <Button
            variant="outline"
            onClick={() => linkToTenant(row.original.metadata.id)}
            leftIcon={<PlusCircleIcon className="size-4" />}
          >
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
            <Button variant="outline">Finish Setup</Button>
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
            <Button variant="ghost" leftIcon={<GearIcon className="size-4" />}>
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
