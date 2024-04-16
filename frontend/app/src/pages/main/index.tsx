import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
  AdjustmentsHorizontalIcon,
  QueueListIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';

import { Link, Outlet, useOutletContext } from 'react-router-dom';
import { Tenant, TenantMember } from '@/lib/api';
import { GearIcon } from '@radix-ui/react-icons';
import React, { useCallback } from 'react';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import { useTenantContext } from '@/lib/atoms';
import { Loading } from '@/components/ui/loading.tsx';
import { useSidebar } from '@/components/sidebar-provider';
import { TenantSwitcher } from '@/components/molecules/nav-bar/tenant-switcher';

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
  const { sidebarOpen, setSidebarOpen } = useSidebar();

  const onNavLinkClick = useCallback(() => {
    if (window.innerWidth > 768) {
      return;
    }

    setSidebarOpen('closed');
  }, [window]);

  if (sidebarOpen === 'closed') {
    return null;
  }

  return (
    <div
      className={cn(
        'h-full border-r w-full md:w-80 top-16 absolute z-[100] md:relative md:top-0 md:bg-[unset] md:dark:bg-[unset] bg-slate-100 dark:bg-slate-900',
        className,
      )}
    >
      <div className="flex flex-col justify-between items-start space-y-4 px-4 py-4 h-full pb-16 md:pb-0">
        <div className="grow">
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Events
            </h2>
            <div className="space-y-1">
              <Link to="/events" onClick={onNavLinkClick}>
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
              <Link to="/workflows" onClick={onNavLinkClick}>
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <Squares2X2Icon className="mr-2 h-4 w-4" />
                  All workflows
                </Button>
              </Link>
              <Link to="/workflow-runs" onClick={onNavLinkClick}>
                <Button variant="ghost" className="w-full justify-start pl-0">
                  <AdjustmentsHorizontalIcon className="mr-2 h-4 w-4" />
                  Runs
                </Button>
              </Link>
              <Link to="/workers" onClick={onNavLinkClick}>
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
              <Link to="/tenant-settings" onClick={onNavLinkClick}>
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
