import { UpgradeGateDialog } from '@/components/v1/cloud/billing/upgrade-gate-dialog';
import { TenantRegionBadge } from '@/components/v1/molecules/nav-bar/tenant-region-badge';
import {
  SwitcherOrganization,
  TenantSwitcherPanel,
} from '@/components/v1/molecules/nav-bar/tenant-switcher-panel';
import { useRecentTenants } from '@/components/v1/molecules/nav-bar/use-recent-tenants';
import { useTenantSwitcherShortcut } from '@/components/v1/molecules/nav-bar/use-tenant-switcher-shortcut';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { Tenant, TenantMember } from '@/lib/api';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { BuildingOffice2Icon, PlusIcon } from '@heroicons/react/24/outline';
import { CaretSortIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import { useNavigate, useRouter } from '@tanstack/react-router';
import { useMemo, useRef, useState } from 'react';

interface OrganizationSelectorProps {
  className?: string;
  memberships: TenantMember[];
}

export function OrganizationSelector({
  className,
  memberships,
}: OrganizationSelectorProps) {
  const navigate = useNavigate();
  const router = useRouter();
  const { canCreateOrganization, invalidate } = useUserUniverse();
  const [upgradeOpen, setUpgradeOpen] = useState(false);
  const {
    setTenant: setCurrTenant,
    isUserUniverseLoaded: isTenantLoaded,
    tenant,
  } = useTenantDetails();
  const [open, setOpen] = useState(false);
  const {
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
    isUserUniverseLoaded: isOrganizationsLoaded,
  } = useOrganizations();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const recentTenantIds = useRecentTenants(tenant?.metadata.id);
  useTenantSwitcherShortcut(triggerRef, () => setOpen(true));

  // The current org first, then the others by name.
  const { switcherOrganizations, currentOrganizationId } = useMemo(() => {
    const tenantsByOrg = new Map<string, Tenant[]>();

    memberships.forEach(({ tenant: memberTenant }) => {
      if (!memberTenant || isTenantArchivedInOrg(memberTenant.metadata.id)) {
        return;
      }

      const org = getOrganizationForTenant(memberTenant.metadata.id);
      if (org) {
        const orgTenants = tenantsByOrg.get(org.metadata.id) ?? [];
        orgTenants.push(memberTenant);
        tenantsByOrg.set(org.metadata.id, orgTenants);
      }
    });

    const currentOrg = tenant
      ? getOrganizationForTenant(tenant.metadata.id)
      : undefined;

    const toSwitcherOrg = (
      org: (typeof organizations)[number],
    ): SwitcherOrganization => ({
      id: org.metadata.id,
      name: org.name,
      isOwner: org.isOwner,
      tenants: tenantsByOrg.get(org.metadata.id) ?? [],
    });

    const others = organizations
      .filter((org) => org.metadata.id !== currentOrg?.metadata.id)
      .sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()))
      .map(toSwitcherOrg);

    return {
      switcherOrganizations: currentOrg
        ? [toSwitcherOrg(currentOrg), ...others]
        : others,
      currentOrganizationId: currentOrg?.metadata.id,
    };
  }, [
    memberships,
    tenant,
    getOrganizationForTenant,
    organizations,
    isTenantArchivedInOrg,
  ]);

  const handleOrganizationSettings = (organizationId: string) => {
    setOpen(false);

    const firstTenant = [
      ...(switcherOrganizations.find((org) => org.id === organizationId)
        ?.tenants ?? []),
    ].sort((a, b) =>
      a.name.toLowerCase().localeCompare(b.name.toLowerCase()),
    )[0];
    if (firstTenant) {
      setCurrTenant(firstTenant, { navigate: false });
    }

    navigate({
      to: appRoutes.organizationsIndexRoute.to,
      params: { organization: organizationId },
    });
  };

  const handleOrganizationNewTenant = (organizationId: string) => {
    setOpen(false);
    globalEmitter.emit('create-new-tenant', {
      defaultOrganizationId: organizationId,
    });
  };

  const hasNoTenant = memberships.length === 0 && organizations.length > 0;
  const triggerDisabled =
    !isTenantLoaded ||
    !isOrganizationsLoaded ||
    (memberships.length === 0 && organizations.length === 0);

  const upgradeOrganization = organizations.find(
    (org) => org.metadata.id === currentOrganizationId,
  );

  const triggerLabel = hasNoTenant
    ? organizations.length === 1
      ? organizations[0].name
      : 'Select organization'
    : (tenant?.name ?? 'Loading tenant…');

  const footerActions = upgradeOrganization
    ? [
        {
          id: 'create-organization',
          label: 'Create organization',
          icon: <PlusIcon className="size-3.5" />,
          onSelect: () => {
            setOpen(false);
            if (canCreateOrganization) {
              navigate({ to: appRoutes.organizationsNewRoute.to });
            } else {
              setUpgradeOpen(true);
            }
          },
        },
      ]
    : [];

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            ref={triggerRef}
            variant="outline"
            size="sm"
            role="combobox"
            aria-expanded={open}
            aria-label="Select a tenant"
            className={cn(
              'w-full min-w-0 max-w-[320px] justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
              open && 'bg-muted/30',
              className,
            )}
            disabled={triggerDisabled}
          >
            <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
              <BuildingOffice2Icon className="size-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate">{triggerLabel}</span>
              {!hasNoTenant && <TenantRegionBadge region={tenant?.region} />}
            </div>
            {(!isTenantLoaded || !isOrganizationsLoaded) && !open ? (
              <Spinner className="mr-0" />
            ) : (
              <CaretSortIcon className="size-4 shrink-0 opacity-50" />
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          className="w-[420px] max-w-[calc(100vw-2rem)] overflow-hidden rounded-xl border border-border bg-popover p-0 text-popover-foreground shadow-lg"
        >
          <TenantSwitcherPanel
            organizations={switcherOrganizations}
            currentTenant={tenant}
            currentOrganizationId={currentOrganizationId}
            recentTenantIds={recentTenantIds}
            isLoading={!isOrganizationsLoaded}
            footerActions={footerActions}
            getTenantHref={(t) =>
              router.buildLocation({
                to: appRoutes.tenantRunsRoute.to,
                params: { tenant: t.metadata.id },
              }).href
            }
            onSelectTenant={(t) => {
              setCurrTenant(t);
              setOpen(false);
            }}
            onTenantSettings={(t) => {
              setOpen(false);
              setCurrTenant(t, { navigate: false });
              navigate({
                to: appRoutes.tenantSettingsOverviewRoute.to,
                params: { tenant: t.metadata.id },
              });
            }}
            onOrganizationNewTenant={handleOrganizationNewTenant}
            onOrganizationSettings={handleOrganizationSettings}
          />
        </PopoverContent>
      </Popover>
      {upgradeOrganization ? (
        <UpgradeGateDialog
          open={upgradeOpen}
          gate="organizations"
          organizationId={upgradeOrganization.metadata.id}
          organizationName={upgradeOrganization.name}
          onDismiss={() => {
            setUpgradeOpen(false);
            void invalidate();
          }}
        />
      ) : null}
    </>
  );
}
