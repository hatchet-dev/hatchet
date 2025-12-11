import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import {
  CalendarDaysIcon,
  CpuChipIcon,
  PlayIcon,
  ScaleIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';

import { SidePanel } from '@/components/side-panel';
import { useSidebar } from '@/components/sidebar-provider';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import {
  Collapsible,
  CollapsibleContent,
} from '@/components/v1/ui/collapsible';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { SidePanelProvider } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TenantMember } from '@/lib/api';
import {
  MembershipsContextType,
  UserContextType,
  useContextFromParent,
} from '@/lib/outlet';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import useCloudFeatureFlags from '@/pages/auth/hooks/use-cloud-feature-flags';
import { ClockIcon, GearIcon } from '@radix-ui/react-icons';
import { Filter, SquareActivityIcon, WebhookIcon } from 'lucide-react';
import React, { useCallback } from 'react';
import { Link, useLocation } from '@tanstack/react-router';
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import { appRoutes } from '@/router';

function Main() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;

  const childCtx = useContextFromParent({
    user,
    memberships,
  });

  if (!user || !memberships) {
    return <Loading />;
  }

  return (
    <SidePanelProvider>
      <div className="flex flex-row flex-1 w-full h-full">
        <Sidebar memberships={memberships} />
        <div
          className="px-8 py-4 flex-grow overflow-y-auto overflow-x-hidden"
          style={{ containerType: 'inline-size' }}
        >
          <OutletWithContext context={childCtx} />
        </div>
        <SidePanel />
      </div>
    </SidePanelProvider>
  );
}

export default Main;

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  memberships: TenantMember[];
}

function Sidebar({ className, memberships }: SidebarProps) {
  const { sidebarOpen, setSidebarOpen } = useSidebar();
  const { tenantId } = useCurrentTenantId();

  const { data: cloudMeta, isCloudEnabled } = useCloudApiMeta();
  const featureFlags = useCloudFeatureFlags(tenantId);

  const onNavLinkClick = useCallback(() => {
    if (window.innerWidth > 768) {
      return;
    }

    setSidebarOpen('closed');
  }, [setSidebarOpen]);

  if (sidebarOpen === 'closed') {
    return null;
  }

  const workers = [
    <SidebarButtonSecondary
      key="all-workers"
      onNavLinkClick={onNavLinkClick}
      to={appRoutes.tenantWorkersAllRoute.to}
      params={{ tenant: tenantId }}
      name="All Workers"
    />,
    <SidebarButtonSecondary
      key="webhook-workers"
      onNavLinkClick={onNavLinkClick}
      to={appRoutes.tenantWorkersWebhookRoute.to}
      params={{ tenant: tenantId }}
      name="Webhook Workers"
    />,
  ];

  return (
    <div
      className={cn(
        'h-full border-r overflow-y-auto w-full md:min-w-64 md:w-64 top-16 absolute z-[100] md:relative md:top-0 md:bg-[unset] md:dark:bg-[unset] bg-slate-100 dark:bg-slate-900',
        className,
      )}
    >
      <div className="flex flex-col justify-between items-start space-y-4 px-4 py-4 h-full pb-16 md:pb-4">
        <div className="grow w-full">
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Activity
            </h2>
            <div className="flex flex-col gap-y-1">
              <SidebarButtonPrimary
                key="runs"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantRunsRoute.to}
                params={{ tenant: tenantId }}
                name="Runs"
                icon={<PlayIcon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                key="events"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantEventsRoute.to}
                params={{ tenant: tenantId }}
                name="Events"
                icon={<SquareActivityIcon className="mr-2 h-4 w-4" />}
              />
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Triggers
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                key="scheduled"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantScheduledRoute.to}
                params={{ tenant: tenantId }}
                name="Scheduled Runs"
                icon={<CalendarDaysIcon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                key="crons"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantCronJobsRoute.to}
                params={{ tenant: tenantId }}
                name="Cron Jobs"
                icon={<ClockIcon className="mr-2 h-4 w-4" />}
              />
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Resources
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                key="workflows"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantWorkflowsRoute.to}
                params={{ tenant: tenantId }}
                name="Workflows"
                icon={<Squares2X2Icon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                key="workers"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantWorkersRoute.to}
                params={{ tenant: tenantId }}
                name="Workers"
                icon={<ServerStackIcon className="mr-2 h-4 w-4" />}
                prefix="/tenants/$tenant/workers"
                collapsibleChildren={workers}
              />
              <SidebarButtonPrimary
                key="webhooks"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantWebhooksRoute.to}
                params={{ tenant: tenantId }}
                name="Webhooks"
                icon={<WebhookIcon className="mr-2 h-4 w-4" />}
              />
              {featureFlags?.data['managed-worker'] && (
                <SidebarButtonPrimary
                  key="managed-compute"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantManagedWorkersRoute.to}
                  params={{ tenant: tenantId }}
                  name="Managed Compute"
                  icon={<CpuChipIcon className="mr-2 h-4 w-4" />}
                />
              )}
              <SidebarButtonPrimary
                key="rate-limits"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantRateLimitsRoute.to}
                params={{ tenant: tenantId }}
                name="Rate Limits"
                icon={<ScaleIcon className="mr-2 h-4 w-4" />}
              />
              <SidebarButtonPrimary
                key="filters"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantFiltersRoute.to}
                params={{ tenant: tenantId }}
                name="Filters"
                icon={<Filter className="mr-2 h-4 w-4" />}
              />
            </div>
          </div>
          <div className="py-2">
            <h2 className="mb-2 text-lg font-semibold tracking-tight">
              Settings
            </h2>
            <div className="space-y-1">
              <SidebarButtonPrimary
                key="tenant-settings"
                onNavLinkClick={onNavLinkClick}
                to={appRoutes.tenantSettingsOverviewRoute.to}
                params={{ tenant: tenantId }}
                prefix="/tenants/$tenant/tenant-settings"
                name="General"
                icon={<GearIcon className="mr-2 h-4 w-4" />}
                collapsibleChildren={[
                  <SidebarButtonSecondary
                    key="tenant-settings-overview"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsOverviewRoute.to}
                    params={{ tenant: tenantId }}
                    name="Overview"
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-api-tokens"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsApiTokensRoute.to}
                    params={{ tenant: tenantId }}
                    name="API Tokens"
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-github"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsGithubRoute.to}
                    params={{ tenant: tenantId }}
                    name="Github"
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-members"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsMembersRoute.to}
                    params={{ tenant: tenantId }}
                    name="Members"
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-billing-and-limits"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsBillingRoute.to}
                    params={{ tenant: tenantId }}
                    name={
                      cloudMeta?.data.canBill
                        ? 'Billing & Limits'
                        : 'Resource Limits'
                    }
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-alerting"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsAlertingRoute.to}
                    params={{ tenant: tenantId }}
                    name="Alerting"
                  />,
                  <SidebarButtonSecondary
                    key="tenant-settings-ingestors"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantSettingsIngestorsRoute.to}
                    params={{ tenant: tenantId }}
                    name="Ingestors"
                  />,
                  <SidebarButtonSecondary
                    key="quickstart"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantOnboardingGetStartedRoute.to}
                    params={{ tenant: tenantId }}
                    name="Quickstart"
                  />,
                ]}
              />
            </div>
          </div>
        </div>
        {isCloudEnabled ? (
          <OrganizationSelector memberships={memberships} />
        ) : (
          <TenantSwitcher memberships={memberships} />
        )}
      </div>
    </div>
  );
}

function SidebarButtonPrimary({
  onNavLinkClick,
  to,
  params,
  name,
  icon,
  prefix,
  collapsibleChildren = [],
}: {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  icon: React.ReactNode;
  prefix?: string;
  collapsibleChildren?: React.ReactNode[];
}) {
  const location = useLocation();
  const open = location.pathname.startsWith(prefix || to);
  const selected = !prefix && location.pathname === to;

  const primaryLink = (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        className={cn(
          'w-full justify-start pl-2',
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
  params,
  name,
  prefix,
}: {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  prefix?: string;
}) {
  const location = useLocation();
  const hasPrefix = prefix && location.pathname.startsWith(prefix);
  const selected = hasPrefix || location.pathname === to;

  return (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          'w-[calc(100%-3px)] justify-start pl-3 pr-0 ml-1 mr-3 my-[1px]',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        {name}
      </Button>
    </Link>
  );
}
