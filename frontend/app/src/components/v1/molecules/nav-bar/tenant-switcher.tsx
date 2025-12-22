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
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { appRoutes } from '@/router';
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
} from '@radix-ui/react-popover';
import { Link } from '@tanstack/react-router';
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
  const meta = useApiMeta();
  const { setTenant: setCurrTenant, tenant: currTenant } = useTenantDetails();
  const [open, setOpen] = React.useState(false);
  const { hasOrganizations } = useOrganizations();

  if (!currTenant) {
    return <Spinner />;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a tenant"
          className={cn('justify-between', className)}
          fullWidth
        >
          <BuildingOffice2Icon className="mr-2 size-4" />
          {currTenant.name}
          <CaretSortIcon className="ml-auto size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent side="right" className="z-50 mb-6 w-full p-0">
        <Command className="min-w-[260px]" value={currTenant.slug}>
          <CommandList>
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
                className="cursor-pointer text-sm"
              >
                <BuildingOffice2Icon className="mr-2 size-4" />
                {membership.tenant?.name}
                <CheckIcon
                  className={cn(
                    'ml-auto size-4',
                    currTenant.slug === membership.tenant?.slug
                      ? 'opacity-100'
                      : 'opacity-0',
                  )}
                />
              </CommandItem>
            ))}
          </CommandList>
          {meta.data?.allowCreateTenant && !hasOrganizations && (
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
      </PopoverContent>
    </Popover>
  );
}
