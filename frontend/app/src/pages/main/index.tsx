import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
  AdjustmentsHorizontalIcon,
  BuildingOffice2Icon,
  // ChartBarSquareIcon,
  CheckIcon,
  QueueListIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';
import invariant from 'tiny-invariant';

import {
  Command,
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';

import { Link, Outlet, useOutletContext } from 'react-router-dom';
import { Tenant, TenantMember } from '@/lib/api';
import {
  CaretSortIcon,
  GearIcon,
  PlusCircledIcon,
} from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import React from 'react';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { useTenantContext } from '@/lib/atoms';
import { Loading, Spinner } from '@/components/ui/loading.tsx';
import { useSidebar } from '@/components/sidebar-provider';

function Main() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();

  const { user, memberships } = ctx;

  const [currTenant] = useTenantContext();

  const childCtx = useContextFromParent({
    user,
    memberships,
    tenant: currTenant,
  });

  if (!user || !memberships || !currTenant) {
    return <Loading />;
  }

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <Sidebar memberships={memberships} currTenant={currTenant} />
      <div className="pt-6 flex-grow overflow-y-auto overflow-x-hidden">
        <Outlet context={childCtx} />
      </div>
    </div>
  );
}

export default Main;

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  memberships: TenantMember[];
  currTenant: Tenant;
}

function Sidebar({ className, memberships, currTenant }: SidebarProps) {
  const { sidebarOpen } = useSidebar();

  if (sidebarOpen === 'closed') {
    return null;
  }

  return (
    <div
      className={cn(
        'h-full border-r w-full md:w-80 top-16 absolute z-50 md:relative md:top-0 md:bg-[unset] bg-slate-900',
        className,
      )}
    >
      <div className="flex flex-col justify-between items-start space-y-4 px-4 py-4 h-full">
        <div className="grow">
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Events
            </h2>
            <div className="space-y-1">
              <Link to="/events">
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <QueueListIcon className="mr-2 h-4 w-4" />
                  All events
                </Button>
              </Link>
              {/* <Button variant="ghost" className="w-full justify-start pl-0">
                <ChartBarSquareIcon className="mr-2 h-4 w-4" />
                Metrics
              </Button> */}
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Workflows
            </h2>
            <div className="space-y-1">
              <Link to="/workflows">
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <Squares2X2Icon className="mr-2 h-4 w-4" />
                  All workflows
                </Button>
              </Link>
              <Link to="/workflow-runs">
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <AdjustmentsHorizontalIcon className="mr-2 h-4 w-4" />
                  Runs
                </Button>
              </Link>
              <Link to="/workers">
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <ServerStackIcon className="mr-2 h-4 w-4" />
                  Workers
                </Button>
              </Link>
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Settings
            </h2>
            <div className="space-y-1">
              <Link to="/tenant-settings">
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <GearIcon className="mr-2 h-4 w-4" />
                  General
                </Button>
              </Link>
            </div>
          </div>
        </div>
        <TenantSwitcher memberships={memberships} currTenant={currTenant} />
      </div>
    </div>
  );
}

interface TenantSwitcherProps {
  className?: string;
  memberships: TenantMember[];
  currTenant: Tenant;
}

function TenantSwitcher({
  className,
  memberships,
  currTenant,
}: TenantSwitcherProps) {
  const setCurrTenant = useTenantContext()[1];
  const [open, setOpen] = React.useState(false);

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
          aria-label="Select a team"
          className={cn('w-full justify-between', className)}
        >
          <BuildingOffice2Icon className="mr-2 h-4 w-4" />
          {currTenant.name}
          <CaretSortIcon className="ml-auto h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent side="right" className="w-full p-0 mb-6 z-50">
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
          <CommandSeparator />
          <CommandList>
            <Link to="/onboarding/create-tenant">
              <CommandItem className="text-sm cursor-pointer">
                <PlusCircledIcon className="mr-2 h-4 w-4" />
                New Tenant
              </CommandItem>
            </Link>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
