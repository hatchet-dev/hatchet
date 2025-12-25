import { useToast } from '@/components/v1/hooks/use-toast';
import { CreateOrganizationDialog } from '@/components/v1/molecules/nav-bar/create-organization-dialog';
import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { OrganizationForUser } from '@/lib/api/generated/cloud/data-contracts';
import { queries } from '@/lib/api/queries';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
import {
  // ChartBarSquareIcon,
  CheckIcon,
} from '@heroicons/react/24/outline';
import { CaretSortIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import React, { useCallback, useMemo } from 'react';
import invariant from 'tiny-invariant';

interface TenantSwitcherProps {
  className?: string;
}

const DEFAULT_TENANT_COLOR = '#3B82F6';

function TenantColorDot({ color }: { color?: string }) {
  return (
    <span
      aria-hidden
      className="size-3 shrink-0 rounded-full"
      style={{ backgroundColor: color || DEFAULT_TENANT_COLOR }}
    />
  );
}
export function TenantSwitcher({ className }: TenantSwitcherProps) {
  const { meta } = useApiMeta();
  const { isLoading: isTenantLoading, tenant, setTenant } = useTenantDetails();
  const [open, setOpen] = React.useState(false);
  const { toast } = useToast();
  const navigate = useNavigate();
  const [createOrganizationOpen, setCreateOrganizationOpen] =
    React.useState(false);
  const [orgName, setOrgName] = React.useState('');
  const {
    enabled: organizationsEnabled,
    organizations,
    getOrganizationForTenant,
    handleCreateOrganization,
    createOrganizationLoading,
    error: organizationsError,
  } = useOrganizations();
  const listMembershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  const memberships = useMemo(
    () => listMembershipsQuery.data?.rows || [],
    [listMembershipsQuery.data],
  );

  const organizationsSupported = useMemo(() => {
    // If cloud is disabled, orgs aren't in play.
    if (!organizationsEnabled) {
      return false;
    }

    // Some deployments run an API version without org endpoints.
    if (organizationsError && isAxiosError(organizationsError)) {
      const status = organizationsError.response?.status;
      if (status === 404 || status === 405 || status === 501) {
        return false;
      }
    }

    return true;
  }, [organizationsEnabled, organizationsError]);

  const [hoveredOrg, setHoveredOrg] = React.useState<
    OrganizationForUser | undefined
  >(undefined);

  const [hoveredAction, setHoveredAction] = React.useState<
    'create-organization' | undefined
  >(undefined);

  const selectedOrg = useMemo(() => {
    const tenantId = tenant?.metadata.id;

    if (!tenantId) {
      return undefined;
    }

    return getOrganizationForTenant(tenantId);
  }, [getOrganizationForTenant, tenant?.metadata.id]);

  const activeOrg = hoveredAction ? undefined : (hoveredOrg ?? selectedOrg);

  const activeOrgMemberships = useMemo(() => {
    if (!activeOrg) {
      return [];
    }

    return memberships.filter((membership) =>
      activeOrg.tenants.some((t) => t.id === membership.tenant?.metadata.id),
    );
  }, [activeOrg, memberships]);

  const reset = useCallback(() => {
    setHoveredOrg(undefined);
    setHoveredAction(undefined);
  }, [setHoveredOrg, setHoveredAction]);

  if (!tenant) {
    return (
      <Button
        variant="outline"
        size="sm"
        aria-label="Loading tenant"
        className={cn(
          'min-w-0 justify-between gap-2 bg-muted/20 shadow-none',
          className,
        )}
        style={{ borderColor: DEFAULT_TENANT_COLOR }}
        disabled
      >
        <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <TenantColorDot />
          <span className="min-w-0 flex-1 truncate text-muted-foreground">
            Loading tenant…
          </span>
        </div>
        <Spinner className="mr-0" />
      </Button>
    );
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) {
          reset();
        }
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a tenant"
          className={cn(
            'min-w-0 justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
            open && 'bg-muted/30',
            className,
          )}
          style={{ borderColor: tenant.color || DEFAULT_TENANT_COLOR }}
          disabled={isTenantLoading || memberships.length === 0}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <TenantColorDot color={tenant.color} />
            <span className="min-w-0 flex-1 truncate">{tenant.name}</span>
          </div>
          {isTenantLoading ? (
            <Spinner className="mr-0" />
          ) : (
            <CaretSortIcon className="size-4 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      {organizationsSupported ? (
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          onMouseLeave={() => {
            reset();
            //   TODO reset if mouse enters the whitespace between the last org and the action
          }}
          // Must render above the mobile sidebar overlay (`side-nav` uses z-[100]).
          className={cn(
            // Deterministic 2-column layout: lock the overall width and column widths so
            // long org/tenant names don't resize the popover.
            'z-[300] grid p-0',
            'w-[520px] max-w-[calc(100vw-2rem)]',
            'grid-cols-[260px_260px]',
            // Our PopoverContent already brings border/bg/shadow/etc; we just ensure we clip overflow.
            'overflow-hidden',
            // Visual separation between columns.
            'divide-x divide-border',
          )}
        >
          {/* Organizations list */}
          <Command className="col-span-1 h-[300px] min-w-0 rounded-none">
            <CommandList
              data-cy="organization-switcher-list"
              className="max-h-none flex-1"
            >
              <CommandGroup heading="Organizations">
                {organizations.length === 0 ? (
                  <div className="px-2 py-6 text-sm text-muted-foreground">
                    No organizations yet.
                  </div>
                ) : (
                  organizations.map((org) => (
                    <CommandItem
                      key={org.metadata.id}
                      onMouseEnter={() => {
                        setHoveredAction(undefined);
                        setHoveredOrg(org);
                      }}
                      onSelect={() => {
                        // setTenant(org.tenants?.[0]);
                        setOpen(false);
                      }}
                      value={org.name}
                      data-cy={`organization-switcher-item-${org.name}`}
                      className="cursor-pointer gap-2 text-sm min-w-0"
                    >
                      <span className="min-w-0 flex-1 truncate">
                        {org.name}
                      </span>
                      <CheckIcon
                        className={cn(
                          'ml-auto size-4',
                          selectedOrg?.metadata.id === org.metadata.id
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </CommandItem>
                  ))
                )}
              </CommandGroup>
            </CommandList>
            {meta?.allowCreateTenant && (
              <div className="border-t border-border p-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-9 w-full justify-start gap-2 px-2 text-xs text-muted-foreground hover:text-foreground"
                  data-cy="create-organization"
                  onMouseEnter={() => {
                    setHoveredOrg(undefined);
                    setHoveredAction('create-organization');
                  }}
                  onClick={() => {
                    setOpen(false);
                    reset();
                    setCreateOrganizationOpen(true);
                  }}
                >
                  <PlusCircledIcon className="size-3.5 shrink-0 opacity-70" />
                  Create organization
                </Button>
              </div>
            )}
          </Command>

          {/* Right column */}
          {hoveredAction === 'create-organization' ? (
            <div className="col-span-1 flex h-[300px] min-w-0 flex-col gap-3 p-4">
              <div className="space-y-1">
                <div className="text-sm font-medium">Create organization</div>
                <div className="text-xs text-muted-foreground">
                  Organizations group tenants. Pick a name, invite teammates,
                  and then create tenants underneath.
                </div>
              </div>
              <div className="flex flex-1 flex-col items-center justify-center gap-3 rounded-md border border-dashed border-border bg-muted/10 p-4 text-center text-xs text-muted-foreground">
                <div>Create a new organization to group tenants.</div>
                <Button
                  type="button"
                  size="sm"
                  onClick={() => {
                    setOpen(false);
                    reset();
                    setCreateOrganizationOpen(true);
                  }}
                  disabled={createOrganizationLoading}
                >
                  {createOrganizationLoading
                    ? 'Creating...'
                    : 'Create organization'}
                </Button>
              </div>
            </div>
          ) : (
            /* Tenants list */
            <Command className="col-span-1 h-[300px] min-w-0 rounded-none">
              <CommandList
                data-cy="tenant-switcher-list"
                className="max-h-none flex-1"
              >
                <CommandGroup heading="Tenants">
                  {!activeOrg ? (
                    <div className="px-2 py-6 text-sm text-muted-foreground">
                      Select an organization to view its tenants.
                    </div>
                  ) : activeOrgMemberships.length === 0 ? (
                    <div className="px-2 py-6 text-sm text-muted-foreground">
                      No tenants in this organization.
                    </div>
                  ) : (
                    activeOrgMemberships.map((membership) => (
                      <CommandItem
                        key={membership.tenant?.metadata.id}
                        onSelect={() => {
                          invariant(membership.tenant);
                          setTenant(membership.tenant);
                          setOpen(false);
                        }}
                        value={membership.tenant?.slug}
                        data-cy={`tenant-switcher-item-${membership.tenant?.slug}`}
                        className="cursor-pointer gap-2 text-sm min-w-0"
                      >
                        <TenantColorDot color={membership.tenant?.color} />
                        <span className="min-w-0 flex-1 truncate">
                          {membership.tenant?.name}
                        </span>
                        <CheckIcon
                          className={cn(
                            'ml-auto size-4',
                            tenant?.metadata.id ===
                              membership.tenant?.metadata.id
                              ? 'opacity-100'
                              : 'opacity-0',
                          )}
                        />
                      </CommandItem>
                    ))
                  )}
                </CommandGroup>
              </CommandList>
              {meta?.allowCreateTenant && (
                <div className="border-t border-border p-1">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-9 w-full justify-start gap-2 px-2 text-xs text-muted-foreground hover:text-foreground"
                    data-cy="create-tenant"
                    onClick={() => {
                      setOpen(false);
                      void navigate({
                        to: appRoutes.onboardingCreateTenantRoute.to,
                        search: activeOrg
                          ? { step: 2, organizationId: activeOrg.metadata.id }
                          : { step: 2 },
                      });
                    }}
                  >
                    <PlusCircledIcon className="size-3.5 shrink-0 opacity-70" />
                    Create tenant
                  </Button>
                </div>
              )}
            </Command>
          )}
        </PopoverContent>
      ) : (
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          // Must render above the mobile sidebar overlay (`side-nav` uses z-[100]).
          className={cn(
            'z-[300] p-0',
            'w-[320px] max-w-[calc(100vw-2rem)]',
            'overflow-hidden',
          )}
        >
          <Command className="h-[300px] min-w-0 rounded-none">
            <CommandList
              data-cy="tenant-switcher-list"
              className="max-h-none flex-1"
            >
              <CommandGroup heading="Tenants">
                {memberships.length === 0 ? (
                  <div className="px-2 py-6 text-sm text-muted-foreground">
                    No tenants yet.
                  </div>
                ) : (
                  memberships.map((membership) => (
                    <CommandItem
                      key={membership.tenant?.metadata.id}
                      onSelect={() => {
                        invariant(membership.tenant);
                        setTenant(membership.tenant);
                        setOpen(false);
                      }}
                      value={membership.tenant?.slug}
                      data-cy={`tenant-switcher-item-${membership.tenant?.slug}`}
                      className="cursor-pointer gap-2 text-sm min-w-0"
                    >
                      <TenantColorDot color={membership.tenant?.color} />
                      <span className="min-w-0 flex-1 truncate">
                        {membership.tenant?.name}
                      </span>
                      <CheckIcon
                        className={cn(
                          'ml-auto size-4',
                          tenant?.metadata.id === membership.tenant?.metadata.id
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </CommandItem>
                  ))
                )}
              </CommandGroup>
            </CommandList>
            {meta?.allowCreateTenant && (
              <div className="border-t border-border p-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-9 w-full justify-start gap-2 px-2 text-xs text-muted-foreground hover:text-foreground"
                  data-cy="create-tenant"
                  onClick={() => {
                    setOpen(false);
                    void navigate({
                      to: appRoutes.onboardingCreateTenantRoute.to,
                      search: { step: 2 },
                    });
                  }}
                >
                  <PlusCircledIcon className="size-3.5 shrink-0 opacity-70" />
                  Create tenant
                </Button>
              </div>
            )}
          </Command>
        </PopoverContent>
      )}
      <CreateOrganizationDialog
        open={createOrganizationOpen}
        onOpenChange={(nextOpen) => {
          setCreateOrganizationOpen(nextOpen);
          if (!nextOpen) {
            setOrgName('');
          }
        }}
        orgName={orgName}
        setOrgName={setOrgName}
        createOrganizationLoading={createOrganizationLoading}
        onCreate={(name) => {
          handleCreateOrganization(name, () => {
            setCreateOrganizationOpen(false);
            setOrgName('');
            toast({
              title: 'Organization created',
              description: 'You can now create tenants under it.',
            });
          });
        }}
      />
    </Popover>
  );
}
