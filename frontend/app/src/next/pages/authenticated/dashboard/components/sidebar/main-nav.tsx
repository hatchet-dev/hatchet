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

export const getMainNavLinks = (currentPath: string): NavStructure => {
  const isActive = (path: string) => currentPath.startsWith(path);

  return {
    sections: {
      activity: {
        label: 'Activity',
        items: [
          {
            title: 'Runs',
            url: ROUTES.runs.list,
            icon: Play,
            isActive: isActive(ROUTES.runs.list),
          },
        ],
      },
      triggers: {
        label: 'Triggers',
        items: [
          {
            title: 'Scheduled Runs',
            url: ROUTES.scheduled.list,
            icon: Calendar,
            isActive: isActive(ROUTES.scheduled.list),
          },
          {
            title: 'Cron Jobs',
            url: ROUTES.crons.list,
            icon: Clock,
            isActive: isActive(ROUTES.crons.list),
          },
        ],
      },
      resources: {
        label: 'Resources',
        items: [
          {
            title: 'Tasks & Workflows',
            url: ROUTES.workflows.list,
            icon: Code,
            isActive: isActive(ROUTES.workflows.list),
          },
          {
            title: 'Worker Services',
            url: ROUTES.services.list,
            icon: Cpu,
            isActive: isActive(ROUTES.services.list),
          },
          {
            title: 'Rate Limits',
            url: ROUTES.rateLimits.list,
            icon: Scale,
            isActive: isActive(ROUTES.rateLimits.list),
          },
        ],
      },
      settings: {
        label: 'Settings',
        items: [
          {
            title: 'API Tokens',
            url: ROUTES.settings.apiTokens,
            icon: Key,
            isActive: isActive(ROUTES.settings.apiTokens),
          },
          {
            title: 'Team',
            url: ROUTES.settings.team,
            icon: Users,
            isActive: isActive(ROUTES.settings.team),
          },
          {
            title: 'More Settings',
            url: FEATURES_BASE_PATH.settings,
            icon: Settings,
            isActive: isActive(FEATURES_BASE_PATH.settings),
            items: [
              {
                title: 'Tenant Settings',
                url: ROUTES.settings.overview,
                icon: UsersIcon,
                isActive: isActive(ROUTES.settings.overview),
              },
              {
                title: 'Github',
                url: ROUTES.settings.github,
                icon: Github,
                isActive: isActive(ROUTES.settings.github),
              },
              {
                title: 'Billing & Usage',
                url: ROUTES.settings.usage,
                icon: Scale,
                isActive: isActive(ROUTES.settings.usage),
              },
              {
                title: 'Alerting',
                url: ROUTES.settings.alerting,
                icon: AlertTriangle,
                isActive: isActive(ROUTES.settings.alerting),
              },
              {
                title: 'Ingestors',
                url: ROUTES.settings.ingestors,
                icon: Database,
                isActive: isActive(ROUTES.settings.ingestors),
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
      },
      {
        title: 'Restart Tutorial',
        url: ROUTES.common.tutorial,
        icon: FaPlay,
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
