import { Button } from '@/components/v1/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/v1/ui/command';
import { useTenantDetails } from '@/hooks/use-tenant';
import { globalEmitter } from '@/lib/global-emitter';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { useUserUniverse } from '@/providers/user-universe';
import {
  CheckIcon,
  ChevronUpDownIcon,
  PlusIcon,
} from '@heroicons/react/24/outline';
import {
  Popover,
  PopoverContent,
  PopoverPortal,
  PopoverTrigger,
} from '@radix-ui/react-popover';
import { useState, useMemo } from 'react';

export function TenantSelector({ className }: { className?: string }) {
  const { setTenant, tenant } = useTenantDetails();
  const {
    isLoaded: isUniverseLoaded,
    getOrganizationForTenant,
    getTenantWithTenantId,
    tenantMemberships,
  } = useUserUniverse();
  const { meta } = useApiMeta();
  const [open, setOpen] = useState(false);

  const currentOrg = useMemo(() => {
    if (!tenant || !getOrganizationForTenant) {
      return null;
    }
    return getOrganizationForTenant(tenant.metadata.id);
  }, [tenant, getOrganizationForTenant]);

  const tenantsToDisplay = isUniverseLoaded
    ? currentOrg
      ? currentOrg.tenants.map((tenant) => getTenantWithTenantId(tenant.id))
      : tenantMemberships.map((membership) => membership.tenant)
    : [];

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
            'w-[150px] md:w-[200px] justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
            open && 'bg-muted/30',
            className,
          )}
          disabled={!isUniverseLoaded}
        >
          <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <span className="min-w-0 flex-1 truncate">
              {tenant?.name ?? ''}
            </span>
          </div>
          {!isUniverseLoaded ? (
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground/70" />
          ) : (
            <ChevronUpDownIcon className="size-4 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent
          side="bottom"
          align="start"
          sideOffset={8}
          className="z-[200] w-[287px] rounded-md border border-border p-0 shadow-md"
        >
          <Command className="border-0">
            <CommandList data-cy="tenant-switcher-list">
              <CommandEmpty>No tenants found.</CommandEmpty>
              <CommandGroup>
                {tenantsToDisplay.map((t) => (
                  <CommandItem
                    key={t.metadata.id}
                    value={`tenant-${t.metadata.id}`}
                    onSelect={() => {
                      setTenant(t);
                      setOpen(false);
                    }}
                    data-cy={
                      t.slug ? `tenant-switcher-item-${t.slug}` : undefined
                    }
                    className="cursor-pointer text-sm hover:bg-accent focus:bg-accent"
                  >
                    <div className="flex w-full items-center justify-between">
                      <span className="min-w-0 flex-1 truncate">{t.name}</span>
                      <CheckIcon
                        className={cn(
                          'ml-2 size-4',
                          tenant?.metadata.id === t.metadata.id
                            ? 'opacity-100'
                            : 'opacity-0',
                        )}
                      />
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
            {meta?.allowCreateTenant && (
              <>
                <CommandSeparator />
                <CommandList>
                  <CommandItem
                    className="cursor-pointer text-sm"
                    data-cy="new-tenant"
                    onSelect={() => {
                      globalEmitter.emit('new-tenant', {
                        defaultOrganizationId: currentOrg?.metadata.id,
                      });
                      setOpen(false);
                    }}
                  >
                    <PlusIcon className="mr-2 size-4" />
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
