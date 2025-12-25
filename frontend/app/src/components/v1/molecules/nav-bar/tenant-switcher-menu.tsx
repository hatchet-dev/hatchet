import { TenantColorDot } from '@/components/v1/molecules/nav-bar/tenant-color-dot';
import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandGroup,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/v1/ui/command';
import { TenantMember } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { CheckIcon, PlusIcon } from '@heroicons/react/24/outline';
import { PlusCircledIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import invariant from 'tiny-invariant';
import * as React from 'react';

export function TenantSwitcherMenu({
  shouldGroupByOrganization,
  memberships,
  tenantSlug,
  orgGroups,
  standaloneTenants,
  onSelectTenant,
  onClose,
  onManageOrganizations,
  onCreateOrganizationClick,
  allowCreateTenant,
  hasOrganizations,
}: {
  shouldGroupByOrganization: boolean;
  memberships: TenantMember[];
  tenantSlug?: string;
  orgGroups: Array<{
    organization: { metadata: { id: string }; name: string };
    tenants: TenantMember[];
  }>;
  standaloneTenants: TenantMember[];
  onSelectTenant: (tenant: NonNullable<TenantMember['tenant']>) => void;
  onClose: () => void;
  onManageOrganizations: () => void;
  onCreateOrganizationClick: () => void;
  allowCreateTenant: boolean;
  hasOrganizations: boolean;
}) {
  const currentGroupId = React.useMemo(() => {
    if (!shouldGroupByOrganization) {
      return null;
    }

    // If the current tenant belongs to an org group, default to that group.
    const groupForCurrent = orgGroups.find((g) =>
      g.tenants.some((m) => m.tenant?.slug === tenantSlug),
    );

    if (groupForCurrent) {
      return groupForCurrent.organization.metadata.id;
    }

    if (orgGroups.length > 0) {
      return orgGroups[0].organization.metadata.id;
    }

    if (standaloneTenants.length > 0) {
      return '__standalone__';
    }

    return null;
  }, [orgGroups, shouldGroupByOrganization, standaloneTenants.length, tenantSlug]);

  const [selectedGroupId, setSelectedGroupId] = React.useState<string | null>(
    currentGroupId,
  );

  React.useEffect(() => {
    setSelectedGroupId(currentGroupId);
  }, [currentGroupId]);

  const selectedTenants: TenantMember[] = React.useMemo(() => {
    if (!shouldGroupByOrganization) {
      return memberships;
    }

    if (!selectedGroupId) {
      return [];
    }

    if (selectedGroupId === '__standalone__') {
      return standaloneTenants;
    }

    return (
      orgGroups.find((g) => g.organization.metadata.id === selectedGroupId)
        ?.tenants ?? []
    );
  }, [
    memberships,
    orgGroups,
    selectedGroupId,
    shouldGroupByOrganization,
    standaloneTenants,
  ]);

  return (
    <>
      {shouldGroupByOrganization ? (
        <div className="grid grid-cols-2 overflow-hidden rounded-md bg-popover text-popover-foreground">
          {/* Teams */}
          <div className="min-w-0">
            <Command className="rounded-none border-0">
              <CommandInput placeholder="Find Team..." />
              <div className="px-3 py-2 text-xs text-muted-foreground">
                Teams
              </div>
              <CommandList
                data-cy="tenant-switcher-teams-list"
                className="max-h-[340px]"
              >
                <CommandEmpty>No teams found.</CommandEmpty>

                {orgGroups.map(({ organization }) => (
                  <CommandItem
                    key={organization.metadata.id}
                    value={organization.name}
                    onSelect={() => setSelectedGroupId(organization.metadata.id)}
                    className="cursor-pointer py-2"
                    data-cy={`tenant-switcher-team-${organization.metadata.id}`}
                  >
                    <div className="flex w-full items-center justify-between gap-2">
                      <span className="min-w-0 flex-1 truncate">
                        {organization.name}
                      </span>
                      <CheckIcon
                        className={cn(
                          'size-4 shrink-0',
                          selectedGroupId === organization.metadata.id
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </div>
                  </CommandItem>
                ))}

                {standaloneTenants.length > 0 && (
                  <>
                    <CommandSeparator />
                    <CommandItem
                      value="Tenants"
                      onSelect={() => setSelectedGroupId('__standalone__')}
                      className="cursor-pointer py-2"
                      data-cy="tenant-switcher-team-standalone"
                    >
                      <div className="flex w-full items-center justify-between gap-2">
                        <span className="min-w-0 flex-1 truncate">Tenants</span>
                        <CheckIcon
                          className={cn(
                            'size-4 shrink-0',
                            selectedGroupId === '__standalone__'
                              ? 'opacity-100'
                              : 'opacity-0',
                          )}
                        />
                      </div>
                    </CommandItem>
                  </>
                )}
              </CommandList>
            </Command>

            <div className="border-t p-2">
              <Button
                variant="outline"
                size="sm"
                fullWidth
                onClick={() => {
                  onClose();
                  onCreateOrganizationClick();
                }}
                leftIcon={<PlusIcon className="size-4" />}
                data-cy="tenant-switcher-create-team"
              >
                Create Team
              </Button>
            </div>
          </div>

          {/* Projects */}
          <div className="min-w-0 border-l">
            <Command className="rounded-none border-0">
              <CommandInput placeholder="Find Project..." />
              <div className="px-3 py-2 text-xs text-muted-foreground">
                Projects
              </div>
              <CommandList
                data-cy="tenant-switcher-projects-list"
                className="max-h-[340px]"
              >
                <CommandEmpty>No projects found.</CommandEmpty>

                {selectedTenants
                  .slice()
                  .sort(
                    (a, b) =>
                      a.tenant?.name
                        ?.toLowerCase()
                        .localeCompare(b.tenant?.name?.toLowerCase() ?? '') ??
                      0,
                  )
                  .map((membership) => (
                    <CommandItem
                      key={membership.metadata.id}
                      value={membership.tenant?.name ?? membership.metadata.id}
                      onSelect={() => {
                        invariant(membership.tenant);
                        onSelectTenant(membership.tenant);
                        onClose();
                      }}
                      className="cursor-pointer gap-2 py-2"
                      data-cy={
                        membership.tenant?.slug
                          ? `tenant-switcher-item-${membership.tenant.slug}`
                          : undefined
                      }
                    >
                      <TenantColorDot color={membership.tenant?.color} />
                      <span className="min-w-0 flex-1 truncate">
                        {membership.tenant?.name}
                      </span>
                      <CheckIcon
                        className={cn(
                          'ml-auto size-4 shrink-0',
                          tenantSlug === membership.tenant?.slug
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </CommandItem>
                  ))}
              </CommandList>
            </Command>

            <div className="border-t p-2">
              {allowCreateTenant ? (
                <Link
                  to={appRoutes.onboardingCreateTenantRoute.to}
                  data-cy="new-tenant"
                  onClick={() => onClose()}
                >
                  <Button
                    variant="outline"
                    size="sm"
                    fullWidth
                    leftIcon={<PlusCircledIcon className="size-4" />}
                  >
                    Create Project
                  </Button>
                </Link>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  fullWidth
                  onClick={() => {
                    onClose();
                    onManageOrganizations();
                  }}
                >
                  Manage organizations
                </Button>
              )}
            </div>
          </div>
        </div>
      ) : (
        <Command className="">
          <CommandList data-cy="tenant-switcher-list">
            <CommandEmpty>No tenants found.</CommandEmpty>
            {memberships.map((membership) => (
              <CommandItem
                key={membership.metadata.id}
                onSelect={() => {
                  invariant(membership.tenant);
                  onSelectTenant(membership.tenant);
                  onClose();
                }}
                value={membership.tenant?.slug}
                data-cy={
                  membership.tenant?.slug
                    ? `tenant-switcher-item-${membership.tenant.slug}`
                    : undefined
                }
                className="cursor-pointer text-sm gap-2"
              >
                <TenantColorDot color={membership.tenant?.color} />
                {membership.tenant?.name}
                <CheckIcon
                  className={cn(
                    'ml-auto size-4',
                    tenantSlug === membership.tenant?.slug
                      ? 'opacity-100'
                      : 'opacity-0',
                  )}
                />
              </CommandItem>
            ))}
          </CommandList>

          {allowCreateTenant && !hasOrganizations && (
            <>
              <CommandSeparator />
              <CommandList>
                <Link
                  to={appRoutes.onboardingCreateTenantRoute.to}
                  data-cy="new-tenant"
                >
                  <CommandItem className="cursor-pointer text-sm">
                    <PlusCircledIcon className="mr-2 size-4" />
                    New Tenant
                  </CommandItem>
                </Link>
              </CommandList>
            </>
          )}
        </Command>
      )}
    </>
  );
}
