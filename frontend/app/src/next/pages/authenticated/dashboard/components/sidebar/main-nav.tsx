import {
  AlertTriangle,
  Bug,
  Calendar,
  Clock,
  Code,
  Cpu,
  Database,
  Github,
  Key,
  Play,
  Scale,
  Settings,
  Users,
  UsersIcon,
  SquareActivity,
} from 'lucide-react';
import { FaDiscord, FaPlay } from 'react-icons/fa';
import { FEATURES_BASE_PATH, ROUTES } from '@/next/lib/routes';

export type NavItem = {
  title: string;
  url: string;
  icon: React.ElementType;
  isActive?: boolean;
  items?: NavItem[];
};

type SupportItem = {
  title: string;
  url: string;
  icon: React.ElementType;
  target?: string;
};

export type NavSection = {
  label: string;
  items: NavItem[];
};

type NavStructure = {
  sections: {
    [key: string]: NavSection;
  };
  support: SupportItem[];
  navSecondary: NavItem[];
};

export const getMainNavLinks = (
  tenantId: string,
  currentPath: string,
): NavStructure => {
  const isActive = (path: string) => currentPath.startsWith(path);

  return {
    sections: {
      activity: {
        label: 'Activity',
        items: [
          {
            title: 'Runs',
            url: ROUTES.runs.list(tenantId),
            icon: Play,
            isActive: isActive(ROUTES.runs.list(tenantId)),
          },
          {
            title: 'Events',
            url: ROUTES.events.list(tenantId),
            icon: SquareActivity,
            isActive: isActive(ROUTES.events.list(tenantId)),
          },
        ],
      },
      triggers: {
        label: 'Triggers',
        items: [
          {
            title: 'Scheduled Runs',
            url: ROUTES.scheduled.list(tenantId),
            icon: Calendar,
            isActive: isActive(ROUTES.scheduled.list(tenantId)),
          },
          {
            title: 'Cron Jobs',
            url: ROUTES.crons.list(tenantId),
            icon: Clock,
            isActive: isActive(ROUTES.crons.list(tenantId)),
          },
        ],
      },
      resources: {
        label: 'Resources',
        items: [
          {
            title: 'Tasks & Workflows',
            url: ROUTES.workflows.list(tenantId),
            icon: Code,
            isActive: isActive(ROUTES.workflows.list(tenantId)),
          },
          {
            title: 'Worker Pools',
            url: ROUTES.workers.list(tenantId),
            icon: Cpu,
            isActive: isActive(ROUTES.workers.list(tenantId)),
          },
          {
            title: 'Rate Limits',
            url: ROUTES.rateLimits.list(tenantId),
            icon: Scale,
            isActive: isActive(ROUTES.rateLimits.list(tenantId)),
          },
        ],
      },
      settings: {
        label: 'Settings',
        items: [
          {
            title: 'API Tokens',
            url: ROUTES.settings.apiTokens(tenantId),
            icon: Key,
            isActive: isActive(ROUTES.settings.apiTokens(tenantId)),
          },
          {
            title: 'Team',
            url: ROUTES.settings.team(tenantId),
            icon: Users,
            isActive: isActive(ROUTES.settings.team(tenantId)),
          },
          {
            title: 'More Settings',
            url: FEATURES_BASE_PATH.settings(tenantId),
            icon: Settings,
            isActive:
              isActive(FEATURES_BASE_PATH.settings(tenantId)) &&
              !isActive(ROUTES.settings.apiTokens(tenantId)) &&
              !isActive(ROUTES.settings.team(tenantId)),
            items: [
              {
                title: 'Tenant Settings',
                url: ROUTES.settings.overview(tenantId),
                icon: UsersIcon,
                isActive: isActive(ROUTES.settings.overview(tenantId)),
              },
              {
                title: 'Github',
                url: ROUTES.settings.github(tenantId),
                icon: Github,
                isActive: isActive(ROUTES.settings.github(tenantId)),
              },
              {
                title: 'Billing & Usage',
                url: ROUTES.settings.usage(tenantId),
                icon: Scale,
                isActive: isActive(ROUTES.settings.usage(tenantId)),
              },
              {
                title: 'Alerting',
                url: ROUTES.settings.alerting(tenantId),
                icon: AlertTriangle,
                isActive: isActive(ROUTES.settings.alerting(tenantId)),
              },
              {
                title: 'Ingestors',
                url: ROUTES.settings.ingestors(tenantId),
                icon: Database,
                isActive: isActive(ROUTES.settings.ingestors(tenantId)),
              },
            ],
          },
        ],
      },
    },
    support: [
      {
        title: 'Join Our Community',
        url: ROUTES.common.community,
        icon: FaDiscord,
        target: '_blank',
      },
      {
        title: 'Restart Tutorial',
        url: ROUTES.learn.firstRun(tenantId),
        icon: FaPlay,
        target: '_self',
      },
    ],
    navSecondary: [
      {
        title: 'Feedback',
        url: ROUTES.common.feedback,
        icon: Bug,
      },
    ],
  };
};
