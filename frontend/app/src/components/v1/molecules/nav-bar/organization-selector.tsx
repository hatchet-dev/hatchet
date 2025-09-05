import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import {
  BuildingOffice2Icon,
  Cog6ToothIcon,
  PlusIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
} from '@heroicons/react/24/outline';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import { Tenant, TenantMember, TenantVersion } from '@/lib/api';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { CaretSortIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useOrganizations } from '@/hooks/use-organizations';
import invariant from 'tiny-invariant';

interface OrganizationGroupProps {
  organization: OrganizationForUser;
  tenants: TenantMember[];
  currentTenant: Tenant;
  isExpanded: boolean;
  onToggleExpand: () => void;
  onTenantSelect: (tenant: Tenant) => void;
  onClose: () => void;
  onNavigate: (path: string) => void;
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
    onNavigate(`/organizations/${organization.metadata.id}`);
  };

  const handleNewTenantClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onClose();
    onNavigate(
      '/onboarding/create-tenant?organizationId=' + organization.metadata.id,
    );
  };

  return (
    <>
      <CommandItem
        onSelect={onToggleExpand}
        className="text-sm cursor-pointer hover:bg-accent focus:bg-accent"
      >
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-2">
            {isExpanded ? (
              <ChevronDownIcon className="h-3 w-3" />
            ) : (
              <ChevronRightIcon className="h-3 w-3" />
            )}
            <BuildingOffice2Icon className="h-4 w-4" />
            <span className="font-medium">{organization.name}</span>
          </div>
          {organization.isOwner && (
            <div className="flex items-center gap-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                    title="New Tenant"
                    onClick={handleNewTenantClick}
                  >
                    <PlusIcon className="h-3 w-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>New Tenant</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                    title="Settings"
                    onClick={handleSettingsClick}
                  >
                    <Cog6ToothIcon className="h-3 w-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Settings</TooltipContent>
              </Tooltip>
            </div>
          )}
        </div>
      </CommandItem>

      {isExpanded &&
        tenants.map((membership) => (
          <CommandItem
            key={membership.metadata.id}
            onSelect={() => {
              invariant(membership.tenant);
              onTenantSelect(membership.tenant);
              onClose();
            }}
            className="text-sm cursor-pointer pl-6 hover:bg-accent focus:bg-accent"
          >
            <div className="flex items-center justify-between w-full">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-green-500" />
                <span>{membership.tenant?.name}</span>
                <span className="text-xs text-muted-foreground">
                  {membership.tenant?.slug}
                </span>
              </div>
              <CheckIcon
                className={cn(
                  'h-4 w-4',
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
  const { tenant: currTenant, setTenant: setCurrTenant } = useTenantDetails();
  const [open, setOpen] = useState(false);
  const [expandedOrgs, setExpandedOrgs] = useState<string[]>([]);
  const { organizations, getOrganizationForTenant } = useOrganizations();

  const handleClose = () => setOpen(false);
  const handleNavigate = (path: string) => navigate(path, { replace: true });

  const handleTenantSelect = (tenant: Tenant) => {
    setCurrTenant(tenant);

    if (tenant.version === TenantVersion.V0) {
      // Hack to wait for next event loop tick so local storage is updated
      setTimeout(() => {
        window.location.href = `/workflow-runs?tenant=${tenant.metadata.id}`;
      }, 0);
    }
  };

  const toggleOrgExpansion = (orgId: string) => {
    setExpandedOrgs((prev) =>
      prev.includes(orgId)
        ? prev.filter((id) => id !== orgId)
        : [...prev, orgId],
    );
  };

  // Group memberships by organization
  const { currentOrgData, otherOrgsData, standaloneTenants } = useMemo(() => {
    const orgMap = new Map<string, TenantMember[]>();
    const standalone: TenantMember[] = [];

    memberships.forEach((membership) => {
      const org = getOrganizationForTenant(
        membership.tenant?.metadata.id || '',
      );
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

    const currentOrg = currTenant
      ? getOrganizationForTenant(currTenant.metadata.id)
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
      .filter((item) => item.tenants.length > 0);

    return {
      currentOrgData: currentOrg
        ? { organization: currentOrg, tenants: currentOrgTenants }
        : null,
      otherOrgsData: otherOrgs,
      standaloneTenants: standalone,
    };
  }, [memberships, organizations, getOrganizationForTenant, currTenant]);

  if (!currTenant) {
    return null;
  }

  // Auto-expand current organization
  const currentOrgId = currentOrgData?.organization.metadata.id;
  if (currentOrgId && !expandedOrgs.includes(currentOrgId)) {
    setExpandedOrgs((prev) => [...prev, currentOrgId]);
  }

  return (
    <TooltipProvider delayDuration={200}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            aria-label="Select a tenant"
            className={cn('w-full justify-between', className)}
          >
            <div className="flex items-center gap-2">
              <BuildingOffice2Icon className="h-4 w-4" />
              <span className="truncate">{currTenant.name}</span>
            </div>
            <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          side="right"
          className="w-[320px] p-0 mb-6 z-50 border border-border shadow-md rounded-md"
        >
          <Command className="border-0">
            <CommandList>
              <CommandEmpty>No tenants found.</CommandEmpty>

              {currentOrgData && (
                <CommandGroup heading="Current Organization">
                  <OrganizationGroup
                    organization={currentOrgData.organization}
                    tenants={currentOrgData.tenants}
                    currentTenant={currTenant}
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
                </CommandGroup>
              )}

              {otherOrgsData.length > 0 && (
                <CommandGroup heading="Other Organizations">
                  {otherOrgsData.map(({ organization, tenants }) => (
                    <OrganizationGroup
                      key={organization.metadata.id}
                      organization={organization}
                      tenants={tenants}
                      currentTenant={currTenant}
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

              {standaloneTenants.length > 0 && (
                <CommandGroup heading="Tenants">
                  {standaloneTenants.map((membership) => (
                    <CommandItem
                      key={membership.metadata.id}
                      onSelect={() => {
                        invariant(membership.tenant);
                        handleTenantSelect(membership.tenant);
                        handleClose();
                      }}
                      className="text-sm cursor-pointer"
                    >
                      <BuildingOffice2Icon className="mr-2 h-4 w-4" />
                      {membership.tenant?.name}
                      <CheckIcon
                        className={cn(
                          'ml-auto h-4 w-4',
                          currTenant.slug === membership.tenant?.slug
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </CommandItem>
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
