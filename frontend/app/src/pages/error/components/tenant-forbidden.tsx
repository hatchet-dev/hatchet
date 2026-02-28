import { ErrorPageLayout } from './layout';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { useCurrentUser } from '@/hooks/use-current-user';
import { useOrganizations } from '@/hooks/use-organizations';
import api, { TenantMember } from '@/lib/api';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { lastTenantAtom } from '@/lib/atoms';
import { getOptionalStringParam } from '@/lib/router-helpers';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import {
  BuildingOffice2Icon,
  ChevronDownIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from '@tanstack/react-router';
import { useSetAtom } from 'jotai';
import { LogOut, ShieldX, Undo2 } from 'lucide-react';
import { useMemo, useState } from 'react';

function TenantPickerItem({
  membership,
  onSelect,
  indented,
}: {
  membership: TenantMember;
  onSelect: () => void;
  indented?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground/80 transition-colors hover:bg-muted/50 hover:text-foreground"
      style={indented ? { paddingLeft: '2rem' } : undefined}
    >
      <div className="flex h-5 w-5 items-center justify-center">
        <div className="h-2 w-2 rounded-full bg-green-500" />
      </div>
      <span className="min-w-0 flex-1 truncate">{membership.tenant?.name}</span>
    </button>
  );
}

function OrgGroup({
  organization,
  tenants,
  onTenantSelect,
  defaultExpanded,
}: {
  organization: OrganizationForUser;
  tenants: TenantMember[];
  onTenantSelect: (membership: TenantMember) => void;
  defaultExpanded?: boolean;
}) {
  const [expanded, setExpanded] = useState(defaultExpanded ?? true);

  if (tenants.length === 0) {
    return null;
  }

  return (
    <div>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors hover:bg-muted/50"
      >
        {expanded ? (
          <ChevronDownIcon className="size-3 shrink-0" />
        ) : (
          <ChevronRightIcon className="size-3 shrink-0" />
        )}
        <BuildingOffice2Icon className="size-4 shrink-0" />
        <span className="min-w-0 flex-1 truncate">{organization.name}</span>
      </button>
      {expanded &&
        tenants
          .sort((a, b) =>
            (a.tenant?.name ?? '')
              .toLowerCase()
              .localeCompare((b.tenant?.name ?? '').toLowerCase()),
          )
          .map((membership) => (
            <TenantPickerItem
              key={membership.metadata.id}
              membership={membership}
              onSelect={() => onTenantSelect(membership)}
              indented
            />
          ))}
    </div>
  );
}

export function TenantForbidden() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const tenant = getOptionalStringParam(params, 'tenant');
  const setLastTenant = useSetAtom(lastTenantAtom);
  const queryClient = useQueryClient();

  const { currentUser } = useCurrentUser();
  const { tenantMemberships, isCloudEnabled } = useUserUniverse();
  const { organizations, getOrganizationForTenant, isTenantArchivedInOrg } =
    useOrganizations();

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate({ to: appRoutes.authLoginRoute.to, replace: true });
    },
  });

  const handleTenantSelect = (membership: TenantMember) => {
    if (!membership.tenant) {
      return;
    }
    setLastTenant(membership.tenant);
    queryClient.clear();
    navigate({
      to: appRoutes.tenantRunsRoute.to,
      params: { tenant: membership.tenant.metadata.id },
    });
  };

  // Filter out the current (forbidden) tenant and archived tenants
  const availableTenants = useMemo(
    () =>
      tenantMemberships?.filter((m) => {
        const id = m.tenant?.metadata.id;
        if (!id || id === tenant) {
          return false;
        }
        if (isCloudEnabled && isTenantArchivedInOrg(id)) {
          return false;
        }
        return true;
      }) || [],
    [tenantMemberships, tenant, isCloudEnabled, isTenantArchivedInOrg],
  );

  const { orgGroups, standaloneTenants } = useMemo(() => {
    if (!isCloudEnabled) {
      return { orgGroups: [], standaloneTenants: availableTenants };
    }

    const orgMap = new Map<string, TenantMember[]>();
    const standalone: TenantMember[] = [];

    availableTenants.forEach((membership) => {
      const tenantId = membership.tenant?.metadata.id || '';
      const org = getOrganizationForTenant(tenantId);
      if (org) {
        const orgId = org.metadata.id;
        if (!orgMap.has(orgId)) {
          orgMap.set(orgId, []);
        }
        orgMap.get(orgId)!.push(membership);
      } else {
        standalone.push(membership);
      }
    });

    const groups = organizations
      .map((org) => ({
        organization: org,
        tenants: orgMap.get(org.metadata.id) || [],
      }))
      .filter((g) => g.tenants.length > 0)
      .sort((a, b) =>
        a.organization.name
          .toLowerCase()
          .localeCompare(b.organization.name.toLowerCase()),
      );

    return { orgGroups: groups, standaloneTenants: standalone };
  }, [
    isCloudEnabled,
    availableTenants,
    organizations,
    getOrganizationForTenant,
  ]);

  return (
    <ErrorPageLayout
      icon={<ShieldX className="h-6 w-6" />}
      title="Access denied"
      description="You don't have permission to view this tenant."
      actions={
        <Button
          leftIcon={<Undo2 className="h-4 w-4" />}
          onClick={() => window.history.back()}
          variant="outline"
        >
          Go back
        </Button>
      }
    >
      <div className="flex justify-center">
        <Badge variant="secondary" className="font-mono">
          403
        </Badge>
      </div>

      <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-3 text-left font-mono text-xs text-muted-foreground">
        <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground/70">
          Requested Tenant
        </div>
        <div className="break-all text-foreground/90">
          {tenant || 'unknown'}
        </div>
      </div>

      {availableTenants.length > 0 && (
        <div className="mx-auto w-full max-w-prose rounded-md border bg-muted/20 p-1">
          <div className="px-3 pb-1 pt-2 text-[10px] uppercase tracking-wide text-muted-foreground/70">
            Switch to another tenant
          </div>
          <div className="max-h-64 overflow-y-auto">
            {orgGroups.map(({ organization, tenants }) => (
              <OrgGroup
                key={organization.metadata.id}
                organization={organization}
                tenants={tenants}
                onTenantSelect={handleTenantSelect}
              />
            ))}
            {standaloneTenants.length > 0 &&
              standaloneTenants
                .sort((a, b) =>
                  (a.tenant?.name ?? '')
                    .toLowerCase()
                    .localeCompare((b.tenant?.name ?? '').toLowerCase()),
                )
                .map((membership) => (
                  <TenantPickerItem
                    key={membership.metadata.id}
                    membership={membership}
                    onSelect={() => handleTenantSelect(membership)}
                  />
                ))}
          </div>
        </div>
      )}

      <div className="flex flex-row flex-wrap items-center justify-center gap-2">
        {!!currentUser?.email && (
          <div className="text-xs text-muted-foreground">
            Signed in as <span className="font-mono">{currentUser.email}</span>
          </div>
        )}
        <Button
          leftIcon={<LogOut className="size-4" />}
          onClick={() => logoutMutation.mutate()}
          size="sm"
          variant="ghost"
        >
          Sign out
        </Button>
      </div>
    </ErrorPageLayout>
  );
}
