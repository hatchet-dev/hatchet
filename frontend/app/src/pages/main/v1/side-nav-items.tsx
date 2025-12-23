import {
  SideNavSection,
  SideNavChild,
} from '../../../components/v1/nav/side-nav';
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
import { Filter, SquareActivityIcon, WebhookIcon } from 'lucide-react';

export function sideNavItems(opts: {
  canBill?: boolean;
  managedWorkerEnabled?: boolean;
}): SideNavSection[] {
  const billingLabel = opts.canBill ? 'Billing & Limits' : 'Resource Limits';

  const settingsChildren: SideNavChild[] = [
    {
      key: 'tenant-settings-overview',
      name: 'Overview',
      to: appRoutes.tenantSettingsOverviewRoute.to,
    },
    {
      key: 'tenant-settings-api-tokens',
      name: 'API Tokens',
      to: appRoutes.tenantSettingsApiTokensRoute.to,
    },
    {
      key: 'tenant-settings-github',
      name: 'Github',
      to: appRoutes.tenantSettingsGithubRoute.to,
    },
    {
      key: 'tenant-settings-members',
      name: 'Members',
      to: appRoutes.tenantSettingsMembersRoute.to,
    },
    {
      key: 'tenant-settings-billing-and-limits',
      name: billingLabel,
      to: appRoutes.tenantSettingsBillingRoute.to,
    },
    {
      key: 'tenant-settings-alerting',
      name: 'Alerting',
      to: appRoutes.tenantSettingsAlertingRoute.to,
    },
    {
      key: 'tenant-settings-ingestors',
      name: 'Ingestors',
      to: appRoutes.tenantSettingsIngestorsRoute.to,
    },
    {
      key: 'quickstart',
      name: 'Quickstart',
      to: appRoutes.tenantOnboardingGetStartedRoute.to,
    },
  ];

  return [
    {
      key: 'activity',
      title: 'Activity',
      itemsClassName: 'flex flex-col gap-y-1',
      items: [
        {
          key: 'runs',
          name: 'Runs',
          to: appRoutes.tenantRunsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <PlayIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'events',
          name: 'Events',
          to: appRoutes.tenantEventsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <SquareActivityIcon
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
            <CalendarDaysIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'crons',
          name: 'Cron Jobs',
          to: appRoutes.tenantCronJobsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <ClockIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'webhooks',
          name: 'Webhooks',
          to: appRoutes.tenantWebhooksRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <WebhookIcon
              className={collapsed ? 'size-5' : 'mr-2 h-4 w-4 shrink-0'}
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
            <ServerStackIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'workflows',
          name: 'Workflows',
          to: appRoutes.tenantWorkflowsRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <Squares2X2Icon
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
                  <CpuChipIcon
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
            <ScaleIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
        },
        {
          key: 'filters',
          name: 'Filters',
          to: appRoutes.tenantFiltersRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <Filter className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'} />
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
          key: 'tenant-settings',
          name: 'General',
          to: appRoutes.tenantSettingsOverviewRoute.to,
          activeTo: appRoutes.tenantSettingsIndexRoute.to,
          activeFuzzy: true,
          prefix: appRoutes.tenantSettingsIndexRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <GearIcon
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
          children: settingsChildren,
        },
      ],
    },
  ];
}
