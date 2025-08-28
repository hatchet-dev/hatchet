import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import {
  BuildingOffice2Icon,
  CheckIcon,
  Cog6ToothIcon,
} from '@heroicons/react/24/outline';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/v1/ui/command';
import { TenantMember, TenantVersion } from '@/lib/api';
import invariant from 'tiny-invariant';
import { CaretSortIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import React, { useState, useMemo, useCallback } from 'react';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useQuery } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { Organization } from '@/lib/api/generated/cloud/data-contracts';

interface OrganizationSelectorProps {
  className?: string;
  memberships: TenantMember[];
}

export function OrganizationSelector({
  className,
  memberships,
}: OrganizationSelectorProps) {
  const cloudMeta = useCloudApiMeta();
  const { setTenant: setCurrTenant, tenant: currTenant } = useTenantDetails();
  const [open, setOpen] = useState(false);
  const [expandedOrgs, setExpandedOrgs] = useState<Set<string>>(new Set());

  const organizationListQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await cloudApi.organizationList();
      return result.data;
    },
    enabled: !!cloudMeta?.data,
  });

  const organizations = useMemo(
    () => organizationListQuery.data?.rows || [],
    [organizationListQuery.data?.rows],
  );

  const isCloudEnabled = useMemo(() => {
    return cloudMeta?.data && organizationListQuery.isSuccess;
  }, [cloudMeta, organizationListQuery.isSuccess]);

  const getOrganizationForTenant = useCallback(
    (tenantId: string) => {
      return organizations.find((org) =>
        org.tenants?.some((tenant) => tenant.id === tenantId),
      );
    },
    [organizations],
  );

  const currentOrganization = useMemo(() => {
    if (!currTenant) {
      return null;
    }
    return getOrganizationForTenant(currTenant.metadata.id);
  }, [currTenant, getOrganizationForTenant]);

  const organizedMemberships = useMemo(() => {
    if (!isCloudEnabled) {
      return { ungrouped: memberships };
    }

    const grouped: Record<
      string,
      { org: Organization; tenants: TenantMember[] }
    > = {};
    const ungrouped: TenantMember[] = [];

    memberships.forEach((membership) => {
      if (!membership.tenant) {
        return;
      }

      const org = getOrganizationForTenant(membership.tenant.metadata.id);
      if (org) {
        if (!grouped[org.metadata.id]) {
          grouped[org.metadata.id] = { org, tenants: [] };
        }
        grouped[org.metadata.id].tenants.push(membership);
      } else {
        ungrouped.push(membership);
      }
    });

    return { grouped: Object.values(grouped), ungrouped };
  }, [isCloudEnabled, memberships, getOrganizationForTenant]);

  if (!currTenant) {
    return <Spinner />;
  }

  if (!isCloudEnabled) {
    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            aria-label="Select a tenant"
            className={cn('w-full justify-between', className)}
          >
            <span className="truncate">{currTenant.name}</span>
            <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          side="top"
          align="start"
          className="w-full p-0 z-50 border border-border shadow-md mb-6 rounded-md"
        >
          <Command className="min-w-[280px] border-0">
            <CommandList>
              <CommandEmpty>No tenants found.</CommandEmpty>
              {memberships.map((membership) => (
                <CommandItem
                  key={membership.metadata.id}
                  onSelect={() => {
                    invariant(membership.tenant);
                    setCurrTenant(membership.tenant);
                    setOpen(false);
                    if (membership.tenant.version === TenantVersion.V0) {
                      setTimeout(() => {
                        window.location.href = `/workflow-runs?tenant=${membership.tenant?.metadata.id}`;
                      }, 0);
                    }
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
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a tenant"
          className={cn('w-full justify-between', className)}
        >
          <span className="truncate">{currTenant.name}</span>
          <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        side="top"
        align="start"
        className="w-full p-0 z-50 border border-border shadow-md mb-6 rounded-md"
      >
        <Command className="min-w-[280px] border-0">
          <CommandList>
            <CommandEmpty>No tenants found.</CommandEmpty>

            {/* Current Organization Header with Settings */}
            {currentOrganization && (
              <>
                <div className="px-2 py-2 border-b">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <BuildingOffice2Icon className="h-4 w-4" />
                      <span className="font-medium">
                        {currentOrganization.name}
                      </span>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      onClick={(e) => {
                        e.preventDefault();
                        setOpen(false);
                        // Navigate to org settings
                        window.location.href = `/organization/${currentOrganization.metadata.id}/settings`;
                      }}
                    >
                      <Cog6ToothIcon className="h-3 w-3" />
                    </Button>
                  </div>
                </div>

                {/* Current Organization Tenants */}
                {organizedMemberships.grouped?.map(({ org, tenants }) =>
                  org.metadata.id === currentOrganization.metadata.id ? (
                    <CommandGroup key={org.metadata.id} heading="Tenants">
                      {tenants.map((membership) => (
                        <CommandItem
                          key={membership.metadata.id}
                          onSelect={() => {
                            invariant(membership.tenant);
                            setCurrTenant(membership.tenant);
                            setOpen(false);
                            if (
                              membership.tenant.version === TenantVersion.V0
                            ) {
                              setTimeout(() => {
                                window.location.href = `/workflow-runs?tenant=${membership.tenant?.metadata.id}`;
                              }, 0);
                            }
                          }}
                          className="text-sm cursor-pointer"
                        >
                          <div className="w-2 h-2 rounded-full bg-primary mr-3" />
                          <div className="flex-1">
                            <div className="font-medium">
                              {membership.tenant?.name}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {membership.tenant?.slug}
                            </div>
                          </div>
                          <CheckIcon
                            className={cn(
                              'ml-auto h-4 w-4',
                              currTenant.slug === membership.tenant?.slug
                                ? 'opacity-100 text-primary'
                                : 'opacity-0',
                            )}
                          />
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  ) : null,
                )}
              </>
            )}

            {/* Other Organizations */}
            {organizedMemberships.grouped &&
              organizedMemberships.grouped.length > 1 && (
                <>
                  <CommandSeparator />
                  <CommandGroup heading="Switch Organizations">
                    {organizedMemberships.grouped?.map(({ org, tenants }) =>
                      org.metadata.id !== currentOrganization?.metadata.id ? (
                        <React.Fragment key={org.metadata.id}>
                          <CommandItem
                            onSelect={() => {
                              if (tenants.length === 1) {
                                // If only one tenant, switch directly
                                const firstTenant = tenants[0]?.tenant;
                                if (firstTenant) {
                                  setCurrTenant(firstTenant);
                                  setOpen(false);
                                  if (
                                    firstTenant.version === TenantVersion.V0
                                  ) {
                                    setTimeout(() => {
                                      window.location.href = `/workflow-runs?tenant=${firstTenant.metadata.id}`;
                                    }, 0);
                                  }
                                }
                              } else {
                                // If multiple tenants, toggle expansion
                                setExpandedOrgs((prev) => {
                                  const newSet = new Set(prev);
                                  if (newSet.has(org.metadata.id)) {
                                    newSet.delete(org.metadata.id);
                                  } else {
                                    newSet.add(org.metadata.id);
                                  }
                                  return newSet;
                                });
                              }
                            }}
                            className="text-sm cursor-pointer"
                          >
                            <BuildingOffice2Icon className="mr-3 h-4 w-4" />
                            <div className="flex-1">
                              <div className="font-medium">{org.name}</div>
                              <div className="text-xs text-muted-foreground">
                                {tenants.length} tenant
                                {tenants.length !== 1 ? 's' : ''}
                              </div>
                            </div>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0 ml-auto"
                              onClick={(e) => {
                                e.stopPropagation();
                                setOpen(false);
                                window.location.href = `/organization/${org.metadata.id}/settings`;
                              }}
                            >
                              <Cog6ToothIcon className="h-3 w-3 opacity-50" />
                            </Button>
                          </CommandItem>

                          {/* Show tenant list when expanded */}
                          {expandedOrgs.has(org.metadata.id) &&
                            tenants.length > 1 && (
                              <div className="ml-6 mb-2">
                                {tenants.map((membership) => (
                                  <CommandItem
                                    key={membership.metadata.id}
                                    onSelect={() => {
                                      invariant(membership.tenant);
                                      setCurrTenant(membership.tenant);
                                      setOpen(false);
                                      if (
                                        membership.tenant.version ===
                                        TenantVersion.V0
                                      ) {
                                        setTimeout(() => {
                                          window.location.href = `/workflow-runs?tenant=${membership.tenant?.metadata.id}`;
                                        }, 0);
                                      }
                                    }}
                                    className="text-sm cursor-pointer py-1"
                                  >
                                    <div className="w-2 h-2 rounded-full bg-muted mr-3" />
                                    <div className="flex-1">
                                      <div className="font-medium text-sm">
                                        {membership.tenant?.name}
                                      </div>
                                      <div className="text-xs text-muted-foreground">
                                        {membership.tenant?.slug}
                                      </div>
                                    </div>
                                  </CommandItem>
                                ))}
                              </div>
                            )}
                        </React.Fragment>
                      ) : null,
                    )}
                  </CommandGroup>
                </>
              )}

            {organizedMemberships.ungrouped.length > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup heading="Individual Tenants">
                  {organizedMemberships.ungrouped.map((membership) => (
                    <CommandItem
                      key={membership.metadata.id}
                      onSelect={() => {
                        invariant(membership.tenant);
                        setCurrTenant(membership.tenant);
                        setOpen(false);
                        if (membership.tenant.version === TenantVersion.V0) {
                          setTimeout(() => {
                            window.location.href = `/workflow-runs?tenant=${membership.tenant?.metadata.id}`;
                          }, 0);
                        }
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
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
