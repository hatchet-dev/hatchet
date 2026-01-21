import {
  SideNavSection,
  SideNavChild,
} from '../../../components/v1/nav/side-nav';
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
  RiToolsLine,
  RiPlayLargeLine,
} from 'react-icons/ri';

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
      name: 'GitHub',
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
  ];

  return [
    {
      key: 'overview',
      title: '',
      itemsClassName: 'flex flex-col gap-y-1',
      items: [
        {
          key: 'overview',
          name: 'Overview',
          to: appRoutes.tenantOverviewRoute.to,
          icon: ({
            collapsed,
            active,
          }: {
            collapsed: boolean;
            active?: boolean;
          }) => (
            <svg
              width="17"
              height="17"
              viewBox="0 0 17 17"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            >
              <style>
                {`
                  @keyframes hatchet-fall {
                    0% {
                      transform: translateX(12px) translateY(-10px) rotate(179deg) scale(0.2);
                      opacity: 0;
                    }
                    50% {
                      opacity: 1;
                      transform: translateX(0) translateY(-1px) rotate(20deg) scale(1);
                    }
                    55% {
                      transform: translateY(0) rotate(-12deg);
                    }
                    60% {
                      transform: translateY(0) rotate(0deg);
                      opacity: 1;
                    }
                  }
                  @keyframes stump-gap-animation {
                    0% {
                      fill-opacity: 0.64;
                    }
                    70% {
                      fill-opacity: 0.64;
                    }
                    100% {
                      fill-opacity: 0;
                    }
                  }
                  .hatchet-animated {
                    transform-origin: 8px 7px;
                    animation: hatchet-fall 1s cubic-bezier(0.8, 0.1, 0.8, 0.2) forwards;
                  }
                  .stump-gap-animated {
                    animation: stump-gap-animation 0.5s cubic-bezier(0.8, 0.1, 0.8, 0.2);
                    animation-fill-mode: both;
                  }
                `}
              </style>
              <path
                key={active ? 'hatchet-active' : 'hatchet-inactive'}
                d="M10.7088 5.33333L8.90785 9.33333L5.39489 5.79376C5.00305 5.39895 4.80713 5.20155 4.73366 4.97425C4.66902 4.77429 4.66902 4.55905 4.73366 4.35909C4.80713 4.13178 5.00305 3.93438 5.39489 3.53957L8.90785 7.38316e-07L11.6096 1.21071e-06L7.07243 4.88519C6.94095 5.02675 6.87522 5.09753 6.87203 5.15789C6.86927 5.2103 6.89135 5.26095 6.93164 5.29459C6.97803 5.33333 7.07463 5.33333 7.26782 5.33333L10.7088 5.33333Z"
                className={active ? 'hatchet-animated' : ''}
                fill="currentColor"
              />
              <path
                d="M11.9336 13.3389L5.93359 16.6729L5.28516 15.5068L11.2852 12.1738L11.9336 13.3389ZM12.9424 8C13.6788 8 14.2764 8.59761 14.2764 9.33398V12.5049C14.2765 12.8584 14.417 13.1983 14.667 13.4482L16.5518 15.334H13.1152L11.0166 16.5L10.6924 15.917L10.3691 15.334L12.4678 14.168C12.6657 14.058 12.8888 14.0001 13.1152 14H13.4023C13.1056 13.5618 12.9424 13.0417 12.9424 12.5049V9.33398H10.6846L11.2852 8H12.9424ZM6.86035 9.33398H3.60938V12.5049C3.60931 13.0252 3.4554 13.5294 3.17578 13.959L5.95215 12.418L6.27637 13L6.59961 13.583L3.75098 15.166C3.55304 15.2759 3.32991 15.3339 3.10352 15.334H0L1.1377 14.1953L1.88574 13.4482C2.1357 13.1983 2.27628 12.8584 2.27637 12.5049V9.33398C2.27637 8.59769 2.87311 8.00013 3.60938 8H5.53711L6.86035 9.33398Z"
                fill="currentColor"
                fillOpacity="0.64"
              />
              <path
                d="M10.6846 9.33398L11.2852 8H5.53711L6.86035 9.33398H10.6846Z"
                fill="currentColor"
                fillOpacity="0"
                id="stump-gap"
                className={active ? 'stump-gap-animated' : ''}
              />
            </svg>
          ),
        },
      ],
    },
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
          key: 'tenant-settings',
          name: 'General',
          to: appRoutes.tenantSettingsOverviewRoute.to,
          activeTo: appRoutes.tenantSettingsIndexRoute.to,
          activeFuzzy: true,
          prefix: appRoutes.tenantSettingsIndexRoute.to,
          icon: ({ collapsed }: { collapsed: boolean }) => (
            <RiToolsLine
              className={collapsed ? 'size-5' : 'mr-2 size-4 shrink-0'}
            />
          ),
          children: settingsChildren,
        },
      ],
    },
  ];
}
