import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/v1/ui/command';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import { TenantMember } from '@/lib/api';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import {
  BuildingOffice2Icon,
  // ChartBarSquareIcon,
  CheckIcon,
} from '@heroicons/react/24/outline';
import { CaretSortIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
  PopoverPortal,
} from '@radix-ui/react-popover';
import React from 'react';
import invariant from 'tiny-invariant';

interface TenantSwitcherProps {
  className?: string;
  memberships: TenantMember[];
}
export function TenantSwitcher({
  className,
  memberships,
}: TenantSwitcherProps) {
  const { meta } = useApiMeta();
  const {
    setTenant: setCurrTenant,
    isUserUniverseLoaded,
    tenant,
  } = useTenantDetails();
  const [open, setOpen] = React.useState(false);
  const { hasOrganizations } = useOrganizations();

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
        disabled
      >
        <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <BuildingOffice2Icon className="size-4 shrink-0 opacity-60" />
          <span className="min-w-0 flex-1 truncate text-muted-foreground">
            Loading tenant…
          </span>
        </div>
        <Spinner className="mr-0" />
      </Button>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
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
          disabled={!isUserUniverseLoaded || memberships.length === 0}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <BuildingOffice2Icon className="size-4 shrink-0" />
            <span className="min-w-0 flex-1 truncate">{tenant.name}</span>
          </div>
          {!isUserUniverseLoaded ? (
            <Spinner className="mr-0" />
          ) : (
            <CaretSortIcon className="size-4 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      {/* Portal so the popover can render above the mobile sidebar overlay (header is z-50). */}
      <PopoverPortal>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          // Must render above the mobile sidebar overlay (`side-nav` uses z-[100]).
          className="z-[200] w-56 p-0"
        >
          <Command className="">
            <CommandList data-cy="tenant-switcher-list">
              <CommandEmpty>No tenants found.</CommandEmpty>
              {memberships.map((membership) => (
                <CommandItem
                  key={membership.metadata.id}
                  onSelect={() => {
                    invariant(membership.tenant);
                    setCurrTenant(membership.tenant);
                    setOpen(false);
                  }}
                  value={membership.tenant?.slug}
                  data-cy={
                    membership.tenant?.slug
                      ? `tenant-switcher-item-${membership.tenant.slug}`
                      : undefined
                  }
                  className="cursor-pointer text-sm"
                >
                  <BuildingOffice2Icon className="mr-2 size-4" />
                  {membership.tenant?.name}
                  <CheckIcon
                    className={cn(
                      'ml-auto size-4',
                      tenant?.slug === membership.tenant?.slug
                        ? 'opacity-100'
                        : 'opacity-0',
                    )}
                  />
                </CommandItem>
              ))}
            </CommandList>
            {meta?.allowCreateTenant && !hasOrganizations && (
              <>
                <CommandSeparator />
                <CommandList>
                  <CommandItem
                    className="cursor-pointer text-sm"
                    data-cy="new-tenant"
                    onSelect={() => {
                      globalEmitter.emit('new-tenant', {});
                      setOpen(false);
                    }}
                  >
                    <PlusCircledIcon className="mr-2 size-4" />
                    New Tenant
                  </CommandItem>
                </CommandList>
              </>
            )}
          </Command>
        </PopoverContent>
      </PopoverPortal>
    </Popover>
  );
}
