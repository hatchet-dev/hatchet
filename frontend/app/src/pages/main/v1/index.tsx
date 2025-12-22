import { ThreeColumnLayout } from '@/components/layout/three-column-layout';
import { SidePanel } from '@/components/side-panel';
import { useSidebar } from '@/components/sidebar-provider';
import { OrganizationSelector } from '@/components/v1/molecules/nav-bar/organization-selector';
import { TenantSwitcher } from '@/components/v1/molecules/nav-bar/tenant-switcher';
import { Button } from '@/components/v1/ui/button';
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
import { OutletWithContext, useOutletContext } from '@/lib/router-helpers';
import { cn } from '@/lib/utils';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import useCloudFeatureFlags from '@/pages/auth/hooks/use-cloud-feature-flags';
import { appRoutes } from '@/router';
import {
  CalendarDaysIcon,
  CpuChipIcon,
  PlayIcon,
  ScaleIcon,
  ServerStackIcon,
  Squares2X2Icon,
} from '@heroicons/react/24/outline';
import { ClockIcon, GearIcon } from '@radix-ui/react-icons';
import { Link, useMatchRoute } from '@tanstack/react-router';
import { Filter, SquareActivityIcon, WebhookIcon } from 'lucide-react';
import React, { useCallback } from 'react';

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
      <ThreeColumnLayout
        sidebar={<Sidebar memberships={memberships} />}
        sidePanel={<SidePanel />}
        mainClassName="overflow-auto px-8 py-4"
        mainContainerType="inline-size"
      >
        <OutletWithContext context={childCtx} />
      </ThreeColumnLayout>
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

  return (
    <div
      className={cn(
        // On mobile, overlay the content area (which is already positioned below the fixed header).
        // On desktop, participate in the grid as a fixed-width sidebar.
        'absolute inset-x-0 top-0 bottom-0 z-[100] w-full overflow-hidden border-r bg-slate-100 dark:bg-slate-900 md:relative md:inset-auto md:top-0 md:bottom-auto md:h-full md:w-64 md:min-w-64 md:bg-[unset] md:dark:bg-[unset]',
        className,
      )}
    >
      <div className="flex h-full flex-col overflow-hidden">
        {/* Scrollable navigation area (keep scrollbar flush to sidebar edge) */}
        <div className="min-h-0 flex-1 overflow-auto [scrollbar-gutter:stable] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground">
          <div className="px-4 py-4">
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
                  icon={<PlayIcon className="mr-2 size-4" />}
                />
                <SidebarButtonPrimary
                  key="events"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantEventsRoute.to}
                  params={{ tenant: tenantId }}
                  name="Events"
                  icon={<SquareActivityIcon className="mr-2 size-4" />}
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
                  icon={<CalendarDaysIcon className="mr-2 size-4" />}
                />
                <SidebarButtonPrimary
                  key="crons"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantCronJobsRoute.to}
                  params={{ tenant: tenantId }}
                  name="Cron Jobs"
                  icon={<ClockIcon className="mr-2 size-4" />}
                />
                <SidebarButtonPrimary
                  key="webhooks"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantWebhooksRoute.to}
                  params={{ tenant: tenantId }}
                  name="Webhooks"
                  icon={<WebhookIcon className="mr-2 h-4 w-4" />}
                />
              </div>
            </div>
            <div className="py-2">
              <h2 className="mb-2 text-lg font-semibold tracking-tight">
                Resources
              </h2>
              <div className="space-y-1">
                <SidebarButtonPrimary
                  key="workers"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantWorkersRoute.to}
                  params={{ tenant: tenantId }}
                  name="Workers"
                  icon={<ServerStackIcon className="mr-2 size-4" />}
                />
                <SidebarButtonPrimary
                  key="workflows"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantWorkflowsRoute.to}
                  params={{ tenant: tenantId }}
                  name="Workflows"
                  icon={<Squares2X2Icon className="mr-2 size-4" />}
                />
                {featureFlags?.data['managed-worker'] && (
                  <SidebarButtonPrimary
                    key="managed-compute"
                    onNavLinkClick={onNavLinkClick}
                    to={appRoutes.tenantManagedWorkersRoute.to}
                    params={{ tenant: tenantId }}
                    name="Managed Compute"
                    icon={<CpuChipIcon className="mr-2 size-4" />}
                  />
                )}
                <SidebarButtonPrimary
                  key="rate-limits"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantRateLimitsRoute.to}
                  params={{ tenant: tenantId }}
                  name="Rate Limits"
                  icon={<ScaleIcon className="mr-2 size-4" />}
                />
                <SidebarButtonPrimary
                  key="filters"
                  onNavLinkClick={onNavLinkClick}
                  to={appRoutes.tenantFiltersRoute.to}
                  params={{ tenant: tenantId }}
                  name="Filters"
                  icon={<Filter className="mr-2 size-4" />}
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
                  prefix={appRoutes.tenantSettingsIndexRoute.to}
                  name="General"
                  icon={<GearIcon className="mr-2 size-4" />}
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
        </div>

        {/* Fixed footer: tenant/org picker is always visible and takes up space */}
        <div className="w-full shrink-0 border-t border-slate-200 px-4 py-4 dark:border-slate-800">
          {isCloudEnabled ? (
            <OrganizationSelector memberships={memberships} />
          ) : (
            <TenantSwitcher memberships={memberships} />
          )}
        </div>
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
  const matchRoute = useMatchRoute();

  // `to` (and `prefix`) are TanStack route templates (e.g. `/tenants/$tenant/...`).
  // Use the router matcher instead of raw string comparisons against `location.pathname`.
  const open =
    collapsibleChildren.length > 0
      ? prefix
        ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
        : Boolean(matchRoute({ to, params, fuzzy: true }))
      : false;

  const selected =
    collapsibleChildren.length > 0 ? open : Boolean(matchRoute({ to, params }));

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
      <CollapsibleContent className={'ml-4 space-y-2 border-l border-muted'}>
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
  const matchRoute = useMatchRoute();
  const hasPrefix = prefix
    ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
    : false;
  const selected = Boolean(matchRoute({ to, params })) || hasPrefix;

  return (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          'my-[1px] ml-1 mr-3 w-[calc(100%-3px)] justify-start pl-3 pr-0',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        {name}
      </Button>
    </Link>
  );
}
