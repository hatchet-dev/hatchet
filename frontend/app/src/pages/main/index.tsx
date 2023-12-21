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
  UserCircleIcon,
} from '@heroicons/react/24/outline';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import hatchet from '@/assets/hatchet_logo.png';
import invariant from 'tiny-invariant';

import {
  Command,
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';

import { Link, Outlet, useNavigate, useOutletContext } from 'react-router-dom';
import api, { TenantMember, User } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { CaretSortIcon, PlusCircledIcon } from '@radix-ui/react-icons';
import {
  PopoverTrigger,
  Popover,
  PopoverContent,
} from '@radix-ui/react-popover';
import React, { useEffect } from 'react';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { useAtom } from 'jotai';
import { currTenantAtom } from '@/lib/atoms';
import { Loading, Spinner } from '@/components/ui/loading.tsx';

function Main() {
  const { user, memberships } = useOutletContext<
    UserContextType & MembershipsContextType
  >();
  const [tenant, setTenant] = useAtom(currTenantAtom);

  useEffect(() => {
    if (!tenant && memberships && memberships.length > 0) {
      const tenant = memberships[0].tenant;
      invariant(tenant);
      setTenant(tenant);
    }
  }, [tenant, memberships, setTenant]);

  const ctx = useContextFromParent({
    user,
    memberships,
  });

  if (!user || !memberships) {
    return <Loading />;
  }

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <MainNav user={user} />
      <Sidebar memberships={memberships} />
      <div className="pt-12 flex-grow">
        <Outlet context={ctx} />
      </div>
    </div>
  );
}

export default Main;

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  memberships: TenantMember[];
}

function Sidebar({ className, memberships }: SidebarProps) {
  return (
    <div className={cn('h-full border-r max-w-xs', className)}>
      <div className="flex flex-col justify-between items-start space-y-4 px-4 py-4 h-full">
        <div className="grow">
          <div className="py-2">
            <img src={hatchet} alt="Hatchet" className="h-9 rounded mb-6" />
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
        </div>
        <TenantSwitcher memberships={memberships} />
      </div>
    </div>
  );
}

interface MainNavProps {
  user: User;
}

function MainNav({ user }: MainNavProps) {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      navigate('/auth/login');
    },
    onError: handleApiError,
  });

  return (
    <div className="absolute top-0 w-screen h-12">
      <div className="flex h-16 items-center pr-4 pl-7">
        <div className="ml-auto flex items-center space-x-4">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="relative h-10 w-10 rounded-full p-1"
              >
                <UserCircleIcon className="h-6 w-6 text-foreground cursor-pointer" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56" align="end" forceMount>
              <DropdownMenuLabel className="font-normal">
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium leading-none">
                    {user.name || user.email}
                  </p>
                  <p className="text-xs leading-none text-muted-foreground">
                    {user.email}
                  </p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              {/* <DropdownMenuGroup>
                <DropdownMenuItem>
                  Profile
                  <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem>
                  Billing
                  <DropdownMenuShortcut>⌘B</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem>
                  Settings
                  <DropdownMenuShortcut>⌘S</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem>New Team</DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator /> */}
              <DropdownMenuItem onClick={() => logoutMutation.mutate()}>
                Log out
                <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}

interface TenantSwitcherProps {
  className?: string;
  memberships: TenantMember[];
}

function TenantSwitcher({ className, memberships }: TenantSwitcherProps) {
  const [currTenant, setTenant] = useAtom(currTenantAtom);
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
                  setTenant(membership.tenant);
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
