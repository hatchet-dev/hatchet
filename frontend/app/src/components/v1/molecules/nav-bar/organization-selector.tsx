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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { TooltipProvider } from '@/components/v1/ui/tooltip';
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
        value={`org-${organization.metadata.id}`}
        className="text-sm cursor-pointer hover:bg-accent focus:bg-accent"
      >
        <div className="flex items-start justify-between w-full">
          <div className="flex items-start gap-2 flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-shrink-0">
              {isExpanded ? (
                <ChevronDownIcon className="h-3 w-3" />
              ) : (
                <ChevronRightIcon className="h-3 w-3" />
              )}
              <BuildingOffice2Icon className="h-4 w-4" />
            </div>
            <span className="font-medium leading-tight break-words">
              {organization.name}
            </span>
          </div>
          {organization.isOwner && (
            <div className="flex items-center gap-1 flex-shrink-0 ml-2">
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                onClick={handleNewTenantClick}
                title="New Tenant"
              >
                <PlusIcon className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 hover:bg-accent-foreground/10"
                onClick={handleSettingsClick}
                title="Settings"
              >
                <Cog6ToothIcon className="h-3 w-3" />
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
              className="text-sm cursor-pointer pl-6 hover:bg-accent focus:bg-accent"
            >
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  <div className="w-5 h-5 flex items-center justify-center">
                    <div className="w-2 h-2 rounded-full bg-green-500" />
                  </div>
                  <span className="text-muted-foreground">
                    {membership.tenant?.name}
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
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [orgName, setOrgName] = useState('');
  const {
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
    handleCreateOrganization,
    createOrganizationLoading,
  } = useOrganizations();

  const handleClose = () => setOpen(false);
  const handleNavigate = (path: string) => {
    // Store the current path before navigating to org settings
    sessionStorage.setItem('orgSettingsPreviousPath', window.location.pathname);
    navigate(path, { replace: false });
  };

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

  const handleCreateOrgClick = () => {
    setOpen(false);
    setShowCreateModal(true);
  };

  const handleCreateOrgSubmit = () => {
    if (!orgName.trim()) {
      return;
    }

    handleCreateOrganization(orgName.trim(), (organizationId) => {
      setShowCreateModal(false);
      setOrgName('');
      navigate(`/organizations/${organizationId}`);
    });
  };

  const handleCreateOrgCancel = () => {
    setShowCreateModal(false);
    setOrgName('');
  };

  // Group memberships by organization
  const { currentOrgData, otherOrgsData, standaloneTenants } = useMemo(() => {
    const orgMap = new Map<string, TenantMember[]>();
    const standalone: TenantMember[] = [];

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
      standaloneTenants: standalone,
    };
  }, [
    memberships,
    organizations,
    getOrganizationForTenant,
    isTenantArchivedInOrg,
    currTenant,
  ]);

  if (!currTenant) {
    return null;
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
          side="top"
          align="start"
          sideOffset={20}
          className="w-[287px] p-0 z-50 border border-border shadow-md rounded-md"
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
                  <div className="px-2 py-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="w-full justify-center gap-2 h-8 text-sm hover:bg-accent"
                      onClick={handleCreateOrgClick}
                    >
                      <PlusIcon className="h-4 w-4" />
                      Create Organization
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
                      value={`standalone-tenant-${membership.tenant?.metadata.id}`}
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

      <Dialog open={showCreateModal} onOpenChange={setShowCreateModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create New Organization</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label htmlFor="org-name" className="text-sm font-medium">
                Organization Name
              </label>
              <Input
                id="org-name"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                placeholder="Enter organization name"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleCreateOrgSubmit();
                  }
                }}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={handleCreateOrgCancel}
              disabled={createOrganizationLoading}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateOrgSubmit}
              disabled={!orgName.trim() || createOrganizationLoading}
            >
              {createOrganizationLoading ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}
