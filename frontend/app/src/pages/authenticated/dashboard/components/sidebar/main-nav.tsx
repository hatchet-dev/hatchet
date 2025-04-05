import {
  AlertTriangle,
  BookOpen,
  Bug,
  Calendar,
  Clock,
  Database,
  Github,
  Key,
  LayoutGrid,
  Play,
  Scale,
  Server,
  Settings,
  Users,
} from 'lucide-react';
import { FaDiscord, FaPlay } from 'react-icons/fa';

export type NavItem = {
  title: string;
  url: string;
  icon: any;
  isActive?: boolean;
  items?: NavItem[];
};

export type SupportItem = {
  title: string;
  url: string;
  icon: any;
};

export type NavSection = {
  label: string;
  items: NavItem[];
};

export type NavStructure = {
  sections: {
    [key: string]: NavSection;
  };
  support: SupportItem[];
  navSecondary: NavItem[];
};

export const getMainNavLinks = (currentPath: string): NavStructure => {
  const isActive = (path: string) => currentPath === path;

  return {
    sections: {
      activity: {
        label: 'Activity',
        items: [
          {
            title: 'Runs',
            url: '/runs',
            icon: Play,
            isActive: isActive('/runs'),
          },
        ],
      },
      triggers: {
        label: 'Triggers',
        items: [
          {
            title: 'Scheduled Runs',
            url: '/scheduled-runs',
            icon: Calendar,
            isActive: isActive('/scheduled-runs'),
          },
          {
            title: 'Cron Jobs',
            url: '/cron-jobs',
            icon: Clock,
            isActive: isActive('/cron-jobs'),
          },
        ],
      },
      resources: {
        label: 'Resources',
        items: [
          {
            title: 'Tasks & Workflows',
            url: '/tasks',
            icon: LayoutGrid,
            isActive: isActive('/tasks'),
          },
          {
            title: 'Workers',
            url: '/workers',
            icon: Server,
            isActive: isActive('/workers'),
          },
          {
            title: 'Rate Limits',
            url: '/rate-limits',
            icon: Scale,
            isActive: isActive('/rate-limits'),
          },
        ],
      },
      settings: {
        label: 'Settings',
        items: [
          {
            title: 'General',
            url: '/settings',
            icon: Settings,
            isActive: isActive('/settings'),
            items: [
              {
                title: 'Overview',
                url: '/settings/overview',
                icon: Settings,
                isActive: isActive('/settings/overview'),
              },
              {
                title: 'API Tokens',
                url: '/settings/api-tokens',
                icon: Key,
                isActive: isActive('/settings/api-tokens'),
              },
              {
                title: 'Github',
                url: '/settings/github',
                icon: Github,
                isActive: isActive('/settings/github'),
              },
              {
                title: 'Members',
                url: '/settings/members',
                icon: Users,
                isActive: isActive('/settings/members'),
              },
              {
                title: 'Resource Limits',
                url: '/settings/resource-limits',
                icon: Scale,
                isActive: isActive('/settings/resource-limits'),
              },
              {
                title: 'Alerting',
                url: '/settings/alerting',
                icon: AlertTriangle,
                isActive: isActive('/settings/alerting'),
              },
              {
                title: 'Ingestors',
                url: '/settings/ingestors',
                icon: Database,
                isActive: isActive('/settings/ingestors'),
              },
            ],
          },
        ],
      },
    },
    support: [
      {
        title: 'Join Our Community',
        url: 'https://hatchet.run/discord',
        icon: FaDiscord,
      },
      {
        title: 'Restart Tutorial',
        url: '/tutorial',
        icon: FaPlay,
      },
    ],
    navSecondary: [
      {
        title: 'Documentation',
        url: 'https://docs.hatchet.run',
        icon: BookOpen,
      },
      {
        title: 'Feedback',
        url: 'https://github.com/hatchet-dev/hatchet/issues',
        icon: Bug,
      },
    ],
  };
};
