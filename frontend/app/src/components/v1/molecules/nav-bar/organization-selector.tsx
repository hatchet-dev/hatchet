import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import { TooltipProvider } from '@/components/v1/ui/tooltip';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { Tenant, TenantMember } from '@/lib/api';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import {
  BuildingOffice2Icon,
  Cog6ToothIcon,
  PlusIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline';
import { CaretSortIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import { useLocation, useNavigate, Link } from '@tanstack/react-router';
import { useState, useMemo } from 'react';
import invariant from 'tiny-invariant';

interface OrganizationGroupProps {
  organization: OrganizationForUser;
  tenants: TenantMember[];
  currentTenant?: Tenant;
  isExpanded: boolean;
  onToggleExpand: () => void;
  onTenantSelect: (tenant: Tenant) => void;
  onClose: () => void;
  onNavigate: (nav: { to: string; params?: Record<string, string> }) => void;
}

function OrganizationGroup({
  organization,
  tenants,
  currentTenant,
  isExpanded,
  onToggleExpand,
  onTenantSelect,
  onClose,
  onNavigate,
}: OrganizationGroupProps) {
  const handleSettingsClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onClose();
    onNavigate({
      to: appRoutes.organizationsRoute.to,
      params: {
        organization: organization.metadata.id,
      },
    });
  };

  const handleNewTenantClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onClose();
    globalEmitter.emit('new-tenant', {
      defaultOrganizationId: organization.metadata.id,
    });
  };

  return (
    <>
      <CommandItem
        onSelect={onToggleExpand}
        value={`org-${organization.metadata.id}`}
        className="cursor-pointer text-sm hover:bg-accent focus:bg-accent"
      >
        <div className="flex w-full items-start justify-between">
          <div className="flex min-w-0 flex-1 items-start gap-2">
            <div className="flex flex-shrink-0 items-center gap-2">
              {isExpanded ? (
                <ChevronDownIcon className="size-3" />
              ) : (
                <ChevronRightIcon className="size-3" />
              )}
              <BuildingOffice2Icon className="size-4" />
            </div>
            <span className="break-words font-medium leading-tight">
              {organization.name}
            </span>
          </div>
          {organization.isOwner && (
            <div className="ml-2 flex flex-shrink-0 items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                onClick={handleNewTenantClick}
                title="New Tenant"
              >
                <PlusIcon className="size-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                onClick={handleSettingsClick}
                title="Settings"
              >
                <Cog6ToothIcon className="size-3" />
              </Button>
            </div>
          )}
        </div>
      </CommandItem>

      {isExpanded &&
        tenants
          .sort(
            (a, b) =>
              a.tenant?.name
                ?.toLowerCase()
                .localeCompare(b.tenant?.name?.toLowerCase() ?? '') ?? 0,
          )
          .map((membership) => (
            <CommandItem
              key={membership.metadata.id}
              value={`tenant-${membership.tenant?.metadata.id}`}
              onSelect={() => {
                invariant(membership.tenant);
                onTenantSelect(membership.tenant);
                onClose();
              }}
              className="cursor-pointer pl-6 text-sm hover:bg-accent focus:bg-accent"
            >
              <div className="flex w-full items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="flex h-5 w-5 items-center justify-center">
                    <div className="h-2 w-2 rounded-full bg-green-500" />
                  </div>
                  <span className="text-muted-foreground">
                    {membership.tenant?.name}
                  </span>
                </div>
                <CheckIcon
                  className={cn(
                    'size-4',
                    currentTenant?.slug === membership.tenant?.slug
                      ? 'opacity-100'
                      : 'opacity-0',
                  )}
                />
              </div>
            </CommandItem>
          ))}
    </>
  );
}

interface OrganizationSelectorProps {
  className?: string;
  memberships: TenantMember[];
}

export function OrganizationSelector({
  className,
  memberships,
}: OrganizationSelectorProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    setTenant: setCurrTenant,
    isUserUniverseLoaded: isTenantLoaded,
    tenant,
  } = useTenantDetails();
  const [open, setOpen] = useState(false);
  const [expandedOrgs, setExpandedOrgs] = useState<string[]>([]);
  const {
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
    isUserUniverseLoaded: isOrganizationsLoaded,
  } = useOrganizations();

  const handleClose = () => setOpen(false);
  const handleNavigate = (nav: {
    to: string;
    params?: Record<string, string>;
  }) => {
    if (!nav.to) {
      return;
    }

    // Store the current path before navigating to org settings
    sessionStorage.setItem('orgSettingsPreviousPath', location.pathname);
    navigate({ to: nav.to, params: nav.params, replace: false });
  };

  const handleTenantSelect = (tenant: Tenant) => {
    setCurrTenant(tenant);
  };

  const toggleOrgExpansion = (orgId: string) => {
    setExpandedOrgs((prev) =>
      prev.includes(orgId)
        ? prev.filter((id) => id !== orgId)
        : [...prev, orgId],
    );
  };

  // Group memberships by organization
  const { currentOrgData, otherOrgsData } = useMemo(() => {
    const orgMap = new Map<string, TenantMember[]>();

    memberships.forEach((membership) => {
      const tenantId = membership.tenant?.metadata.id || '';
      if (isTenantArchivedInOrg(tenantId)) {
        return;
      }

      const org = getOrganizationForTenant(tenantId);
      if (org) {
        const orgId = org.metadata.id;
        if (!orgMap.has(orgId)) {
          orgMap.set(orgId, []);
        }
        orgMap.get(orgId)!.push(membership);
      }
    });

    const currentOrg = tenant
      ? getOrganizationForTenant(tenant.metadata.id)
      : null;
    const currentOrgTenants = currentOrg
      ? orgMap.get(currentOrg.metadata.id) || []
      : [];

    const otherOrgs = organizations
      .filter((org) => org.metadata.id !== currentOrg?.metadata.id)
      .map((org) => ({
        organization: org,
        tenants: orgMap.get(org.metadata.id) || [],
      }))
      .sort((a, b) =>
        a.organization.name
          .toLowerCase()
          .localeCompare(b.organization.name.toLowerCase()),
      );

    return {
      currentOrgData: currentOrg
        ? { organization: currentOrg, tenants: currentOrgTenants }
        : null,
      otherOrgsData: otherOrgs,
    };
  }, [
    memberships,
    tenant,
    getOrganizationForTenant,
    organizations,
    isTenantArchivedInOrg,
  ]);

  const triggerDisabled =
    !isTenantLoaded || !isOrganizationsLoaded || memberships.length === 0;

  return (
    <TooltipProvider delayDuration={200}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            role="combobox"
            aria-expanded={open}
            aria-label="Select a tenant"
            className={cn(
              'w-full min-w-0 justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
              open && 'bg-muted/30',
              className,
            )}
            disabled={triggerDisabled}
          >
            <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
              <BuildingOffice2Icon className="size-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate">
                {tenant?.name ?? 'Loading tenant…'}
              </span>
            </div>
            {(!isTenantLoaded || !isOrganizationsLoaded) && !open ? (
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground/70" />
            ) : (
              <CaretSortIcon className="size-4 shrink-0 opacity-50" />
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          className="w-[287px] rounded-md border border-border p-0 shadow-md"
        >
          <Command className="border-0">
            <CommandList>
              <CommandEmpty>No tenants found.</CommandEmpty>

              {!isOrganizationsLoaded && (
                <div className="flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground/70" />
                  Loading organizations…
                </div>
              )}

              {currentOrgData && (
                <CommandGroup heading="Current Organization">
                  <OrganizationGroup
                    organization={currentOrgData.organization}
                    tenants={currentOrgData.tenants}
                    currentTenant={tenant}
                    isExpanded={expandedOrgs.includes(
                      currentOrgData.organization.metadata.id,
                    )}
                    onToggleExpand={() =>
                      toggleOrgExpansion(
                        currentOrgData.organization.metadata.id,
                      )
                    }
                    onTenantSelect={handleTenantSelect}
                    onClose={handleClose}
                    onNavigate={handleNavigate}
                  />
                  <div className="px-2 py-1">
                    <Button
                      variant="outline"
                      size="sm"
                      fullWidth
                      leftIcon={<PlusIcon className="size-4" />}
                      asChild
                    >
                      <Link
                        to={appRoutes.organizationsNewRoute.to}
                        onClick={() => setOpen(false)}
                      >
                        Create Organization
                      </Link>
                    </Button>
                  </div>
                </CommandGroup>
              )}

              {otherOrgsData.length > 0 && (
                <CommandGroup heading="Other Organizations">
                  {otherOrgsData.map(({ organization, tenants }) => (
                    <OrganizationGroup
                      key={organization.metadata.id}
                      organization={organization}
                      tenants={tenants}
                      currentTenant={tenant}
                      isExpanded={expandedOrgs.includes(
                        organization.metadata.id,
                      )}
                      onToggleExpand={() =>
                        toggleOrgExpansion(organization.metadata.id)
                      }
                      onTenantSelect={handleTenantSelect}
                      onClose={handleClose}
                      onNavigate={handleNavigate}
                    />
                  ))}
                </CommandGroup>
              )}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  );
}
