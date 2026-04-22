import { SideNavSection } from '../../../components/v1/nav/side-nav';
import { appRoutes } from '@/router';
import {
  RiPulseAiLine,
  RiFilterLine,
  RiCalendarEventLine,
  RiTimeLine,
  RiStackLine,
  RiWebhookLine,
  RiCpuLine,
  RiEqualizer3Line,
  RiFunctionLine,
  RiPlayLargeLine,
  RiFileTextLine,
  RiOrganizationChart,
  RiSunLine,
  RiMoonLine,
  RiSettings3Line,
  RiKey2Line,
  RiTeamLine,
  RiBillLine,
  RiPlugLine,
} from 'react-icons/ri';

export function sideNavItems(opts: {
  canBill?: boolean;
  managedWorkerEnabled?: boolean;
  isCloudEnabled?: boolean;
  onToggleTheme: () => void;
  currentlyVisibleTheme: 'light' | 'dark';
}): SideNavSection[] {
  const billingLabel = opts.canBill ? 'Billing & Limits' : 'Resource Limits';

  return [
    {
      key: 'activity',
      title: 'Activity',
      itemsClassName: 'space-y-1',
      items: [
        {
          key: 'runs',
          name: 'Runs',
          to: appRoutes.tenantRunsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiPlayLargeLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'events',
          name: 'Events',
          to: appRoutes.tenantEventsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiPulseAiLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'logs',
          name: 'Logs',
          to: appRoutes.tenantLogsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiFileTextLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
      ],
    },
    {
      key: 'triggers',
      title: 'Triggers',
      itemsClassName: 'space-y-1',
      items: [
        {
          key: 'scheduled',
          name: 'Scheduled Runs',
          to: appRoutes.tenantScheduledRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiCalendarEventLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'crons',
          name: 'Cron Jobs',
          to: appRoutes.tenantCronJobsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiTimeLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'webhooks',
          name: 'Webhooks',
          to: appRoutes.tenantWebhooksRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiWebhookLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
      ],
    },
    {
      key: 'resources',
      title: 'Resources',
      itemsClassName: 'space-y-1',
      items: [
        {
          key: 'workers',
          name: 'Workers',
          to: appRoutes.tenantWorkersRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiStackLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'workflows',
          name: 'Workflows',
          to: appRoutes.tenantWorkflowsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiFunctionLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        ...(opts.managedWorkerEnabled
          ? [
              {
                key: 'managed-compute',
                name: 'Managed Compute',
                to: appRoutes.tenantManagedWorkersRoute.to,
                icon: ({ collapsed }: { collapsed: boolean }) => (
                  <RiCpuLine
                    className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
                  />
                ),
              },
            ]
          : []),
        {
          key: 'rate-limits',
          name: 'Rate Limits',
          to: appRoutes.tenantRateLimitsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiEqualizer3Line
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'filters',
          name: 'Filters',
          to: appRoutes.tenantFiltersRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiFilterLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
      ],
    },
    {
      key: 'settings',
      title: 'Settings',
      itemsClassName: 'space-y-1',
      items: [
        {
          key: 'tenant-settings-overview',
          name: 'General',
          to: appRoutes.tenantSettingsOverviewRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiSettings3Line
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'organizations',
          name: opts.isCloudEnabled ? 'Organizations' : 'Tenants',
          to: appRoutes.tenantOrganizationsAndTenantsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiOrganizationChart
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'tenant-settings-api-tokens',
          name: 'API Tokens',
          to: appRoutes.tenantSettingsApiTokensRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiKey2Line
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'tenant-settings-members',
          name: 'Members',
          to: appRoutes.tenantSettingsMembersRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiTeamLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'tenant-settings-billing-and-limits',
          name: billingLabel,
          to: appRoutes.tenantSettingsBillingRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiBillLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'tenant-settings-integrations',
          name: 'Integrations',
          to: appRoutes.tenantSettingsIntegrationsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiPlugLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'theme',
          name: `Theme: ${opts.currentlyVisibleTheme === 'dark' ? 'Dark' : 'Light'}`,
          onClick: opts.onToggleTheme,
          icon: ({ collapsed }: { collapsed: boolean }) =>
            opts.currentlyVisibleTheme === 'light' ? (
              <RiSunLine
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ) : (
              <RiMoonLine
                className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
              />
            ),
        },
      ],
    },
  ];
}
