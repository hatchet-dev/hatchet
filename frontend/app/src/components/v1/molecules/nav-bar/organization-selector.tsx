import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { Cog6ToothIcon, PlusIcon } from '@heroicons/react/24/outline';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import { TenantMember } from '@/lib/api';
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
import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useTenantDetails } from '@/hooks/use-tenant';
import { useQuery } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';

interface OrganizationSelectorProps {
  className?: string;
  memberships: TenantMember[];
}

export function OrganizationSelector({ className }: OrganizationSelectorProps) {
  const navigate = useNavigate();
  const cloudMeta = useCloudApiMeta();
  const { tenant: currTenant } = useTenantDetails();
  const [open, setOpen] = useState(false);

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

  const { currentOrganization, otherOrganizations } = useMemo(() => {
    if (!currTenant) {
      return { currentOrganization: null, otherOrganizations: organizations };
    }

    const current = getOrganizationForTenant(currTenant.metadata.id);

    if (current) {
      // Current tenant belongs to an organization - separate current from others
      const others = organizations.filter(
        (org) => org.metadata.id !== current.metadata.id,
      );
      return { currentOrganization: current, otherOrganizations: others };
    } else {
      // Current tenant doesn't belong to any organization - show all as "Organizations"
      return { currentOrganization: null, otherOrganizations: organizations };
    }
  }, [currTenant, getOrganizationForTenant, organizations]);

  if (!currTenant || !isCloudEnabled || organizations.length === 0) {
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
            aria-label="Select an organization"
            className={cn('justify-between', className)}
          >
            <div className="flex items-center gap-2">
              <span className="truncate">
                {currentOrganization?.name || 'No Organization'}
              </span>
            </div>
            <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          side="bottom"
          align="end"
          className="w-[280px] p-0 z-50 border border-border shadow-md rounded-md"
        >
          <Command className="border-0">
            <CommandList>
              <CommandEmpty>No organizations found.</CommandEmpty>

              {currentOrganization && (
                <CommandGroup heading="Current Organization">
                  <CommandItem
                    key={currentOrganization.metadata.id}
                    className="text-sm hover:bg-transparent focus:bg-transparent data-[selected]:bg-transparent aria-selected:bg-transparent [&[aria-selected=true]]:bg-transparent"
                  >
                    <div className="flex items-center justify-between w-full">
                      <span className="flex-1 font-medium">
                        {currentOrganization.name}
                      </span>
                      <div className="flex items-center gap-1 ml-2">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0 hover:bg-accent"
                              title="Settings"
                              onClick={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                setOpen(false);
                                navigate(
                                  `/organizations/${currentOrganization.metadata.id}`,
                                  {
                                    replace: true,
                                  },
                                );
                              }}
                            >
                              <Cog6ToothIcon className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Settings</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0 hover:bg-accent"
                              title="New tenant"
                              onClick={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                setOpen(false);
                                navigate(`/onboarding/create-tenant`, {
                                  replace: true,
                                });
                              }}
                            >
                              <PlusIcon className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>New tenant</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                  </CommandItem>
                </CommandGroup>
              )}

              {otherOrganizations.length > 0 && (
                <CommandGroup
                  heading={
                    currentOrganization
                      ? 'Other Organizations'
                      : 'Organizations'
                  }
                >
                  {otherOrganizations.map((org) => (
                    <CommandItem
                      key={org.metadata.id}
                      className="text-sm hover:bg-transparent focus:bg-transparent data-[selected]:bg-transparent aria-selected:bg-transparent [&[aria-selected=true]]:bg-transparent"
                    >
                      <div className="flex items-center justify-between w-full">
                        <span className="flex-1">{org.name}</span>
                        <div className="flex items-center gap-1 ml-2">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 w-6 p-0 hover:bg-accent"
                                title="Settings"
                                onClick={(e) => {
                                  e.preventDefault();
                                  e.stopPropagation();
                                  setOpen(false);
                                  navigate(
                                    `/organizations/${org.metadata.id}`,
                                    {
                                      replace: true,
                                    },
                                  );
                                }}
                              >
                                <Cog6ToothIcon className="h-3 w-3" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Settings</TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 w-6 p-0 hover:bg-accent"
                                title="New tenant"
                                onClick={(e) => {
                                  e.preventDefault();
                                  e.stopPropagation();
                                  setOpen(false);
                                  navigate(`/onboarding/create-tenant`, {
                                    replace: true,
                                  });
                                }}
                              >
                                <PlusIcon className="h-3 w-3" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>New tenant</TooltipContent>
                          </Tooltip>
                        </div>
                      </div>
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
