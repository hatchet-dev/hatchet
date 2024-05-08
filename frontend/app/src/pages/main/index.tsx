import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
  AdjustmentsHorizontalIcon,
  QueueListIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';

import { Link, Outlet, useLocation, useOutletContext } from 'react-router-dom';
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
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible';

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
  }, [setSidebarOpen]);

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
        <div className="grow w-full">
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Activity
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                onNavLinkClick={onNavLinkClick}
                to="/events"
                name="Events"
                icon={<QueueListIcon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                onNavLinkClick={onNavLinkClick}
                to="/workflow-runs"
                name="Workflow Runs"
                icon={<AdjustmentsHorizontalIcon className="mr-2 h-4 w-4" />}
              />
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Resources
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                onNavLinkClick={onNavLinkClick}
                to="/workflows"
                name="Workflows"
                icon={<Squares2X2Icon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                onNavLinkClick={onNavLinkClick}
                to="/workers"
                name="Workers"
                icon={<ServerStackIcon className="mr-2 h-4 w-4" />}
              />
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Settings
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                onNavLinkClick={onNavLinkClick}
                to="/tenant-settings/overview"
                prefix="/tenant-settings"
                name="General"
                icon={<GearIcon className="mr-2 h-4 w-4" />}
                collapsibleChildren={[
                  <SidebarButtonSecondary
                    key={1}
                    onNavLinkClick={onNavLinkClick}
                    to="/tenant-settings/overview"
                    name="Overview"
                  />,
                  <SidebarButtonSecondary
                    key={2}
                    onNavLinkClick={onNavLinkClick}
                    to="/tenant-settings/api-tokens"
                    name="API Tokens"
                  />,
                  <SidebarButtonSecondary
                    key={3}
                    onNavLinkClick={onNavLinkClick}
                    to="/tenant-settings/members"
                    name="Members"
                  />,
                  <SidebarButtonSecondary
                    key={4}
                    onNavLinkClick={onNavLinkClick}
                    to="/tenant-settings/alerting"
                    name="Alerting"
                  />,
                  <SidebarButtonSecondary
                    key={4}
                    onNavLinkClick={onNavLinkClick}
                    to="/tenant-settings/ingestors"
                    name="Ingestors"
                  />,
                ]}
              />
            </div>
          </div>
        </div>
        <TenantSwitcher memberships={memberships} currTenant={currTenant} />
      </div>
    </div>
  );
}

function SidebarButtonPrimary({
  onNavLinkClick,
  to,
  name,
  icon,
  prefix,
  collapsibleChildren = [],
}: {
  onNavLinkClick: () => void;
  to: string;
  name: string;
  icon: React.ReactNode;
  prefix?: string;
  collapsibleChildren?: React.ReactNode[];
}) {
  const location = useLocation();
  const open = location.pathname.startsWith(prefix || to);
  const selected = !prefix && location.pathname === to;

  const primaryLink = (
    <Link to={to} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        className={cn(
          'w-full justify-start pl-2 cursor-default',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        {icon}
        {name}
      </Button>
    </Link>
  );

  return collapsibleChildren.length == 0 ? (
    primaryLink
  ) : (
    <Collapsible
      open={open}
      // onOpenChange={setIsOpen}
      className="w-full"
    >
      {primaryLink}
      <CollapsibleContent className={'space-y-2 ml-4 border-l border-muted'}>
        {collapsibleChildren}
      </CollapsibleContent>
    </Collapsible>
  );
}

function SidebarButtonSecondary({
  onNavLinkClick,
  to,
  name,
}: {
  onNavLinkClick: () => void;
  to: string;
  name: string;
}) {
  const location = useLocation();
  const selected = location.pathname === to;

  return (
    <Link to={to} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          'w-[calc(100%-3px)] justify-start pl-3 pr-0 ml-1 mr-3 cursor-default my-[1px]',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        {name}
      </Button>
    </Link>
  );
}
