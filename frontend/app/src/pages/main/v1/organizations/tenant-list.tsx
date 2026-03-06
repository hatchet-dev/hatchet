import type { TenantWithRole } from './index';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import { TenantMemberRole } from '@/lib/api/generated/data-contracts';
import { globalEmitter } from '@/lib/global-emitter';
import { appRoutes } from '@/router';
import { PlusIcon } from '@heroicons/react/24/outline';
import { Link, useNavigate } from '@tanstack/react-router';

const makeTenantColumns = ({
  onViewTenant,
  onInviteMember,
}: {
  onViewTenant: (tenantId: string) => void;
  onInviteMember: (tenantId: string) => void;
}) => [
  {
    columnLabel: 'Name',
    cellRenderer: (tenant: TenantWithRole) => (
      <Link
        to={appRoutes.tenantRoute.to}
        params={{ tenant: tenant.metadata.id }}
        className="font-medium hover:underline"
      >
        {tenant.name}
      </Link>
    ),
  },
  {
    columnLabel: 'ID',
    cellRenderer: (tenant: TenantWithRole) => (
      <div className="flex items-center gap-2">
        <span className="font-mono text-sm">{tenant.metadata.id}</span>
        <CopyToClipboard text={tenant.metadata.id} />
      </div>
    ),
  },
  {
    columnLabel: 'Slug',
    cellRenderer: (tenant: TenantWithRole) => (
      <span className="text-muted-foreground">{tenant.slug}</span>
    ),
  },
  {
    columnLabel: 'Actions',
    cellRenderer: (tenant: TenantWithRole) => {
      const canManage =
        tenant.currentUsersRole === TenantMemberRole.OWNER ||
        tenant.currentUsersRole === TenantMemberRole.ADMIN;

      return (
        <div className="flex items-center gap-2">
          {canManage && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onInviteMember(tenant.metadata.id)}
              leftIcon={<PlusIcon className="size-4" />}
            >
              Invite new member
            </Button>
          )}
        </div>
      );
    },
  },
];

export { makeTenantColumns };

export const TenantList = ({ tenants }: { tenants: TenantWithRole[] }) => {
  const navigate = useNavigate();

  if (tenants.length === 0) {
    return (
      <div className="py-16 text-center">
        <h3 className="mb-2 text-lg font-medium">No Tenants</h3>
        <p className="text-muted-foreground">
          You are not a member of any tenants.
        </p>
      </div>
    );
  }

  const columns = makeTenantColumns({
    onViewTenant: (tenantId) =>
      navigate({
        to: appRoutes.tenantRoute.to,
        params: { tenant: tenantId },
      }),
    onInviteMember: (tenantId) =>
      globalEmitter.emit('create-tenant-invite', { tenantId }),
  });

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Tenants</h2>
      <SimpleTable data={tenants} columns={columns} />
    </div>
  );
};
